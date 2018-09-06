#!/bin/bash

set -euo pipefail

readonly dir=$(dirname "${BASH_SOURCE[0]}")

function say
{
    printf '%s\n' "${1}"
}

# CRD file used in example and the test should be in sync.
readonly example="${dir}/../examples/rbac-restricted/crd.yml"
readonly test="${dir}/../test/e2e/v1beta1/namespaced/resources/operator/crd.yml"

# Doing simple diff would help us detect any inconsistencies in two files
say "Diff between ${example} and ${test}:"
if diff ${example} ${test}
then
    say 'OK, none'
else
    exit 1
fi
