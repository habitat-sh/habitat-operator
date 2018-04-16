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
	"runtime"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	flag "github.com/spf13/pflag"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	habclientset "github.com/habitat-sh/habitat-operator/pkg/client/clientset/versioned"
	habinformers "github.com/habitat-sh/habitat-operator/pkg/client/informers/externalversions"
	habv1beta1controller "github.com/habitat-sh/habitat-operator/pkg/controller/v1beta1"
	habv1beta2controller "github.com/habitat-sh/habitat-operator/pkg/controller/v1beta2"
)

const resyncPeriod = 30 * time.Second

type Clientsets struct {
	KubeClientset          *kubernetes.Clientset
	HabClientset           *habclientset.Clientset
	ApiextensionsClientset *apiextensionsclient.Clientset
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
	apiextensionsClientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	// This is the clientset for interacting with the Habitat API.
	habClientset, err := habclientset.NewForConfig(config)
	if err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	// This is the clientset for interacting with the stable API group.
	kubeClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	cSets := Clientsets{
		KubeClientset:          kubeClientset,
		HabClientset:           habClientset,
		ApiextensionsClientset: apiextensionsClientset,
	}

	if err := v1beta1(ctx, cSets, logger); err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	if err := v1beta2(ctx, cSets, logger); err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

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

func v1beta1(ctx context.Context, cSets Clientsets, logger log.Logger) error {
	// Create Habitat CRD.
	_, err := habv1beta1controller.CreateCRD(cSets.ApiextensionsClientset)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}

		level.Info(logger).Log("msg", "Habitat CRD already exists, continuing")
	} else {
		level.Info(logger).Log("msg", "created Habitat CRD")
	}

	config := habv1beta1controller.Config{
		HabitatClient:       cSets.HabClientset.HabitatV1beta1().RESTClient(),
		KubernetesClientset: cSets.KubeClientset,
	}
	controller, err := habv1beta1controller.New(config, log.With(logger, "component", "controller/v1beta1"))
	if err != nil {
		return err
	}

	go controller.Run(runtime.NumCPU(), ctx)

	return nil
}

func v1beta2(ctx context.Context, cSets Clientsets, logger log.Logger) error {
	// Create Habitat CRD.
	_, err := habv1beta2controller.CreateCRD(cSets.ApiextensionsClientset)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}

		level.Info(logger).Log("msg", "Habitat CRD already exists, continuing")
	} else {
		level.Info(logger).Log("msg", "created Habitat CRD")
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(cSets.KubeClientset, resyncPeriod)
	habInformerFactory := habinformers.NewSharedInformerFactory(cSets.HabClientset, resyncPeriod)

	config := habv1beta2controller.Config{
		HabitatClient:          cSets.HabClientset.HabitatV1beta1().RESTClient(),
		KubernetesClientset:    cSets.KubeClientset,
		KubeInformerFactory:    kubeInformerFactory,
		HabitatInformerFactory: habInformerFactory,
	}
	controller, err := habv1beta2controller.New(config, log.With(logger, "component", "controller/v1beta2"))
	if err != nil {
		return err
	}

	go kubeInformerFactory.Start(ctx.Done())
	go habInformerFactory.Start(ctx.Done())

	go controller.Run(runtime.NumCPU(), ctx)

	return nil
}

func main() {
	os.Exit(run())
}
