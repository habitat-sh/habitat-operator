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

package e2e

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestServiceGroupCreate tests service group creation.
func TestServiceGroupCreate(t *testing.T) {
	sgName := "test-standalone"
	sg := framework.NewStandaloneSG(sgName, "foobar", false)

	if err := framework.CreateSG(sg); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(sgName, 1); err != nil {
		t.Fatal(err)
	}

	_, err := framework.KubeClient.CoreV1().ConfigMaps(apiv1.NamespaceDefault).Get(sgName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

// TestServiceGroupInitialConfig tests initial configuration.
func TestServiceGroupInitialConfig(t *testing.T) {
	sgName := "mytutorialapp"
	msg := "Hello from Tests!"
	configMsg := fmt.Sprintf("message = '%s'", msg)

	// Create Secret as a user would.
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: sgName,
		},
		Data: map[string][]byte{
			"user.toml": []byte(configMsg),
		},
	}

	_, err := framework.KubeClient.CoreV1().Secrets(apiv1.NamespaceDefault).Create(secret)
	if err != nil {
		t.Fatal(err)
	}

	sg := framework.NewStandaloneSG(sgName, "foobar", true)

	if err := framework.CreateSG(sg); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(sgName, 1); err != nil {
		t.Fatal(err)
	}

	// Create Kubernetes Service to expose port.
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: sgName,
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				crv1.ServiceGroupLabel: sgName,
			},
			Type: "NodePort",
			Ports: []apiv1.ServicePort{
				apiv1.ServicePort{
					Name:     "web",
					NodePort: 30003,
					Port:     5555,
				},
			},
		},
	}

	// Create Service.
	_, err = framework.KubeClient.CoreV1().Services(apiv1.NamespaceDefault).Create(service)
	if err != nil {
		t.Fatal(err)
	}

	// Wait until endpoints are ready.
	if err := framework.WaitForEndpoints(sgName); err != nil {
		t.Fatal(err)
	}

	// Get response from Habitat Service.
	url := fmt.Sprintf("http://%s:30003/", framework.ExternalIP)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatal("We did not get a 200 OK from the deployed Habitat Service.")
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// Check if the msg we supplied was picked up by the Habitat Service.
	actualMsg := string(bodyBytes)
	if msg != actualMsg {
		t.Fatalf("Initial Configuration failed. Msg did not match the one expected. Expected: %s got: %s", msg, actualMsg)
	}
}

// TestServiceGroupFunctioning tests that operator deploys a habitat service and that it has started.
func TestServiceGroupFunctioning(t *testing.T) {
	sgName := "test-service-group"
	sg := framework.NewStandaloneSG(sgName, "foobar", false)

	if err := framework.CreateSG(sg); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(sgName, 1); err != nil {
		t.Fatal(err)
	}

	// Create Kubernetes Service to expose port.
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: sgName,
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				crv1.ServiceGroupLabel: sgName,
			},
			Type: "NodePort",
			Ports: []apiv1.ServicePort{
				apiv1.ServicePort{
					Name:     "web",
					NodePort: 30002,
					Port:     5555,
				},
			},
		},
	}
	// Create Service.
	_, err := framework.KubeClient.CoreV1().Services(apiv1.NamespaceDefault).Create(service)
	if err != nil {
		t.Fatal(err)
	}

	// Wait until endpoints are ready.
	if err := framework.WaitForEndpoints(sgName); err != nil {
		t.Fatal(err)
	}

	// Get response from Habitat Service.
	url := fmt.Sprintf("http://%s:30002/", framework.ExternalIP)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Habitat Service did not start correctly.")
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// This msg is set in the default.toml in kinvolk/nodejs-hab Habitat Service.
	expectedMsg := "Hello, World!"
	actualMsg := string(bodyBytes)
	if expectedMsg != actualMsg {
		t.Fatalf("Habitat Service msg does not match one in default.toml. Expected: %s got: %s", expectedMsg, actualMsg)
	}
}

// TestServiceGroupDelete tests Service Group deletion.
func TestServiceGroupDelete(t *testing.T) {
	sgName := "test-deletion"
	sg := framework.NewStandaloneSG(sgName, "foobar", false)

	if err := framework.CreateSG(sg); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(sgName, 1); err != nil {
		t.Fatal(err)
	}

	// Delete SG.
	if err := framework.DeleteSG(sgName); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be deleted.
	if err := framework.WaitForResources(sgName, 0); err != nil {
		t.Fatal(err)
	}

	// Check if all the resources the operator creates are deleted.
	// We do not care about secrets being deleted, as the user needs to delete those manually.
	d, err := framework.KubeClient.AppsV1beta1().Deployments(apiv1.NamespaceDefault).Get(sgName, metav1.GetOptions{})
	if err == nil && d != nil {
		t.Fatal("Deployment was not deleted.")
	}

	cm, err := framework.KubeClient.CoreV1().ConfigMaps(apiv1.NamespaceDefault).Get(sgName, metav1.GetOptions{})
	if err == nil && cm != nil {
		t.Fatal("ConfigMap was not deleted.")
	}
}
