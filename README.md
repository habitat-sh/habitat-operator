[![Build Status](https://travis-ci.org/kinvolk/habitat-operator.svg?branch=master)](https://travis-ci.org/kinvolk/habitat-operator) 
[![Go Report Card](https://goreportcard.com/badge/github.com/kinvolk/habitat-operator)](https://goreportcard.com/report/github.com/kinvolk/habitat-operator)

# habitat-operator

This project is currently unstable - breaking changes may still land in the future.

## Overview

The Habitat operator is a Kubernetes controller designed to solve running and auto-managing Habitat Services on Kubernetes. It does this by making use of [`Custom Resource Definition`][crd]s.

To learn more about Habitat, please visit the [Habitat website](https://www.habitat.sh/).

For a more detailed description of the Habitat operator API have a look at the [API documentation](https://github.com/kinvolk/habitat-operator/blob/master/docs/api.md).

## Prerequisites

- Habitat `>= 0.36.0`
- Kubernetes `>= 1.7.0`.

## Installing

    go get -u github.com/kinvolk/habitat-operator/cmd/habitat-operator

## Building manually from source directory

First clone the operator:

    git clone https://github.com/kinvolk/habitat-operator.git
    cd habitat-operator

Then build it:

    make build

## Usage

### Running outside of a Kubernetes cluster

Start the Habitat operator by running:

    habitat-operator --kubeconfig ~/.kube/config

If you built the operator manually, you'll have to specify the path to the binary. So from the root of the source directory, run:

    ./habitat-operator --kubeconfig ~/.kube/config

### Running inside a Kubernetes cluster

#### Building image from source

First build the image:

    make image

This will produce a `kinvolk/habitat-operator` image, which can then be deployed to your cluster.

The name of the generated docker image can be changed with an `IMAGE` variable, for example `make image IMAGE=mycorp/my-habitat-operator`. If the `habitat-operator` name is fine, then a `REPO` variable can be used like `make image REPO=mycorp` to generate the `mycorp/habitat-operator` image. Use the `TAG` variable to change the tag to something else (the default value is taken from `git describe --tags --always`) and a `HUB` variable to avoid using the default docker hub.

#### Using release image

Habitat operator images are located [here](https://hub.docker.com/r/kinvolk/habitat-operator/), they are tagged with the release version.

#### Deploying Habitat operator

To deploy the operator inside the Kubernetes cluster use the Deployment manifest file located under the examples directory:

    kubectl create -f examples/habitat-operator-deployment.yml

### Deploying an example

To create an example service run:

    kubectl create -f examples/standalone/habitat.yml

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
