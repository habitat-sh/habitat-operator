HUB :=
GITHUB_ORG := habitat-sh
DOCKER_ORG := habitat
IMAGE := $(if $(HUB),$(HUB)/)$(DOCKER_ORG)/habitat-operator
BIN_PATH := habitat-operator
TAG := $(shell git describe --tags --always)
TESTIMAGE :=
SUDO :=

build:
	go build -i github.com/$(GITHUB_ORG)/habitat-operator/cmd/habitat-operator

linux:
	# Compile statically linked binary for linux.
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s" -o $(BIN_PATH) github.com/$(GITHUB_ORG)/habitat-operator/cmd/habitat-operator

image: linux
	$(SUDO) docker build -t "$(IMAGE):$(TAG)" .

test:
	go test -v $(shell go list ./... | grep -v /vendor/ | grep -v /test/)

# requires minikube to be running
e2e:
	@if test 'x$(TESTIMAGE)' = 'x'; then echo "TESTIMAGE must be passed."; exit 1; fi
	go test -v ./test/e2e/ --image "$(TESTIMAGE)" --kubeconfig ~/.kube/config --ip "$$(minikube ip)"

clean-test:
	kubectl delete namespace testing
	kubectl delete clusterrolebinding habitat-operator
	kubectl delete clusterrole habitat-operator

update-version:
	find examples -name "*.yml" -type f \
		-exec sed -i.bak \
		-e "s/habitat-operator:.*/habitat-operator:v$$(cat VERSION)/g" \
		'{}' \;
	find helm -name "*.yaml" -type f \
		-exec sed -i.bak \
		-e "s/tag:.*/tag: v$$(cat VERSION)/g" \
		-e "s/version:.*/version: $$(cat VERSION)/g" \
		'{}' \;
	find examples helm -regex '.*\.ya?ml$$.bak' -type f \
		-exec rm '{}' \;

codegen:
	CODEGEN_PKG=../../../k8s.io/code-generator hack/update-codegen.sh

.PHONY: build test linux image e2e clean-test update-version
