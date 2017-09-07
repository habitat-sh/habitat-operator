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

package framework

import (
	"time"

	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

// NewStandaloneSG returns a new Standalone ServiceGroup.
func (f *Framework) NewStandaloneSG(sgName, group string, secret bool) *crv1.ServiceGroup {
	sg := crv1.ServiceGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: sgName,
		},
		Spec: crv1.ServiceGroupSpec{
			Image: "kinvolk/nodejs-hab",
			Count: 1,
			Habitat: crv1.Habitat{
				Group:    group,
				Topology: crv1.TopologyStandalone,
			},
		},
	}

	if secret {
		sg.Spec.Habitat.Config = sgName
	}
	return &sg
}

// CreateSG creates a ServiceGroup.
func (f *Framework) CreateSG(sg *crv1.ServiceGroup) error {
	return f.Client.Post().
		Namespace(apiv1.NamespaceDefault).
		Resource(crv1.ServiceGroupResourcePlural).
		Body(sg).
		Do().
		Error()
}

// WaitForResources waits until numPods are in the "Running" state.
// We wait for pods, because those take the longest to create.
// Waiting for anything else would be already testing.
func (f *Framework) WaitForResources(sgName string, numPods int) error {
	return wait.Poll(2*time.Second, 5*time.Minute, func() (bool, error) {
		fs := fields.SelectorFromSet(fields.Set{
			"status.phase": "Running",
		})

		ls := labels.SelectorFromSet(labels.Set{
			crv1.ServiceGroupLabel: sgName,
		})

		pods, err := f.KubeClient.CoreV1().Pods(apiv1.NamespaceDefault).List(metav1.ListOptions{FieldSelector: fs.String(), LabelSelector: ls.String()})
		if err != nil {
			return false, err
		}

		if len(pods.Items) != numPods {
			return false, nil
		}

		return true, nil
	})
}

func (f *Framework) WaitForEndpoints(sgName string) error {
	return wait.Poll(time.Second, time.Minute*5, func() (bool, error) {
		ep, err := f.KubeClient.CoreV1().Endpoints(apiv1.NamespaceDefault).Get(sgName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if len(ep.Subsets) == 0 && len(ep.Subsets[0].Addresses) == 0 {
			return false, nil
		}

		return true, nil
	})
}

// DeleteSG deletes a ServiceGroup as a user would.
func (f *Framework) DeleteSG(sgName string) error {
	return f.Client.Delete().
		Namespace(apiv1.NamespaceDefault).
		Resource(crv1.ServiceGroupResourcePlural).
		Name(sgName).
		Do().
		Error()
}

func (f *Framework) DeleteService(service string) error {
	return f.KubeClient.CoreV1().Services(apiv1.NamespaceDefault).Delete(service, &metav1.DeleteOptions{})
}
