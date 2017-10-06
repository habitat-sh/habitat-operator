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
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	flag "github.com/spf13/pflag"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	habitatclient "github.com/kinvolk/habitat-operator/pkg/habitat/client"
	habitatcontroller "github.com/kinvolk/habitat-operator/pkg/habitat/controller"
)

type Config struct {
	Client kubernetes.Interface
}

func run() int {
	// Parse config flags.
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	verbose := flag.BoolP("verbose", "v", false, "Enable verbose logging.")
	flag.Parse()

	// Set up logging.
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestamp)

	if *verbose {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	// Build operator config.
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	// This is the clientset for interacting with the apiextensions group.
	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	// Create Habitat CRD.
	_, crdErr := habitatclient.CreateCRD(apiextensionsclientset)
	if crdErr != nil {
		if !apierrors.IsAlreadyExists(crdErr) {
			level.Error(logger).Log("msg", crdErr)
			return 1
		}

		level.Info(logger).Log("msg", "Habitat CRD already exists, continuing")
	} else {
		level.Info(logger).Log("msg", "created Habitat CRD")
	}

	habitatClient, scheme, err := habitatclient.NewClient(config)
	if err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	// This is the clientset for interacting with the stable API group.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	controllerConfig := habitatcontroller.Config{
		HabitatClient:       habitatClient,
		KubernetesClientset: clientset,
		Scheme:              scheme,
	}
	hc, err := habitatcontroller.New(controllerConfig, log.With(logger, "component", "controller"))
	if err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go hc.Run(ctx)

	term := make(chan os.Signal)
	// Relay these signals to the `term` channel.
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-term:
		level.Info(logger).Log("msg", "received SIGTERM, exiting gracefully...")
	case <-ctx.Done():
		level.Info(logger).Log("msg", "context channel closed, exiting")
	}

	return 0
}

func main() {
	os.Exit(run())
}
