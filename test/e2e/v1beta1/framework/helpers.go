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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	habv1beta1 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta1"
	habclient "github.com/habitat-sh/habitat-operator/pkg/client/clientset/versioned/typed/habitat/v1beta1"

	"github.com/pkg/errors"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Framework struct {
	Image               string
	KubeClient          kubernetes.Interface
	APIExtensionsClient apiextensionsclient.Interface
	Client              *habclient.HabitatV1beta1Client
	ExternalIP          string
	Namespace           string
}

// Setup sets up the test Framework object by creating essential clients
// needed to talk to the Kubernetes API server.
func Setup(image, kubeconfig, externalIP, namespace string) (*Framework, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "config building from kubeconfig failed")
	}

	apiclientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "create kubernetes api clientset failed")
	}

	apiextensionsClientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "create kubernetes api extension clientset failed")
	}

	cl, err := habclient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "create habitat client failed")
	}

	f := &Framework{
		Image:               image,
		KubeClient:          apiclientset,
		APIExtensionsClient: apiextensionsClientset,
		Client:              cl,
		ExternalIP:          externalIP,
		Namespace:           namespace,
	}

	// Create a new Kubernetes namespace for testing purpose.
	_, err = f.KubeClient.CoreV1().Namespaces().Create(&apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "create namespace failed")
	}
	return f, nil
}

// CreateHabitat creates a Habitat resource in cluster.
func (f *Framework) CreateHabitat(habitat *habv1beta1.Habitat) error {
	if _, err := f.Client.Habitats(f.Namespace).Create(habitat); err != nil {
		return errors.Wrap(err, "create Habitat failed")
	}
	return nil
}

// WaitForResources waits until numPods are in the "Running" state.
// We wait for pods, because those take the longest to create.
// Waiting for anything else would be already testing.
func (f *Framework) WaitForResources(labelName, habitatName string, numPods int) error {
	return wait.Poll(2*time.Second, 5*time.Minute, func() (bool, error) {
		fs := fields.SelectorFromSet(fields.Set{
			"status.phase": string(apiv1.NodeRunning),
		})

		ls := labels.SelectorFromSet(labels.Set{
			labelName: habitatName,
		})

		pods, err := f.KubeClient.CoreV1().Pods(f.Namespace).List(
			metav1.ListOptions{
				FieldSelector: fs.String(),
				LabelSelector: ls.String(),
			},
		)
		if err != nil {
			return false, errors.Wrap(err, "list Pod failed")
		}

		if len(pods.Items) != numPods {
			return false, nil
		}

		return true, nil
	})
}

func (f *Framework) WaitForEndpoints(habitatName string) error {
	return wait.Poll(time.Second, time.Minute*5, func() (bool, error) {
		ep, err := f.KubeClient.CoreV1().Endpoints(f.Namespace).Get(habitatName, metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrap(err, "get Endpoints failed")
		}

		if len(ep.Subsets) == 0 {
			return false, nil
		}

		if len(ep.Subsets[0].Addresses) == 0 {
			return false, nil
		}

		return true, nil
	})
}

// GetLoadBalancerIP waits for Load Balancer IP to become available and returns it
func (f *Framework) GetLoadBalancerIP(serviceName string) (string, error) {
	loadBalancerIP := ""
	err := wait.Poll(2*time.Second, 5*time.Minute, func() (bool, error) {
		service, err := f.KubeClient.Core().Services(f.Namespace).Get(serviceName, metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrap(err, "get Services failed")
		}

		if len(service.Status.LoadBalancer.Ingress) == 0 {
			return false, nil
		}
		loadBalancerIP = service.Status.LoadBalancer.Ingress[0].IP
		return true, nil
	})
	return loadBalancerIP, errors.Wrap(err, "wait poll failed")
}

// DeleteHabitat deletes a Habitat as a user would.
func (f *Framework) DeleteHabitat(habitatName string, ns string) error {
	return f.Client.Habitats(ns).Delete(habitatName, nil)
}

// DeleteService delete a Kubernetes service provided.
func (f *Framework) DeleteService(service string) error {
	return f.KubeClient.CoreV1().Services(f.Namespace).Delete(service, &metav1.DeleteOptions{})
}

// DeleteNamespace deletes the namespace that is stored in the
// Framework object while it's initialization
func (f *Framework) DeleteNamespace() error {
	return f.KubeClient.CoreV1().Namespaces().Delete(
		f.Namespace,
		&metav1.DeleteOptions{},
	)
}

// DeleteCRD deletes the CRD with given name
func (f *Framework) DeleteCRD(name string) error {
	return f.APIExtensionsClient.Apiextensions().CustomResourceDefinitions().Delete(
		name,
		&metav1.DeleteOptions{},
	)
}

// ConvertServiceAccount takes in a path to the YAML file containing the manifest
// It converts it from that file to the ServiceAccount object.
func ConvertServiceAccount(pathToYaml string) (*apiv1.ServiceAccount, error) {
	sa := apiv1.ServiceAccount{}

	if err := convertToK8sResource(pathToYaml, &sa); err != nil {
		return nil, errors.Wrap(err, "convert yml file to Kubernetes resource failed")
	}

	return &sa, nil
}

// ConvertClusterRole takes in a path to the YAML file containing the manifest.
// It converts the file to the ClusterRole object.
func ConvertClusterRole(pathToYaml string) (*rbacv1.ClusterRole, error) {
	cr := rbacv1.ClusterRole{}

	if err := convertToK8sResource(pathToYaml, &cr); err != nil {
		return nil, errors.Wrap(err, "convert yml file to Kubernetes resource failed")
	}

	return &cr, nil
}

// ConvertClusterRoleBinding takes in a path to the YAML file containing the manifest.
// It converts the file to the ClusterRoleBinding object.
func ConvertClusterRoleBinding(pathToYaml string) (*rbacv1.ClusterRoleBinding, error) {
	crb := rbacv1.ClusterRoleBinding{}

	if err := convertToK8sResource(pathToYaml, &crb); err != nil {
		return nil, errors.Wrap(err, "convert yml file to Kubernetes resource failed")
	}

	return &crb, nil
}

// ConvertRole takes in a path to the YAML file containing the manifest.
// It converts the file to the Role object.
func ConvertRole(pathToYaml string) (*rbacv1.Role, error) {
	role := &rbacv1.Role{}

	if err := convertToK8sResource(pathToYaml, role); err != nil {
		return nil, errors.Wrap(err, "convert yml file to Kubernetes resource failed")
	}

	return role, nil
}

// ConvertRoleBinding takes in a path to the YAML file containing the manifest.
// It converts the file to the RoleBinding object.
func ConvertRoleBinding(pathToYaml string) (*rbacv1.RoleBinding, error) {
	rb := &rbacv1.RoleBinding{}

	if err := convertToK8sResource(pathToYaml, rb); err != nil {
		return nil, errors.Wrap(err, "convert yml file to Kubernetes resource failed")
	}

	return rb, nil
}

// ConvertCRD takes in a path to the YAML file containing the manifest.
// It converts the file to the CustomResourceDefinition object.
func ConvertCRD(pathToYaml string) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{}

	if err := convertToK8sResource(pathToYaml, crd); err != nil {
		return nil, errors.Wrap(err, "convert yml file to Kubernetes resource failed")
	}

	return crd, nil
}

// ConvertDeployment takes in a path to the YAML file containing the manifest.
// It converts the file to the Deployment object.
func ConvertDeployment(pathToYaml string) (*appsv1beta1.Deployment, error) {
	d := appsv1beta1.Deployment{}

	if err := convertToK8sResource(pathToYaml, &d); err != nil {
		return nil, errors.Wrap(err, "convert yml file to Kubernetes resource failed")
	}

	return &d, nil
}

// ConvertHabitat takes in a path to the YAML file containing the manifest.
// It converts the file to the Habitat object.
func ConvertHabitat(pathToYaml string) (*habv1beta1.Habitat, error) {
	hab := habv1beta1.Habitat{}

	if err := convertToK8sResource(pathToYaml, &hab); err != nil {
		return nil, errors.Wrap(err, "convert yml file to Kubernetes resource failed")
	}

	return &hab, nil
}

// ConvertService takes in a path to the YAML file containing the manifest.
// It converts the file to the Service object.
func ConvertService(pathToYaml string) (*v1.Service, error) {
	s := v1.Service{}

	if err := convertToK8sResource(pathToYaml, &s); err != nil {
		return nil, errors.Wrap(err, "convert yml file to Kubernetes resource failed")
	}

	return &s, nil
}

// ConvertSecret takes in a path to the YAML file containing the manifest.
// It converts the file to the Secret object.
func ConvertSecret(pathToYaml string) (*v1.Secret, error) {
	s := v1.Secret{}

	if err := convertToK8sResource(pathToYaml, &s); err != nil {
		return nil, errors.Wrap(err, "convert yml file to Kubernetes resource failed")
	}

	return &s, nil
}

func convertToK8sResource(pathToYaml string, into runtime.Object) error {
	manifest, err := pathToOSFile(pathToYaml)
	if err != nil {
		return err
	}

	if err := yaml.NewYAMLToJSONDecoder(manifest).Decode(into); err != nil {
		return err
	}

	return nil
}

// pathToOSFile takes in a path and converts it to a File.
func pathToOSFile(relativePath string) (*os.File, error) {
	path, err := filepath.Abs(relativePath)
	if err != nil {
		return nil, errors.Wrap(err, "finding absolute path failed")
	}

	manifest, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "open file failed: %s", path)
	}

	return manifest, nil
}

// QueryService makes an HTTP GET request to `url` and returns the body.
func QueryService(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Habitat Service did not start correctly.")
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}
