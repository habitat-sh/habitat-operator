HUB :=
REPO := kinvolk
IMAGE := $(if $(HUB),$(HUB)/)$(REPO)/habitat-operator
TAG := $(shell git describe --tags --always)

build:
	go build -i github.com/kinvolk/habitat-operator/cmd/operator

linux:
	env GOOS=linux go build github.com/kinvolk/habitat-operator/cmd/operator

image: linux
	docker build -t "$(IMAGE):$(TAG)" .

test:
	go test -v $(shell go list ./... | grep -v /vendor/)

.PHONY: build test linux image
