// Copyright (c) 2018 Chef Software Inc. and/or applicable contributors
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
	"reflect"
	"testing"

	habv1beta1 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestHabitatKeyFromLabeledResource(t *testing.T) {
	labelName := habv1beta1.HabitatNameLabel
	tests := []struct {
		name    string
		arg     *metav1.ObjectMeta
		want    string
		wantErr bool
	}{
		{
			name: "label does not exists",
			arg: &metav1.ObjectMeta{
				Labels: map[string]string{},
			},
			wantErr: true,
		},
		{
			name: "label exists but value is empty",
			arg: &metav1.ObjectMeta{
				Labels: map[string]string{
					labelName: "",
				},
			},
			wantErr: true,
		},
		{
			name: "label exists and test passes",
			arg: &metav1.ObjectMeta{
				Labels: map[string]string{
					labelName: "myapp",
				},
				Namespace: "myproject",
			},
			want: "myproject/myapp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := habitatKeyFromLabeledResource(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("habitatKeyFromLabeledResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err != nil {
				t.Logf("habitatKeyFromLabeledResource() failed as expected with error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("habitatKeyFromLabeledResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewConfigMap(t *testing.T) {
	type args struct {
		ip string
		h  *habv1beta1.Habitat
	}
	// only one code path so only one test
	namespace := "myproject"
	name := peerFile
	uid := types.UID("7ed09361-9b98-11e8-ba7d-080027cc5126")
	ip := "192.168.1.1"

	tests := []struct {
		name string
		args args
		want *apiv1.ConfigMap
	}{
		{
			name: "working test case",
			args: args{
				ip: ip,
				h: &habv1beta1.Habitat{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
						UID:       uid,
					},
				},
			},
			want: &apiv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Labels: map[string]string{
						habv1beta1.HabitatLabel: "true",
					},
					OwnerReferences: []metav1.OwnerReference{
						metav1.OwnerReference{
							APIVersion: habv1beta1.SchemeGroupVersion.String(),
							Kind:       habv1beta1.HabitatKind,
							Name:       name,
							UID:        uid,
						},
					},
				},
				Data: map[string]string{
					peerFile: ip,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newConfigMap(tt.args.ip, tt.args.h); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newConfigMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
