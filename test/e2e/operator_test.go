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

	habv1 "github.com/kinvolk/habitat-operator/pkg/apis/habitat/v1"
	utils "github.com/kinvolk/habitat-operator/test/e2e/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitForPorts  = 1 * time.Minute
	configMapName = "peer-watch-file"

	nodejsImage = "kinvolk/nodejs-hab:test"
)

// TestFunction tests that the operator correctly created two Habitat Services and bound them together.
func TestFunction(t *testing.T) {
	// Get Habitat object from Habitat go example.
	habitatGo, err := utils.ConvertHabitat("resources/bind-config/habitat-go.yml")
	if err != nil {
		t.Fatal(err)
	}

	if err := framework.CreateHabitat(habitatGo); err != nil {
		t.Fatal(err)
	}

	// Get Habitat object from Habitat db example.
	habitatDB, err := utils.ConvertHabitat("resources/bind-config/habitat-postgresql.yml")
	if err != nil {
		t.Fatal(err)
	}

	if err := framework.CreateHabitat(habitatDB); err != nil {
		t.Fatal(err)
	}

	// Get Service object from example file.
	service, err := utils.ConvertService("resources/bind-config/service.yml")
	if err != nil {
		t.Fatal(err)
	}

	// Create Service.
	_, err = framework.KubeClient.CoreV1().Services(utils.TestNs).Create(service)
	if err != nil {
		t.Fatal(err)
	}

	// Get Secret object from example file.
	secret, err := utils.ConvertSecret("resources/bind-config/secret.yml")
	if err != nil {
		t.Fatal(err)
	}

	// Create Secret.
	_, err = framework.KubeClient.CoreV1().Secrets(utils.TestNs).Create(secret)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(habv1.HabitatNameLabel, habitatGo.ObjectMeta.Name, 1); err != nil {
		t.Fatal(err)
	}
	if err := framework.WaitForResources(habv1.HabitatNameLabel, habitatDB.ObjectMeta.Name, 1); err != nil {
		t.Fatal(err)
	}

	// Wait until endpoints are ready.
	if err := framework.WaitForEndpoints(service.ObjectMeta.Name); err != nil {
		t.Fatal(err)
	}

	time.Sleep(waitForPorts)

	// Get response from Habitat Service.
	url := fmt.Sprintf("http://%s:30001/", framework.ExternalIP)
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
	expectedMsg := "hello from port: 4444"
	actualMsg := string(bodyBytes)
	// actualMsg can contain whitespace and newlines or different formatting,
	// the only thing we need to check is it contains the expectedMsg.
	if !strings.Contains(actualMsg, expectedMsg) {
		t.Fatalf("Habitat Service msg does not match one in default.toml. Expected: *%s* got: *%s*", expectedMsg, actualMsg)
	}

	// Delete Service so it doesn't interfere with other tests.
	if err := framework.DeleteService(service.ObjectMeta.Name); err != nil {
		t.Fatal(err)
	}
}

// TestHabitatDelete tests Habitat deletion.
func TestHabitatDelete(t *testing.T) {
	// Get Habitat object from Habitat go example.
	habitat, err := utils.ConvertHabitat("resources/standalone/habitat.yml")
	if err != nil {
		t.Fatal(err)
	}

	if err := framework.CreateHabitat(habitat); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(habv1.HabitatNameLabel, habitat.ObjectMeta.Name, 1); err != nil {
		t.Fatal(err)
	}

	// Delete Habitat.
	if err := framework.DeleteHabitat(habitat.ObjectMeta.Name); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be deleted.
	if err := framework.WaitForResources(habv1.HabitatNameLabel, habitat.ObjectMeta.Name, 0); err != nil {
		t.Fatal(err)
	}

	// Check if all the resources the operator creates are deleted.
	// We do not care about secrets being deleted, as the user needs to delete those manually.
	d, err := framework.KubeClient.AppsV1beta1().Deployments(utils.TestNs).Get(habitat.ObjectMeta.Name, metav1.GetOptions{})
	if err == nil && d != nil {
		t.Fatal("Deployment was not deleted.")
	}

	// The CM with the peer IP should still be alive, despite the Habitat being deleted as it was created outside of the scope of a Habitat.
	_, err = framework.KubeClient.CoreV1().ConfigMaps(utils.TestNs).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
}
