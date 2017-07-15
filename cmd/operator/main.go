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

package main

import (
	"context"
	"flag"
	"os"

	"github.com/go-kit/kit/log"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	habitatclient "github.com/kinvolk/habitat-operator/pkg/habitat/client"
	habitatcontroller "github.com/kinvolk/habitat-operator/pkg/habitat/controller"
)

type Config struct {
	Client kubernetes.Interface
}

func run() int {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))

	// Parse config flags.
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.Parse()

	// Build operator config.
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		logger.Log("error", err)
		return 1
	}

	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		logger.Log("error", err)
		return 1
	}

	// Create ServiceGroup CRD.
	crd, err := habitatclient.CreateCRD(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		logger.Log("error", err)
		return 1
	}

	logger.Log("info", "created ServiceGroup CRD")

	defer func() {
		apiextensionsclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, nil)
		logger.Log("info", "deleted ServiceGroup CRD")
	}()

	client, scheme, err := habitatclient.NewClient(config)
	if err != nil {
		logger.Log("error", err)
		return 1
	}

	controller := habitatcontroller.HabitatController{
		HabitatClient: client,
		HabitatScheme: scheme,
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go controller.Run(ctx)

	habitatclient.WaitForServiceGroupInstanceProcessed(client, "sg1")
	if err != nil {
		logger.Log("error", err)
		return 1
	}

	logger.Log("info", "exiting habitat-operator")
	return 0
}

func main() {
	os.Exit(run())
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	return rest.InClusterConfig()
}
