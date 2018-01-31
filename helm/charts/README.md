# The habitat-operator helm charts

This directory hosts all the helm charts related to the habitat-operator.

To add chart for a new release of habitat-operator:

```console
$ cd stable # assuming you're already in helm/charts directory
$ helm package ../../habitat-operator
$ helm repo index . --url https://kinvolk.github.io/habitat-operator/helm/charts/stable/ --merge index.yaml
$ git add index.yaml habitat-operator-RELEASE.tgz
$ git commit -m 'Add helm chart for RELEASE'
$ git push origin gh-pages
```

Then to test if all works:

```console
$ helm repo add habitat https://kinvolk.github.io/habitat-operator/helm/charts/stable/
$ helm install habitat/habitat-operator
```
