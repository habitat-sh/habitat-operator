#!/bin/bash

# Copyright 2017 The Kubernetes Authors.
# Copyright 2018 Chef Software Inc. and/or applicable contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Based on https://github.com/kubernetes/code-generator/blob/893b4433a4ba929dd0dacebf7c8956682d7a5d5f/hack/verify-codegen.sh

# This script is run in the CI.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

"${SCRIPT_ROOT}/hack/update-codegen.sh"
echo "Checking against freshly generated codegen..."

ret=0

git diff --quiet "*.deepcopy.go" || ret=$?

if [[ $ret -eq 0 ]]; then
  echo "  Up to date."
else
  echo "  Out of date. Please run hack/update-codegen.sh"
  exit 1
fi
