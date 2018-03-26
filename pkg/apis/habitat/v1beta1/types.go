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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	HabitatResourcePlural = "habitats"

	// HabitatLabel labels the resources that belong to Habitat.
	// Example: 'habitat: true'
	HabitatLabel = "habitat"
	// HabitatNameLabel contains the user defined Habitat Service name.
	// Example: 'habitat-name: db'
	HabitatNameLabel = "habitat-name"

	TopologyLabel = "topology"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Habitat struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              HabitatSpec   `json:"spec"`
	Status            HabitatStatus `json:"status,omitempty"`
}

type HabitatSpec struct {
	// Count is the amount of Services to start in this Habitat.
	Count int `json:"count"`
	// Image is the Docker image of the Habitat Service.
	Image   string  `json:"image"`
	Service Service `json:"service"`
	// Env is a list of environment variables.
	// The EnvVar type is documented at https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#envvar-v1-core.
	// Optional.
	Env []corev1.EnvVar `json:"env,omitempty"`
}

type HabitatStatus struct {
	State   HabitatState `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
}

type HabitatState string

type Service struct {
	// Group is the value of the --group flag for the hab client.
	// Optional. Defaults to `default`.
	Group string `json:"group"`
	// Topology is the value of the --topology flag for the hab client.
	Topology `json:"topology"`
	// ConfigSecretName is the name of a Secret containing a Habitat service's config in TOML format.
	// It will be mounted inside the pod as a file, and it will be used by Habitat to configure the service.
	// Optional.
	ConfigSecretName string `json:"configSecretName,omitempty"`
	// The name of the secret that contains the ring key.
	// Optional.
	RingSecretName string `json:"ringSecretName,omitempty"`
	// Bind is when one service connects to another forming a producer/consumer relationship.
	// Optional.
	Bind []Bind `json:"bind,omitempty"`
	// Name is the name of the Habitat service that this Habitat object represents.
	// This field is used to mount the user.toml file in the correct directory under /hab/user/ in the Pod.
	Name string `json:"name"`
}

type Bind struct {
	// Name is the name of the bind specified in the Habitat configuration files.
	Name string `json:"name"`
	// Service is the name of the service this bind refers to.
	Service string `json:"service"`
	// Group is the group of the service this bind refers to.
	Group string `json:"group"`
}

type Topology string

func (t Topology) String() string {
	return string(t)
}

const (
	HabitatStateCreated   HabitatState = "Created"
	HabitatStateProcessed HabitatState = "Processed"

	TopologyStandalone Topology = "standalone"
	TopologyLeader     Topology = "leader"

	HabitatKind = "Habitat"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type HabitatList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Habitat `json:"items"`
}
