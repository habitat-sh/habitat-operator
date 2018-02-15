#!/bin/sh

set -e

helm package ../habitat-operator
mv habitat-operator-$(cat ../../VERSION).tgz stable/
helm repo index stable --url https://kinvolk.github.io/habitat-operator/helm/charts/stable/ --merge stable/index.yaml
git add stable/index.yaml stable/habitat-operator-$(cat ../../VERSION).tgz
git commit -m "Add helm chart for $(cat ../../VERSION)"
