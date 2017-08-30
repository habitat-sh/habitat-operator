# Namespaced Habitat example

This demonstrates how to deploy service in a [Kubernetes namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/).

## Workflow

Simply run:

 `kubectl create -f examples/namespaced/habitat.yml`.

Note that any Secrets used by a `Habitat` must be in the same namespace as the `Habitat` itself.
