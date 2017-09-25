EIP := $(shell minikube ip)
HUB :=
REPO := kinvolk
IMAGE := $(if $(HUB),$(HUB)/)$(REPO)/habitat-operator
TAG := $(shell git describe --tags --always)
TESTIMAGE :=

build:
	go build -i github.com/kinvolk/habitat-operator/cmd/habitat-operator

linux:
	# Compile statically linked binary for linux.
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -i --ldflags="-s" -o habitat-operator github.com/kinvolk/habitat-operator/cmd/habitat-operator

image: linux
	docker build -t "$(IMAGE):$(TAG)" .

test:
	go test -v $(shell go list ./... | grep -v /vendor/ | grep -v /test/)

e2e:
	@if test 'x$(TESTIMAGE)' = 'x'; then echo "TESTIMAGE must be passed."; exit 1; fi
	go test -v ./test/e2e/ --image "$(TESTIMAGE)" --kubeconfig ~/.kube/config --ip "$(EIP)"

clean-test:
	kubectl delete sg mytutorialapp
	kubectl delete sg test-service-group
	kubectl delete sg test-standalone
	kubectl delete sg test-bind-go
	kubectl delete sg test-bind-db
	kubectl delete crd servicegroups.habitat.sh
	kubectl delete pod habitat-operator
	kubectl delete secret mytutorialapp

.PHONY: build test linux image e2e clean-test
