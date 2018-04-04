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

package v1beta1

import (
	"fmt"
	"reflect"
	"time"

	"github.com/habitat-sh/habitat-operator/pkg/apis/habitat"
	habv1beta1 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta1"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

const (
	leaderFollowerTopologyMinCount = 3
	pollInterval                   = 500 * time.Millisecond
	timeOut                        = 10 * time.Second
	habitatCRDName                 = habv1beta1.HabitatResourcePlural + "." + habitat.GroupName
)

type keyNotFoundError struct {
	key string
}

func (err keyNotFoundError) Error() string {
	return fmt.Sprintf("could not find Object with key %s in the cache", err.key)
}

func validateCustomObject(h habv1beta1.Habitat) error {
	spec := h.Spec

	switch spec.Service.Topology {
	case habv1beta1.TopologyStandalone:
	case habv1beta1.TopologyLeader:
		if spec.Count < leaderFollowerTopologyMinCount {
			return fmt.Errorf("too few instances: %d, leader-follower topology requires at least %d", spec.Count, leaderFollowerTopologyMinCount)
		}
	default:
		return fmt.Errorf("unknown topology: %s", spec.Service.Topology)
	}

	if rsn := spec.Service.RingSecretName; rsn != "" {
		ringParts := ringRegexp.FindStringSubmatch(rsn)

		// The ringParts slice should have a second element for the capturing group
		// in the ringRegexp regular expression, containing the ring's name.
		if len(ringParts) < 2 {
			return fmt.Errorf("malformed ring secret name: %s", rsn)
		}
	}

	return nil
}

// newListWatchFromClientWithLabels is a modified newListWatchFromClient function from listWatch.
// Instead of using fields to filter, we modify the function to use labels.
func newListWatchFromClientWithLabels(c cache.Getter, resource string, namespace string, op metav1.ListOptions) *cache.ListWatch {
	listFunc := func(_ metav1.ListOptions) (runtime.Object, error) {
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&op, metav1.ParameterCodec).
			Do().
			Get()
	}
	watchFunc := func(_ metav1.ListOptions) (watch.Interface, error) {
		op.Watch = true
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&op, metav1.ParameterCodec).
			Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func labelListOptions() metav1.ListOptions {
	ls := labels.SelectorFromSet(labels.Set(map[string]string{
		habv1beta1.HabitatLabel: "true",
	}))

	return metav1.ListOptions{
		LabelSelector: ls.String(),
	}
}

func CreateCRD(clientset apiextensionsclient.Interface) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	name := habv1beta1.Kind(habv1beta1.HabitatResourcePlural)

	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name.String(),
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   habv1beta1.SchemeGroupVersion.Group,
			Version: habv1beta1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural:     habv1beta1.HabitatResourcePlural,
				Kind:       reflect.TypeOf(habv1beta1.Habitat{}).Name(),
				ShortNames: []string{habv1beta1.HabitatShortName},
			},
		},
	}

	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return nil, err
	}

	// wait for CRD being established.
	err = wait.Poll(pollInterval, timeOut, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(habitatCRDName, metav1.GetOptions{})

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
					// TODO re-introduce logging?
					// fmt.Printf("Error: Name conflict: %v\n", cond.Reason)
				}
			}
		}

		return false, err
	})

	// delete CRD if there was an error.
	if err != nil {
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(habitatCRDName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}

		return nil, err
	}

	return crd, nil
}
