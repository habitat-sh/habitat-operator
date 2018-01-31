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

package controller

import (
	"fmt"

	habv1beta1 "github.com/kinvolk/habitat-operator/pkg/apis/habitat/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

const leaderFollowerTopologyMinCount = 3

type habitatNotFoundError struct {
	key string
}

func (err habitatNotFoundError) Error() string {
	return fmt.Sprintf("could not find Habitat with key %s", err.key)
}

func validateCustomObject(h habv1beta1.Habitat) error {
	spec := h.Spec

	switch spec.Service.Topology {
	case habv1beta1.TopologyStandalone:
	case habv1beta1.TopologyLeader:
		if spec.Count < leaderFollowerTopologyMinCount {
			return fmt.Errorf("too few instances: %d, leader-follower topology requires at least %d", spec.Count, leaderFollowerTopologyMinCount)
		}
	default:
		return fmt.Errorf("unkown topology: %s", spec.Service.Topology)
	}

	if rsn := spec.Service.RingSecretName; rsn != "" {
		ringParts := ringRegexp.FindStringSubmatch(rsn)

		// The ringParts slice should have a second element for the capturing group
		// in the ringRegexp regular expression, containing the ring's name.
		if len(ringParts) < 2 {
			return fmt.Errorf("malformed ring secret name: %s", rsn)
		}
	}

	return nil
}

// newListWatchFromClientWithLabels is a modified newListWatchFromClient function from listWatch.
// Instead of using fields to filter, we modify the function to use labels.
func newListWatchFromClientWithLabels(c cache.Getter, resource string, namespace string, op metav1.ListOptions) *cache.ListWatch {
	listFunc := func(_ metav1.ListOptions) (runtime.Object, error) {
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&op, metav1.ParameterCodec).
			Do().
			Get()
	}
	watchFunc := func(_ metav1.ListOptions) (watch.Interface, error) {
		op.Watch = true
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&op, metav1.ParameterCodec).
			Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}
