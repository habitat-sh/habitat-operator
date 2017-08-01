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
	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

const leaderFollowerTopologyMinCount = 3

type validationError struct {
	msg string
	// The key in the spec that contains an error.
	Key string
}

func (err validationError) Error() string {
	return err.msg
}

func validateCustomObject(sg crv1.ServiceGroup) error {
	spec := sg.Spec

	switch spec.Habitat.Topology {
	case crv1.TopologyStandalone:
	case crv1.TopologyLeaderFollower:
		if spec.Count < leaderFollowerTopologyMinCount {
			return validationError{msg: "too few instances", Key: "count"}
		}
	default:
		return validationError{msg: "unknown topology", Key: "topology"}
	}

	return nil
}

// newListWatchFromClientWithLabels is a modified newListWatchFromClient function from listWatch.
// Instead of using fields to filter, we modify the function to use labels.
func newListWatchFromClientWithLabels(c cache.Getter, resource string, namespace string, labelSelector labels.Selector) *cache.ListWatch {
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			LabelsSelectorParam(labelSelector).
			Do().
			Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			LabelsSelectorParam(labelSelector).
			Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}
