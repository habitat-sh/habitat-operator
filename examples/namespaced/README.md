# Namespaced ServiceGroup example

This demonstrates how to deploy service in a [Kubernetes namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/).

## Workflow

Simply run:

 `kubectl create -f examples/namespaced/service_group.yml`.

Note that any Secrets used by a `ServiceGroup` must be in the same namespace as the `ServiceGroup` itself.
