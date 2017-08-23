# Namespaced ServiceGroup example

## Workflow

Simply run `kubectl create -f examples/namespaced/service_group.yml`.

Note that any Secrets used by a ServiceGroup must be in the same namespace as
the ServiceGroup itself.
