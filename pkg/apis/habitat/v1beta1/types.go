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
	HabitatShortName      = "hab"

	// HabitatLabel labels the resources that belong to Habitat.
	// Example: 'habitat: true'
	HabitatLabel = "habitat"
	// HabitatNameLabel contains the user defined Habitat Service name.
	// Example: 'habitat-name: db'
	HabitatNameLabel = "habitat-name"

	TopologyLabel        = "topology"
	HabitatTopologyLabel = "operator.habitat.sh/topology"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Habitat struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              HabitatSpec   `json:"spec"`
	Status            HabitatStatus `json:"status,omitempty"`
	// CustomVersion is a field that works around the lack of support for running
	// multiple versions of a CRD.  It encodes the actual version of the type, so
	// that controllers can decide whether to discard an object if the version
	// doesn't match.
	CustomVersion *string `json:"customVersion,omitempty"`
}

type HabitatSpec struct {
	// V1beta2 are fields for the v1beta2 type.
	// +optional
	V1beta2 *V1beta2 `json:"v1beta2"`
}

// V1beta2 are fields for the v1beta2 type.
type V1beta2 struct {
	// Count is the amount of Services to start in this Habitat.
	Count int `json:"count"`
	// ServiceAccountName is the service account that your service can run as when
	// Kubernetes is running in RBAC mode
	ServiceAccountName *string `json:"serviceAccountName,omitempty"`
	// Image is the Docker image of the Habitat Service.
	Image   string         `json:"image"`
	Service ServiceV1beta2 `json:"service"`
	// Env is a list of environment variables.
	// The EnvVar type is documented at https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#envvar-v1-core.
	// Optional.
	Env []corev1.EnvVar `json:"env,omitempty"`
	// +optional
	PersistentStorage *PersistentStorage `json:"persistentStorage,omitempty"`
}

// PersistentStorage contains the details of the persistent storage that the
// cluster should provision.
type PersistentStorage struct {
	// Size is the volume's size.
	// It uses the same format as Kubernetes' size fields, e.g. 10Gi
	Size string `json:"size"`
	// MountPath is the path at which the PersistentVolume will be mounted.
	MountPath string `json:"mountPath"`
	// StorageClassName is the name of the StorageClass that the StatefulSet will request.
	StorageClassName string `json:"storageClassName"`
}

type HabitatStatus struct {
	State   HabitatState `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
}

type HabitatState string

type ServiceV1beta2 struct {
	// Group is the value of the --group flag for the hab client.
	// Defaults to `default`.
	// +optional
	Group *string `json:"group,omitempty"`
	// Topology is the value of the --topology flag for the hab client.
	Topology `json:"topology"`
	// ConfigSecretName is the name of a Secret containing a Habitat service's config in TOML format.
	// It will be mounted inside the pod as a file, and it will be used by Habitat to configure the service.
	// +optional
	ConfigSecretName *string `json:"configSecretName,omitempty"`
	// The name of the secret that contains the ring key.
	// +optional
	RingSecretName *string `json:"ringSecretName,omitempty"`
	// The name of a secret containing the files directory.  It will be mounted inside the pod
	// as a directory.
	// +optional
	FilesSecretName *string `json:"filesSecretName,omitempty"`
	// Bind is when one service connects to another forming a producer/consumer relationship.
	// +optional
	Bind []Bind `json:"bind,omitempty"`
	// Name is the name of the Habitat service that this Habitat object represents.
	// This field is used to mount the user.toml file in the correct directory under /hab/user/ in the Pod.
	Name string `json:"name"`
	// Channel is the value of the --channel flag for the hab client.
	// It can be used to track upstream packages in builder channels but will never be used directly by the supervisor.
	// The should only be used in conjunction with the habitat updater https://github.com/habitat-sh/habitat-updater
	// Defaults to `stable`.
	// +optional
	Channel *string `json:"channel,omitempty"`
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
