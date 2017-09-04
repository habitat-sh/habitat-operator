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

// Package framework sets up the test framework needed to run the end-to-end
// tests on Kubernetes.
package framework

import (
	habclient "github.com/kinvolk/habitat-operator/pkg/habitat/client"

	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Framework struct {
	Image      string
	KubeClient kubernetes.Interface
	Client     *rest.RESTClient
	ExternalIP string
}

// Setup sets up the test framework.
func Setup(image, kubeconfig, externalIP string) (*Framework, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	apiclientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cl, _, err := habclient.NewClient(config)
	if err != nil {
		return nil, err
	}

	f := &Framework{
		Image:      image,
		KubeClient: apiclientset,
		Client:     cl,
		ExternalIP: externalIP,
	}

	if err = f.setupOperator(); err != nil {
		return nil, err
	}

	return f, nil
}

func (f *Framework) setupOperator() error {
	name := "habitat-operator"
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				crv1.ServiceGroupLabel: name,
			},
		},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{
					Name:  name,
					Image: f.Image,
				},
			},
		},
	}

	// Create pod with the Habitat operator image.
	_, err := f.KubeClient.CoreV1().Pods(apiv1.NamespaceDefault).Create(pod)
	if err != nil {
		return err
	}

	// Wait until the operator is ready.
	f.WaitForResources(name, 1)

	return nil
}
