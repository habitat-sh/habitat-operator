# The habitat-operator helm charts

This directory hosts all the helm charts related to the habitat-operator.

To add the chart for a new release of the habitat-operator run the following from the `helm/charts` directory:

```console
$ git merge master
$ ./chart-for-latest.sh
$ git push origin gh-pages
```

Then to test if everything works:

```console
$ helm repo add habitat https://kinvolk.github.io/habitat-operator/helm/charts/stable/
$ helm install habitat/habitat-operator
```
