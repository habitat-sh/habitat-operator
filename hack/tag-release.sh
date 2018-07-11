#!/bin/bash

# This script will create a tag from the contents of the VERSION file
# and push it to the habitat upstream repo.

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
  local REMOTE

  if [[ $# -gt 0 ]]; then
    # Let's err on the side of caution.
    if [[ "${1}" == '-r' ]] || [[ "${1}" == '--run' ]]; then
      DRY_RUN=0
    fi
  fi

  if [[ "${DRY_RUN}" -eq 1 ]]; then
    echo "Script is running in dry run mode. Following commands will be executed if you pass the -r or --run flag:"
  fi

  run git tag -a "${VERSION}" -m "${VERSION}"

  REMOTE=$(git remote -v | grep habitat-sh/habitat-operator | tail -1 | cut -f 1)
  run git push "${REMOTE}" "${VERSION}"
}

main "$@"
