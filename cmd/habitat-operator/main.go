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
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	habclientset "github.com/habitat-sh/habitat-operator/pkg/client/clientset/versioned"
	habinformers "github.com/habitat-sh/habitat-operator/pkg/client/informers/externalversions"
	habv1beta2controller "github.com/habitat-sh/habitat-operator/pkg/controller/v1beta2"
	"github.com/habitat-sh/habitat-operator/pkg/version"
)

const resyncPeriod = 30 * time.Second

type Clientsets struct {
	KubeClientset          *kubernetes.Clientset
	HabClientset           *habclientset.Clientset
	ApiextensionsClientset *apiextensionsclient.Clientset
}

// FlagOpts struct is used to save all the flag values for operator
type FlagOpts struct {
	Namespace           string
	AssumeCRDRegistered bool
}

func run() int {
	// Parse config flags.
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	verbose := flag.Bool("verbose", false, "Enable verbose logging.")
	namespace := flag.String("namespace", metav1.NamespaceAll, "Specify namespace this Operator will be monitoring. (default: Monitors all namespaces)")
	assumeCRDRegistered := flag.Bool("assume-crd-registered", false, "If cluster admin has already registered CRD then provide this flag with namespace flag.")
	flag.Parse()

	// Set up logging.
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestamp)

	if *verbose {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	flags := &FlagOpts{
		Namespace:           *namespace,
		AssumeCRDRegistered: *assumeCRDRegistered,
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

	// check if the operator has right permissions when trying to run cluster wide
	// here since namespace is not provided so we are looking at all the namespaces
	// Operator should have permission to query all the namespaces
	if flags.Namespace == metav1.NamespaceAll {
		level.Info(logger).Log("msg", "Running operator at cluster scope, looking for all the namespaces")
		if _, err := kubeClientset.CoreV1().Namespaces().List(metav1.ListOptions{}); err != nil {
			level.Error(logger).Log("msg", errors.Wrap(err, "Operator does not have cluster wide permissions"))
			return 1
		}
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	var wg sync.WaitGroup
	wg.Add(1)

	cSets := Clientsets{
		KubeClientset:          kubeClientset,
		HabClientset:           habClientset,
		ApiextensionsClientset: apiextensionsClientset,
	}

	if err := v1beta2(ctx, &wg, cSets, logger, flags); err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	term := make(chan os.Signal, 2)
	// Relay these signals to the `term` channel.
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-term
		level.Info(logger).Log("msg", "received termination signal, exiting gracefully...")
		cancelFunc()

		<-term
		os.Exit(1)
	}()

	<-ctx.Done()

	// Block until the WaitGroup counter is zero
	wg.Wait()

	level.Info(logger).Log("msg", "controllers stopped, exiting")

	return 0
}

// createCRD creates Habitat CRD in the cluster, provided the operator has 'create'
// permission on apiextensions.k8s.io/CustomResourceDefinitions type
// if it does not then this fails, logs information about the existing CRD.
func createCRD(cSets Clientsets, logger log.Logger) error {
	if _, err := habv1beta2controller.CreateCRD(cSets.ApiextensionsClientset); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "create Habitat CRD failed")
		}
		level.Info(logger).Log("msg", "Habitat CRD already exists, continuing")
		return nil
	}
	level.Info(logger).Log("msg", "created Habitat CRD")
	return nil
}

func v1beta2(ctx context.Context, wg *sync.WaitGroup, cSets Clientsets, logger log.Logger, flags *FlagOpts) error {
	// if user has already created CRD in the cluster with help of cluster-admin
	// then operator does not need to create CRD
	if !flags.AssumeCRDRegistered {
		if err := createCRD(cSets, logger); err != nil {
			return err
		}
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(
		cSets.KubeClientset,
		resyncPeriod,
		kubeinformers.WithNamespace(flags.Namespace),
	)
	habInformerFactory := habinformers.NewSharedInformerFactoryWithOptions(
		cSets.HabClientset,
		resyncPeriod,
		habinformers.WithNamespace(flags.Namespace),
	)

	config := habv1beta2controller.Config{
		// NOTE: The v1beta2 controller still needs to use a v1beta1 client,
		// because we _have_ only one client.  This is due to the fact that it's
		// not currently possible to have multiple versions of a CRD (and
		// therefore, of a client), running at the same time
		HabitatClient:          cSets.HabClientset.HabitatV1beta1().RESTClient(),
		KubernetesClientset:    cSets.KubeClientset,
		KubeInformerFactory:    kubeInformerFactory,
		HabitatInformerFactory: habInformerFactory,
		Namespace:              flags.Namespace,
	}
	controller, err := habv1beta2controller.New(config, log.With(logger, "component", "controller/v1beta2"))
	if err != nil {
		return err
	}

	var factoriesWg sync.WaitGroup
	factoriesWg.Add(2)

	go func() {
		kubeInformerFactory.Start(ctx.Done())
		factoriesWg.Done()
	}()

	go func() {
		habInformerFactory.Start(ctx.Done())
		factoriesWg.Done()
	}()

	go func() {
		controller.Run(ctx, runtime.NumCPU())
		factoriesWg.Wait()
		wg.Done()
	}()

	return nil
}

func printVersion() {
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Operator Version: %s\n", version.VERSION)
}

func main() {
	printVersion()
	os.Exit(run())
}
