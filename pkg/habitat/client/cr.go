// Copyright (c) 2017 Chef Software Inc. and/or applicable contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"fmt"
	"reflect"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"
	apiv1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

const (
	habitatCRDName                    = crv1.HabitatResourcePlural + "." + crv1.GroupName
	habitatPromotionCRDName           = crv1.HabitatPromotionResourcePlural + "." + crv1.GroupName
	habitatResourceShortName          = "hab"
	habitatPromotionResourceShortName = "habprom"

	pollInterval = 500 * time.Millisecond
	timeOut      = 10 * time.Second
)

// createCRD creates the Custom Resource Definition with the given name.
// It checks if creation has completed successfully, and deletes the CRD in case of error.
func createCRD(clientset apiextensionsclient.Interface, logger log.Logger, crdName, plural, shortName string, kind interface{}) error {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   crv1.GroupName,
			Version: crv1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural:     plural,
				Kind:       reflect.TypeOf(kind).Name(),
				ShortNames: []string{shortName},
			},
		},
	}

	if _, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
		level.Info(logger).Log("msg", fmt.Sprintf("%s CRD already exists, continuing", crdName))
	}

	// wait for CRD being established.
	if err := wait.Poll(pollInterval, timeOut, func() (bool, error) {
		var err error

		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, metav1.GetOptions{})

		if err != nil {
			return false, err
		}

		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					level.Error(logger).Log("msg", fmt.Sprintf("Error: Name conflict: %v\n", cond.Reason))
				}
			}
		}

		return false, err
	}); err != nil {
		// delete CRD if there was an error.
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crdName, nil)
		if deleteErr != nil {
			return errors.NewAggregate([]error{err, deleteErr})
		}

		return err
	}

	return nil
}

func CreateCRDs(clientset apiextensionsclient.Interface, logger log.Logger) error {
	// Create Habitat CRD.
	if err := createCRD(clientset, logger, habitatCRDName, crv1.HabitatResourcePlural, habitatResourceShortName, crv1.Habitat{}); err != nil {
		return err
	}

	// Create HabitatPromotion CRD.
	if err := createCRD(clientset, logger, habitatPromotionCRDName, crv1.HabitatPromotionResourcePlural, habitatPromotionResourceShortName, crv1.HabitatPromotion{}); err != nil {
		return err
	}

	return nil
}

// WaitForHabitatInstanceProcessed polls the API for a specific Habitat with a state of "Processed".
func WaitForHabitatInstanceProcessed(client *rest.RESTClient, name string) error {
	return wait.Poll(100*time.Millisecond, 10*time.Second, func() (bool, error) {
		var hab crv1.Habitat
		err := client.Get().
			Resource(crv1.HabitatResourcePlural).
			Namespace(apiv1.NamespaceDefault).
			Name(name).
			Do().Into(&hab)

		if err == nil && hab.Status.State == crv1.HabitatStateProcessed {
			return true, nil
		}

		return false, err
	})
}
