![travis] ![report]

# habitat-operator

## Prerequisites

The Habitat Operator makes use of [`Custom Resource Definition`][crd]s, and requires a Kubernetes cluster of version `>= 1.7.0`.

## Installing

    go get -u github.com/kinvolk/habitat-operator/cmd/operator

## Usage

To run the `habitat-operator` as a binary outside of a Kubernetes cluster, run:

    operator --kubeconfig ~/.kube/config

To try out the operator with an example service, run:

    kubectl create -f examples/habitat_service-standalone.yml

This will create a 1-pod deployment of an `nginx` Habitat service.

## Contributing

### Dependency management

This project uses [go dep](https://github.com/golang/dep/) for dependency
management.

If you add, remove or change an import, run:

    dep ensure

[travis]: https://travis-ci.org/kinvolk/habitat-operator.svg?branch=master
[report]: https://goreportcard.com/badge/github.com/kinvolk/habitat-operator
[crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/
