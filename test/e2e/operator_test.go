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
	"strings"
	"testing"
	"time"

	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"
	utils "github.com/kinvolk/habitat-operator/test/e2e/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	waitForPorts  = 1 * time.Minute
	configMapName = "peer-watch-file"

	nodejsImage = "kinvolk/nodejs-hab:test"
)

// TestHabitatCreate tests Habitat creation.
func TestHabitatCreate(t *testing.T) {
	habitatName := "test-standalone"
	habitat := utils.NewStandaloneHabitat(habitatName, "foobar", nodejsImage)

	if err := framework.CreateHabitat(habitat); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(habitatName, 1); err != nil {
		t.Fatal(err)
	}

	_, err := framework.KubeClient.CoreV1().ConfigMaps(utils.TestNs).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

// TestHabitatInitialConfig tests initial configuration.
func TestHabitatInitialConfig(t *testing.T) {
	habitatName := "mytutorialapp"
	msg := "Hello from Tests!"
	configMsg := fmt.Sprintf("message = '%s'", msg)

	// Create Secret as a user would.
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: habitatName,
		},
		Data: map[string][]byte{
			"user.toml": []byte(configMsg),
		},
	}

	_, err := framework.KubeClient.CoreV1().Secrets(utils.TestNs).Create(secret)
	if err != nil {
		t.Fatal(err)
	}

	habitat := utils.NewStandaloneHabitat(habitatName, "foobar", nodejsImage)
	utils.AddConfigToHabitat(habitat)

	if err := framework.CreateHabitat(habitat); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(habitatName, 1); err != nil {
		t.Fatal(err)
	}

	// Create Kubernetes Service to expose port.
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: habitatName,
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				crv1.HabitatNameLabel: habitatName,
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
	_, err = framework.KubeClient.CoreV1().Services(utils.TestNs).Create(service)
	if err != nil {
		t.Fatal(err)
	}

	// Wait until endpoints are ready.
	if err := framework.WaitForEndpoints(habitatName); err != nil {
		t.Fatal(err)
	}
	time.Sleep(waitForPorts)

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

	// Delete Service so it doesn't interfere with other tests.
	if err := framework.DeleteService(habitatName); err != nil {
		t.Fatal(err)
	}
}

// TestHabitatFunctioning tests that operator deploys a Habitat service and that it has started.
func TestHabitatFunctioning(t *testing.T) {
	habitatName := "test-habitat"
	habitat := utils.NewStandaloneHabitat(habitatName, "foobar", nodejsImage)

	if err := framework.CreateHabitat(habitat); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(habitatName, 1); err != nil {
		t.Fatal(err)
	}

	// Create Kubernetes Service to expose port.
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: habitatName,
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				crv1.HabitatNameLabel: habitatName,
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
	_, err := framework.KubeClient.CoreV1().Services(utils.TestNs).Create(service)
	if err != nil {
		t.Fatal(err)
	}

	// Wait until endpoints are ready.
	if err := framework.WaitForEndpoints(habitatName); err != nil {
		t.Fatal(err)
	}
	time.Sleep(waitForPorts)

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

	// Delete Service so it doesn't interfere with other tests.
	if err := framework.DeleteService(habitatName); err != nil {
		t.Fatal(err)
	}
}

// TestHabitatDelete tests Habitat deletion.
func TestHabitatDelete(t *testing.T) {
	habitatName := "test-deletion"
	habitat := utils.NewStandaloneHabitat(habitatName, "foobar", nodejsImage)

	if err := framework.CreateHabitat(habitat); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(habitatName, 1); err != nil {
		t.Fatal(err)
	}

	// Delete Habitat.
	if err := framework.DeleteHabitat(habitatName); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be deleted.
	if err := framework.WaitForResources(habitatName, 0); err != nil {
		t.Fatal(err)
	}

	// Check if all the resources the operator creates are deleted.
	// We do not care about secrets being deleted, as the user needs to delete those manually.
	d, err := framework.KubeClient.AppsV1beta1().Deployments(utils.TestNs).Get(habitatName, metav1.GetOptions{})
	if err == nil && d != nil {
		t.Fatal("Deployment was not deleted.")
	}

	// The CM with the peer IP should still be alive, despite the Habitat being deleted as it was created outside of the scope of a Habitat.
	_, err = framework.KubeClient.CoreV1().ConfigMaps(utils.TestNs).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

// TestBind tests that the operator correctly created two Habitat Services and bound them together.
func TestBind(t *testing.T) {
	// Create two Habitat to test binding between them.
	habitatGoName := "test-bind-go"
	bindName := "db"
	habitatGo := utils.NewStandaloneHabitat(habitatGoName, "foobar", "kinvolk/bindgo-hab:test")
	utils.AddBindToHabitat(habitatGo, bindName, "postgresql")

	if err := framework.CreateHabitat(habitatGo); err != nil {
		t.Fatal(err)
	}

	habitatDBName := "test-bind-db"
	habitatDB := utils.NewStandaloneHabitat(habitatDBName, "foobar", "kinvolk/postgresql-hab:test")

	if err := framework.CreateHabitat(habitatDB); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(habitatGoName, 1); err != nil {
		t.Fatal(err)
	}
	if err := framework.WaitForResources(habitatDBName, 1); err != nil {
		t.Fatal(err)
	}

	// Create Kubernetes Service to expose port.
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: habitatGoName,
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				crv1.HabitatNameLabel: habitatGoName,
			},
			Type: "NodePort",
			Ports: []apiv1.ServicePort{
				apiv1.ServicePort{
					Name:     "go",
					NodePort: 30005,
					Port:     5555,
				},
			},
		},
	}

	// Create Service.
	_, err := framework.KubeClient.CoreV1().Services(utils.TestNs).Create(service)
	if err != nil {
		t.Fatal(err)
	}

	// Wait until endpoints are ready.
	if err := framework.WaitForEndpoints(habitatGoName); err != nil {
		t.Fatal(err)
	}
	time.Sleep(waitForPorts)

	// Get response from Habitat Service.
	url := fmt.Sprintf("http://%s:30005/", framework.ExternalIP)
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

	// This msg is set in the config of the kinvolk/bindgo-hab Go Habitat Service.
	expectedMsg := "hello from port: 5432"
	actualMsg := string(bodyBytes)
	// actualMsg can contain whitespace and newlines or different formatting,
	// the only thing we need to check is it contains the expectedMsg.
	if !strings.Contains(actualMsg, expectedMsg) {
		t.Fatalf("Habitat Service msg does not match one in default.toml. Expected: *%s* got: *%s*", expectedMsg, actualMsg)
	}

	// Delete Service so it doesn't interfere with other tests.
	if err := framework.DeleteService(habitatGoName); err != nil {
		t.Fatal(err)
	}
}
