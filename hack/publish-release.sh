#!/bin/bash

# This script will create a docker image, tag it based on the contents
# of the VERSION file and push it to the docker repository.

set -euo pipefail

DRY_RUN=1
readonly VERSION="v$(cat VERSION)"

run() {
  if [[ "${DRY_RUN}" -eq 1 ]]; then
    printf '%s\n' "$*"
  else
    "$@"
  fi
}

main() {
  if [[ $# -gt 0 ]]; then
    # Let's err on the side of caution.
    if [[ "${1}" == '-r' ]] || [[ "${1}" == '--run' ]]; then
      DRY_RUN=0
    fi
  fi

  if [[ "${DRY_RUN}" -eq 1 ]]; then
    echo "Script is running in dry run mode. Following commands will be executed if you pass the -r or --run flag:"
  fi

  run make image
  run docker push habitat/habitat-operator:"${VERSION}"
  run docker tag habitat/habitat-operator:"${VERSION}" habitat/habitat-operator:latest
  run docker push habitat/habitat-operator:latest
}

main "$@"
