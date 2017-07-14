package main

import (
	"flag"
	"os"

	"github.com/go-kit/kit/log"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	Client kubernetes.Interface
}

func run() int {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))

	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.Parse()

	config, err := buildConfig(*kubeconfig)
	if err != nil {
		logger.Log("error", err)
		return 1
	}

	if _, err := apiextensionsclient.NewForConfig(config); err != nil {
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
