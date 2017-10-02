[![Build Status](https://travis-ci.org/kinvolk/habitat-operator.svg?branch=master)](https://travis-ci.org/kinvolk/habitat-operator) 
[![Go Report Card](https://goreportcard.com/badge/github.com/kinvolk/habitat-operator)](https://goreportcard.com/report/github.com/kinvolk/habitat-operator)

# habitat-operator

This project is currently unstable - breaking changes may still land in the future.
Note: At the moment the Habitat operator requires a forked version of the Habitat supervisor that adds support for the `--peer-watch-file` flag. See [this issue](https://github.com/habitat-sh/habitat/issues/2735) to track progress on upstreaming the feature.

## Overview

The Habitat operator is a Kubernetes controller designed to solve running and auto-managing Habitat Services on Kubernetes. It does this by making use of [`Custom Resource Definition`][crd]s.

To learn more about Habitat, please visit the [Habitat website](https://www.habitat.sh/).

For a more detailed description of the Habitat operator API have a look at the [API documentation](https://github.com/kinvolk/habitat-operator/blob/master/docs/api.md).

## Prerequisites

- Kubernetes `>= 1.7.0`.
- Forked version of the Habitat supervisor. This means that images need to be created using the aforementioned fork of the `hab` client, which can be accomplished following [this script](https://gist.github.com/LiliC/c028fc4687f466e3e3bd5981d2529173).

## Installing

    go get -u github.com/kinvolk/habitat-operator/cmd/habitat-operator

## Usage

### Running outside of a Kubernetes cluster

Start the Habitat operator by running:

    habitat-operator --kubeconfig ~/.kube/config

### Running inside a Kubernetes cluster

First build the image:

    make image

This will produce a `kinvolk/habitat-operator` image, which can then be deployed to your cluster.

The name of the generated docker image can be changed with an `IMAGE` variable, for example `make image IMAGE=mycorp/my-habitat-operator`. If the `habitat-operator` name is fine, then a `REPO` variable can be used like `make image REPO=mycorp` to generate the `mycorp/habitat-operator` image. Use the `TAG` variable to change the tag to something else (the default value is taken from `git describe --tags --always`) and a `HUB` variable to avoid using the default docker hub.

### Deploying an example

To create an example service run:

    kubectl create -f examples/standalone/service_group.yml

This will create a single-pod deployment of an `nginx` Habitat service.
More examples are located in the [example directory](https://github.com/kinvolk/habitat-operator/tree/master/examples/).

## Contributing

### Dependency management

This project uses [go dep](https://github.com/golang/dep/) for dependency management.

If you add, remove or change an import, run:

    dep ensure

### Testing

To run unit tests locally, run:

    make test

To run end-to-end tests locally you need to have `minikube` up and running. After that just run:
 
    make TESTIMAGE=YOUR_OPERATOR_IMAGE e2e

Clean up after the tests with:

    make clean-test

[crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/
