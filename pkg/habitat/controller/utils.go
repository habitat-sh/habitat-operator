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

	switch spec.Topology {
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
