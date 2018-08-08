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

import "testing"

func TestCheckCustomVersionMatch(t *testing.T) {
	tests := []struct {
		name    string
		arg     *string
		wantErr bool
	}{
		{
			name:    "nothing specified in the version",
			arg:     nil,
			wantErr: true,
		},
		{
			name:    "custom version is v1beta2",
			arg:     strToPtr("v1beta2"),
			wantErr: false,
		},
		{
			name:    "random custom version specified",
			arg:     strToPtr("v1alpha1"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkCustomVersionMatch(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkCustomVersionMatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err != nil {
				t.Logf("failed with error: %v", err)
			}
		})
	}
}

func strToPtr(s string) *string {
	return &s
}
