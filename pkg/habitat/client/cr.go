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

	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"
	apiv1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

const (
	serviceGroupCRDName           = crv1.ServiceGroupResourcePlural + "." + crv1.GroupName
	serviceGroupResourceShortName = "sg"

	pollInterval = 500 * time.Millisecond
	timeOut      = 10 * time.Second
)

// CreateCRD creates the ServiceGroup Custom Resource Definition.
// It checks if creation has completed successfully, and deletes the CRD in case of error.
func CreateCRD(clientset apiextensionsclient.Interface) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceGroupCRDName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   crv1.GroupName,
			Version: crv1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural:     crv1.ServiceGroupResourcePlural,
				Kind:       reflect.TypeOf(crv1.ServiceGroup{}).Name(),
				ShortNames: []string{serviceGroupResourceShortName},
			},
		},
	}

	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return nil, err
	}

	// wait for CRD being established.
	err = wait.Poll(pollInterval, timeOut, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(serviceGroupCRDName, metav1.GetOptions{})

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
					fmt.Printf("Error: Name conflict: %v\n", cond.Reason)
				}
			}
		}

		return false, err
	})

	// delete CRD if there was an error.
	if err != nil {
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(serviceGroupCRDName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}

		return nil, err
	}

	return crd, nil
}

// WaitForServiceGroupInstanceProcessed polls the API for a specific ServiceGroup with a state of "Processed".
func WaitForServiceGroupInstanceProcessed(client *rest.RESTClient, name string) error {
	return wait.Poll(100*time.Millisecond, 10*time.Second, func() (bool, error) {
		var sg crv1.ServiceGroup
		err := client.Get().
			Resource(crv1.ServiceGroupResourcePlural).
			Namespace(apiv1.NamespaceDefault).
			Name(name).
			Do().Into(&sg)

		if err == nil && sg.Status.State == crv1.ServiceGroupStateProcessed {
			return true, nil
		}

		return false, err
	})
}
