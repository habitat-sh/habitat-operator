[![Build Status](https://travis-ci.org/kinvolk/habitat-operator.svg?branch=master)](https://travis-ci.org/kinvolk/habitat-operator) 
[![Go Report Card](https://goreportcard.com/badge/github.com/kinvolk/habitat-operator)](https://goreportcard.com/report/github.com/kinvolk/habitat-operator)

# habitat-operator

## Prerequisites

The Habitat Operator makes use of [`Custom Resource Definition`][crd]s, and requires a Kubernetes cluster of version `>= 1.7.0`.

At the moment, the Operator requires a forked version of the Habitat supervisor
that adds support for the `--peer-watch-file` flag.
See [this issue](https://github.com/habitat-sh/habitat/issues/2735) to track
progress on upstreaming the feature.

This means that images need to be created using the aforementioned fork of the
`hab` client, which can be accomplished using [this
script](https://gist.github.com/krnowak/3c854e94245e2f33a8366e629bfb09c8).

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

### Testing

To run unit tests, run:

    make test

[crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/

## Testing

To run end-to-end tests you need a `minikube` up and running. After that just run:
 
    make TESTIMAGE=YOUR_OPERATOR_IMAGE e2e

Clean up after the tests by either running `minikube delete` (this will delete your entire cluster) or:

    make clean-test
