[![Build Status](https://travis-ci.org/kinvolk/habitat-operator.svg?branch=master)](https://travis-ci.org/kinvolk/habitat-operator) 
[![Go Report Card](https://goreportcard.com/badge/github.com/kinvolk/habitat-operator)](https://goreportcard.com/report/github.com/kinvolk/habitat-operator)

# habitat-operator

## Prerequisites

The Habitat Operator makes use of [`Custom Resource Definition`][crd]s, and requires a Kubernetes cluster of version `>= 1.7.0`.

## Installing

    go get -u github.com/kinvolk/habitat-operator/cmd/operator

## Usage

### Running outside of the Kubernetes cluster:

First build the `habitat-operator` binary by running:

    make build

This will produce a binary file, then start your operator by running:

    operator --kubeconfig ~/.kube/config

To try out the operator with an example service, run:

    kubectl create -f examples/standalone

This will create a 1-pod deployment of an `nginx` Habitat service.

### Running inside of the Kubernetes cluster:

First build the image:

    make image

This will produce a `kinvolk/habitat-operator` image, which can then be deployed to your cluster.

The name of the generated docker image can be changed with an `IMAGE` variable, for example `make image IMAGE=mycorp/my-habitat-operator`. If the `habitat-operator` name is fine, then a `REPO` variable can be used like `make image REPO=mycorp` to generate the `mycorp/habitat-operator` image. Use the `TAG` variable to change the tag to something else (the default value is taken from `git describe --tags --always`) and a `HUB` variable to avoid using the default docker hub.

## Contributing

### Dependency management

This project uses [go dep](https://github.com/golang/dep/) for dependency
management.

If you add, remove or change an import, run:

    dep ensure

[crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/
