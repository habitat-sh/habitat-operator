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

	habv1beta1 "github.com/kinvolk/habitat-operator/pkg/apis/habitat/v1beta1"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// CreateHabitat creates a Habitat.
func (f *Framework) CreateHabitat(habitat *habv1beta1.Habitat) error {
	return f.Client.Post().
		Namespace(TestNs).
		Resource(habv1beta1.HabitatResourcePlural).
		Body(habitat).
		Do().
		Error()
}

// WaitForResources waits until numPods are in the "Running" state.
// We wait for pods, because those take the longest to create.
// Waiting for anything else would be already testing.
func (f *Framework) WaitForResources(labelName, habitatName string, numPods int) error {
	return wait.Poll(2*time.Second, 5*time.Minute, func() (bool, error) {
		fs := fields.SelectorFromSet(fields.Set{
			"status.phase": "Running",
		})

		ls := labels.SelectorFromSet(labels.Set{
			labelName: habitatName,
		})

		pods, err := f.KubeClient.CoreV1().Pods(TestNs).List(metav1.ListOptions{FieldSelector: fs.String(), LabelSelector: ls.String()})
		if err != nil {
			return false, err
		}

		if len(pods.Items) != numPods {
			return false, nil
		}

		return true, nil
	})
}

func (f *Framework) WaitForEndpoints(habitatName string) error {
	return wait.Poll(time.Second, time.Minute*5, func() (bool, error) {
		ep, err := f.KubeClient.CoreV1().Endpoints(TestNs).Get(habitatName, metav1.GetOptions{})
		if err != nil {
			return false, err
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

// DeleteHabitat deletes a Habitat as a user would.
func (f *Framework) DeleteHabitat(habitatName string) error {
	return f.Client.Delete().
		Namespace(TestNs).
		Resource(habv1beta1.HabitatResourcePlural).
		Name(habitatName).
		Do().
		Error()
}

// DeleteService delete a Kubernetes service provided.
func (f *Framework) DeleteService(service string) error {
	return f.KubeClient.CoreV1().Services(TestNs).Delete(service, &metav1.DeleteOptions{})
}

func (f *Framework) createRBAC() error {
	// Create Service account.
	sa, err := convertServiceAccount("resources/operator/service-account.yml")
	if err != nil {
		return err
	}
	_, err = f.KubeClient.CoreV1().ServiceAccounts(TestNs).Create(sa)
	if err != nil {
		return err
	}

	// Create cluster role.
	cr, err := convertClusterRole("resources/operator/cluster-role.yml")
	if err != nil {
		return err
	}
	_, err = f.KubeClient.RbacV1().ClusterRoles().Create(cr)
	if err != nil {
		return err
	}

	// Create cluster role bindings.
	crb, err := convertClusterRoleBinding("resources/operator/cluster-role-binding.yml")
	if err != nil {
		return err
	}
	_, err = f.KubeClient.RbacV1().ClusterRoleBindings().Create(crb)
	if err != nil {
		return err
	}

	return nil
}

// convertServiceAccount takes in a path to the YAML file containing the manifest
// It converts it from that file to the ServiceAccount object.
func convertServiceAccount(pathToYaml string) (*apiv1.ServiceAccount, error) {
	sa := apiv1.ServiceAccount{}

	if err := convertToK8sResource(pathToYaml, &sa); err != nil {
		return nil, err
	}

	return &sa, nil
}

// convertClusterRole takes in a path to the YAML file containing the manifest.
// It converts the file to the ClusterRole object.
func convertClusterRole(pathToYaml string) (*rbacv1.ClusterRole, error) {
	cr := rbacv1.ClusterRole{}

	if err := convertToK8sResource(pathToYaml, &cr); err != nil {
		return nil, err
	}

	return &cr, nil
}

// convertClusterRoleBinding takes in a path to the YAML file containing the manifest.
// It converts the file to the ClusterRoleBinding object.
func convertClusterRoleBinding(pathToYaml string) (*rbacv1.ClusterRoleBinding, error) {
	crb := rbacv1.ClusterRoleBinding{}

	if err := convertToK8sResource(pathToYaml, &crb); err != nil {
		return nil, err
	}

	return &crb, nil
}

// ConvertDeployment takes in a path to the YAML file containing the manifest.
// It converts the file to the Deployment object.
func ConvertDeployment(pathToYaml string) (*appsv1beta1.Deployment, error) {
	d := appsv1beta1.Deployment{}

	if err := convertToK8sResource(pathToYaml, &d); err != nil {
		return nil, err
	}

	return &d, nil
}

// ConvertHabitat takes in a path to the YAML file containing the manifest.
// It converts the file to the Habitat object.
func ConvertHabitat(pathToYaml string) (*habv1beta1.Habitat, error) {
	hab := habv1beta1.Habitat{}

	if err := convertToK8sResource(pathToYaml, &hab); err != nil {
		return nil, err
	}

	return &hab, nil
}

// ConvertService takes in a path to the YAML file containing the manifest.
// It converts the file to the Service object.
func ConvertService(pathToYaml string) (*v1.Service, error) {
	s := v1.Service{}

	if err := convertToK8sResource(pathToYaml, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

// ConvertSecret takes in a path to the YAML file containing the manifest.
// It converts the file to the Secret object.
func ConvertSecret(pathToYaml string) (*v1.Secret, error) {
	s := v1.Secret{}

	if err := convertToK8sResource(pathToYaml, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

func convertToK8sResource(pathToYaml string, into interface{}) error {
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
		return nil, err
	}

	manifest, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

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
