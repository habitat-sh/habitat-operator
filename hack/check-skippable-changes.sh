#!/bin/bash

# This script checks to see if changes only affect documentation and examples
# and skips running tests if that's the case in a pull request

set -eu

# Finds the branching point of two commits.
# For example, let B and D be two commits, and their ancestry graph as A -> B, A -> C -> D.
# Given commits B and D, it returns A.
# https://github.com/rkt/rkt/blob/2de014aa1a8f1d9e92bc869e7d666679f2e45a1d/tests/build-and-run-tests.sh#L31-L38
function getBranchingPoint {
    diff --old-line-format='' --new-line-format='' \
        <(git rev-list --first-parent "${1}") \
            <(git rev-list --first-parent "${2}") | head -1
}

DOC_EXAMPLE_CHANGE_PATTERN=(
            '-e' '^doc/'
            '-e' '^examples/'
            '-e' '^(VERSION|LICENSE)$'
            '-e' '\.md$'
)


BRANCHING_POINT=$(getBranchingPoint HEAD origin/master)
SRC_CHANGES=$(git diff-tree --no-commit-id --name-only -r HEAD.."${BRANCHING_POINT}" | grep -cEv "${DOC_EXAMPLE_CHANGE_PATTERN[@]}") || true

if [[ "${SRC_CHANGES}" -eq 0 ]] && [[ -n "${CIRCLE_PULL_REQUEST}" ]]; then
  # Skip build because only documentation and examples changed and this is a PR
  echo "Skipping CI build"
  circleci step halt
else
  echo "Continuing CI build"
fi
