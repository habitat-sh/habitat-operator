# The habitat-operator helm charts

This directory hosts all the helm charts related to the habitat-operator.

To add chart for a new release of habitat-operator run the following from the `helm/chart` directory:

```console
$ ./chart-for-latest.sh
$ git push origin gh-pages
```

Then to test if all works:

```console
$ helm repo add habitat https://kinvolk.github.io/habitat-operator/helm/charts/stable/
$ helm install habitat/habitat-operator
```
