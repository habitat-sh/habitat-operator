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
	habv1 "github.com/kinvolk/habitat-operator/pkg/apis/habitat/v1"
	habclient "github.com/kinvolk/habitat-operator/pkg/client"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	TestNs = "testing"
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

	// Create a new Kubernetes namespace for testing purposes.
	_, err = f.KubeClient.CoreV1().Namespaces().Create(&apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: TestNs,
		},
	})
	if err != nil {
		return nil, err
	}

	if err = f.setupOperator(); err != nil {
		return nil, err
	}

	return f, nil
}

func (f *Framework) setupOperator() error {
	// Setup RBAC for operator.
	err := f.createRBAC()
	if err != nil {
		return err
	}

	// Get Habitat operator deployment from examples.
	d, err := ConvertDeployment("resources/operator/habitat.yml")
	if err != nil {
		return err
	}

	// Override image with the one passed to the tests.
	d.Spec.Template.Spec.Containers[0].Image = f.Image

	// Create deployment for the Habitat operator.
	_, err = f.KubeClient.AppsV1beta1().Deployments(TestNs).Create(d)
	if err != nil {
		return err
	}

	// Wait until the operator is ready.
	f.WaitForResources(name, 1)

	return nil
}
