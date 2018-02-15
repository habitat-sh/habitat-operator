# The habitat-operator helm charts

The `helm/charts` directory hosts all the helm charts related to the habitat-operator.

To add the chart for a new release of the habitat-operator run the `chart-for-latest.sh` script - it's in the toplevel directory. The script will create a new commit for you to review. If everything looks fine, feel free to push the changes:

```console
$ git push origin gh-pages
```

Then to test if everything works:

```console
$ helm repo add habitat https://kinvolk.github.io/habitat-operator/helm/charts/stable/
$ helm install habitat/habitat-operator
```
