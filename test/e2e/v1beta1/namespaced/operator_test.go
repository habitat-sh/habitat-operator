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

package namespaced

import (
	"testing"

	habv1beta1 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta1"
	utils "github.com/habitat-sh/habitat-operator/test/e2e/v1beta1/framework"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespacedStandalone(t *testing.T) {
	// Create the habitat operator
	h, err := utils.ConvertHabitat("resources/standalone/habitat.yml")
	if err != nil {
		t.Fatal(errors.Wrap(err, "convert Habitat from yml file failed"))
	}
	h.Namespace = framework.Namespace
	if err := framework.CreateHabitat(h); err != nil {
		t.Fatal(err)
	}

	if err := framework.WaitForResources(habv1beta1.HabitatNameLabel, h.Name, 1); err != nil {
		t.Fatal(errors.Wrap(err, "wait for Resorces failed"))
	}

	// check that the `StatefulSet` has been created
	if _, err := framework.KubeClient.AppsV1().StatefulSets(h.Namespace).Get(
		h.Name,
		metav1.GetOptions{},
	); err != nil {
		t.Fatalf("StatefulSet not created by the operator: %v", err)
	}
}
