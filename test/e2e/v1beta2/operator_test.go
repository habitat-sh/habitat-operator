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

package v1beta2

import (
	"fmt"
	"strings"
	"testing"
	"time"

	habv1beta1 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta1"
	utils "github.com/habitat-sh/habitat-operator/test/e2e/v1beta2/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	serviceStartupWaitTime = 1 * time.Minute
	secretUpdateTimeout    = 2 * time.Minute
	secretUpdateQueryTime  = 10 * time.Second

	configMapName = "peer-watch-file"
)

// TestBind tests that the operator correctly created two Habitat Services and bound them together.
func TestBind(t *testing.T) {
	// Get Habitat object from Habitat go example.
	web, err := utils.ConvertHabitat("resources/bind-config/webapp.yml")
	if err != nil {
		t.Fatal(err)
	}

	if err := framework.CreateHabitat(web); err != nil {
		t.Fatal(err)
	}

	// Get Habitat object from Habitat db example.
	db, err := utils.ConvertHabitat("resources/bind-config/db.yml")
	if err != nil {
		t.Fatal(err)
	}

	if err := framework.CreateHabitat(db); err != nil {
		t.Fatal(err)
	}

	// Get Service object from example file.
	svc, err := utils.ConvertService("resources/bind-config/service.yml")
	if err != nil {
		t.Fatal(err)
	}

	// Create Service.
	_, err = framework.KubeClient.CoreV1().Services(utils.TestNs).Create(svc)
	if err != nil {
		t.Fatal(err)
	}
	// Delete Service so it doesn't interfere with other tests.
	defer (func(name string) {
		if err := framework.DeleteService(name); err != nil {
			t.Fatal(err)
		}
	})(svc.Name)

	// Get Secret object from example file.
	sec, err := utils.ConvertSecret("resources/bind-config/secret.yml")
	if err != nil {
		t.Fatal(err)
	}

	// Create Secret.
	sec, err = framework.KubeClient.CoreV1().Secrets(utils.TestNs).Create(sec)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be ready.
	if err := framework.WaitForResources(habv1beta1.HabitatNameLabel, web.ObjectMeta.Name, 1); err != nil {
		t.Fatal(err)
	}
	if err := framework.WaitForResources(habv1beta1.HabitatNameLabel, db.ObjectMeta.Name, 1); err != nil {
		t.Fatal(err)
	}

	// Wait until endpoints are ready.
	if err := framework.WaitForEndpoints(svc.ObjectMeta.Name); err != nil {
		t.Fatal(err)
	}

	time.Sleep(serviceStartupWaitTime)

	// Get response from Habitat Service.
	url := fmt.Sprintf("http://%s:30001/", framework.ExternalIP)

	body, err := utils.QueryService(url)
	if err != nil {
		t.Fatal(err)
	}

	// This msg is set in the config of the habitat/bindgo-hab Go Habitat Service.
	expectedMsg := "hello from port: 4444"
	actualMsg := body
	// actualMsg can contain whitespace and newlines or different formatting,
	// the only thing we need to check is it contains the expectedMsg.
	if !strings.Contains(actualMsg, expectedMsg) {
		t.Fatalf("Habitat Service msg does not match one in default.toml. Expected: \"%s\", got: \"%s\"", expectedMsg, actualMsg)
	}

	// Test `user.toml` updates.

	// Update secret.
	newPort := "port = 6333"

	sec.Data["user.toml"] = []byte(newPort)
	if _, err = framework.KubeClient.CoreV1().Secrets(utils.TestNs).Update(sec); err != nil {
		t.Fatalf("Could not update Secret: \"%s\"", err)
	}

	ticker := time.NewTicker(secretUpdateQueryTime)
	defer ticker.Stop()
	timer := time.NewTimer(secretUpdateTimeout)
	defer timer.Stop()
	// Update the message set in the config of the habitat/bindgo-hab Go Habitat Service.
	expectedMsg = fmt.Sprintf("hello from port: %v", 6333)
	for {
		// Check that the port differs after the update.
		actualMsg, err := utils.QueryService(url)
		if err != nil {
			t.Fatal(err)
		}

		// actualMsg can contain whitespace and newlines or different formatting,
		// the only thing we need to check is it contains the expectedMsg.
		if strings.Contains(actualMsg, expectedMsg) {
			break
		}

		fail := func() {
			t.Fatalf("Configuration update did not go through. Expected: \"%s\", got: \"%s\"", expectedMsg, actualMsg)
		}

		select {
		case <-timer.C:
			fail()
		case <-ticker.C:
			// This is to avoid infinite loops when go
			// decides to always pick the ticker channel,
			// even when timer channel is ready too.
			select {
			case <-timer.C:
				fail()
			default:
			}
		}
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
	if err := framework.WaitForResources(habv1beta1.HabitatNameLabel, habitat.ObjectMeta.Name, 1); err != nil {
		t.Fatal(err)
	}

	// Delete Habitat.
	if err := framework.DeleteHabitat(habitat.ObjectMeta.Name); err != nil {
		t.Fatal(err)
	}

	// Wait for resources to be deleted.
	if err := framework.WaitForResources(habv1beta1.HabitatNameLabel, habitat.ObjectMeta.Name, 0); err != nil {
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

func TestPersistentStorage(t *testing.T) {
	// We run minikube in a VM on Travis. In that environment, we cannot create PersistentVolumes.
	t.Skip("This test cannot be run successfully in our current testing setup")

	ephemeral, err := utils.ConvertHabitat("resources/standalone/habitat.yml")
	if err != nil {
		t.Fatal(err)
	}

	persisted, err := utils.ConvertHabitat("resources/persisted/habitat.yml")
	if err != nil {
		t.Fatal(err)
	}

	if err := framework.CreateHabitat(ephemeral); err != nil {
		t.Fatal(err)
	}

	if err := framework.CreateHabitat(persisted); err != nil {
		t.Fatal(err)
	}

	// Delete all PVCs at the end of the test.
	// For dynamically provisioned PVs (as is the case on minikube), this will
	// also delete the PVs.
	defer (func(name string) {
		ls := labels.SelectorFromSet(labels.Set(map[string]string{
			habv1beta1.HabitatNameLabel: name,
		}))

		lo := metav1.ListOptions{
			LabelSelector: ls.String(),
		}

		err := framework.KubeClient.CoreV1().PersistentVolumeClaims(utils.TestNs).DeleteCollection(&metav1.DeleteOptions{}, lo)
		if err != nil {
			t.Fatal(err)
		}
	})(persisted.Name)

	if err := framework.WaitForResources(habv1beta1.HabitatNameLabel, persisted.Name, 1); err != nil {
		t.Fatal(err)
	}

	// Test that persistence is only enabled if requested
	ephemeralSTS, err := framework.KubeClient.AppsV1beta1().StatefulSets(utils.TestNs).Get(ephemeral.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(ephemeralSTS.Spec.VolumeClaimTemplates) != 0 {
		t.Fatal("PersistentVolumeClaims created for ephemeral StatefulSet")
	}

	persistedSTS, err := framework.KubeClient.AppsV1beta1().StatefulSets(utils.TestNs).Get(persisted.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(persistedSTS.Spec.VolumeClaimTemplates) == 0 {
		t.Fatal("No PersistentVolumeClaims created for persistent StatefulSet")
	}
}
