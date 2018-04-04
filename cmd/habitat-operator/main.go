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
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	habclientset "github.com/habitat-sh/habitat-operator/pkg/client/clientset/versioned"
	habinformers "github.com/habitat-sh/habitat-operator/pkg/client/informers/externalversions"
	habv1beta1controller "github.com/habitat-sh/habitat-operator/pkg/controller/v1beta1"
	habv1beta2controller "github.com/habitat-sh/habitat-operator/pkg/controller/v1beta2"
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

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClientset, time.Second*30)
	habInformerFactory := habinformers.NewSharedInformerFactory(habClientset, time.Second*30)

	controllerConfig := habv1beta2controller.Config{
		HabitatClient:          habClientset.HabitatV1beta2().RESTClient(),
		KubernetesClientset:    kubeClientset,
		ClusterConfig:          config,
		KubeInformerFactory:    kubeInformerFactory,
		HabitatInformerFactory: habInformerFactory,
	}
	hc, err := habv1beta2controller.New(controllerConfig, log.With(logger, "component", "controller"))
	if err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	beta1Config := habv1beta1controller.Config{
		HabitatClient:       habClientset.HabitatV1beta1().RESTClient(),
		KubernetesClientset: kubeClientset,
	}
	beta1Controller, err := habv1beta1controller.New(beta1Config, log.With(logger, "component", "controller"))
	if err != nil {
		level.Error(logger).Log("msg", err)
		return 1
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go hc.Run(runtime.NumCPU(), ctx)
	go beta1Controller.Run(runtime.NumCPU(), ctx)

	go kubeInformerFactory.Start(ctx.Done())
	go habInformerFactory.Start(ctx.Done())

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
