#!/bin/sh

set -e

dir=$(dirname $0)

git -C "${dir}" checkout master -- VERSION helm/habitat-operator
# Drop unwanted/untracked contents in the helm/habitat-operator
# directory, so they won't end up in the chart file. This will not
# remove new files that appeared after the "git checkout" command,
# because they are also immediately staged to be commited.
git -C "${dir}" clean -ffdx -- helm/habitat-operator

stablesubdir='helm/charts/stable'
version=$(cat "${dir}/VERSION")

helm package "${dir}/helm/habitat-operator"
mv "habitat-operator-${version}.tgz" "${dir}/${stablesubdir}/"
helm repo index stable --url "https://kinvolk.github.io/habitat-operator/${stablesubdir}/" --merge "${dir}/${stablesubdir}/index.yaml"
git -C "${dir}" add "${stablesubdir}/index.yaml" "${stablesubdir}/habitat-operator-${version}.tgz"
git -C "${dir}" commit -m "Add helm chart for ${version}"
