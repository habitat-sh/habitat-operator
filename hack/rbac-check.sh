#!/bin/bash

set -euo pipefail

readonly dir=$(dirname "${BASH_SOURCE[0]}")

function say
{
    printf '%s\n' "${1}"
}

function join_newline
{
    local IFS=$'\n'
    say "$*"
}

function read_rules
{
    local path
    local line
    local ignore
    local rules

    path="${1}"
    ignore=1
    rules=()

    while IFS="" read -r line || [[ -n "${line}" ]]
    do
        # strip the newline
        line="${line%\\n}"
        if [[ -z "${line}" ]]
        then
            # ignore empty lines
            :
        elif [[ "${line}" = 'rules:' ]]
        then
            # rules begin from the next line
            ignore=0
        elif [[ "${line}" =~ '{{' ]]
        then
            # some helm specific stuff, ignore a line after rules:
            # that starts with {{
            :
        elif [[ $ignore -eq 0 ]]
        then
            rules+=("${line}")
        fi
    done <"${path}"

    say "$(join_newline "${rules[@]}")"
}

readonly example_path='examples/rbac/rbac.yml'
readonly chart_path='helm/habitat-operator/templates/clusterrole.yaml'
readonly test_path='test/e2e/v1beta1/resources/operator/cluster-role.yml'

readonly example_rules="$(read_rules "${dir}/../${example_path}")"
readonly chart_rules="$(read_rules "${dir}/../${chart_path}")"
readonly test_rules="$(read_rules "${dir}/../${test_path}")"

# rely on transitivity, if example == chart and chart == test then
# example == test

say "Diff between ${example_path} and ${chart_path}:"
if diff <(say "${example_rules}") <(say "${chart_rules}")
then
    say 'OK, none'
else
    exit 1
fi

say "Diff between ${chart_path} and ${test_path}:"
if diff <(say "${chart_rules}") <(say "${test_rules}")
then
    say 'OK, none'
else
    exit 1
fi
