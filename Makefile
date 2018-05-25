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
	go test $(shell go list ./... | grep -v /vendor/ | grep -v /test/)

# requires minikube or any kubernetes cluster to be running
e2e:
	$(eval IP := $(shell kubectl config view --output=jsonpath='{.clusters[0].cluster.server}' --minify | grep --only-matching '[0-9.]\+' | head --lines 1))
	$(eval KUBECONFIG_PATH := $(shell mktemp --tmpdir operator-e2e.XXXXXXX))
	kubectl config view --minify --flatten > $(KUBECONFIG_PATH)
	@if test 'x$(TESTIMAGE)' = 'x'; then echo "TESTIMAGE must be passed."; exit 1; fi
	go test -v ./test/e2e/... --image "$(TESTIMAGE)" --kubeconfig $(KUBECONFIG_PATH) --ip "$(IP)"

clean-test:
	kubectl delete namespace testing-v1beta1
	kubectl delete clusterrolebinding habitat-operator-v1beta1
	kubectl delete clusterrole habitat-operator-v1beta1

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
	find examples helm -regex '.*\.ya?ml.bak' -type f \
		-exec rm '{}' \;
	sed \
		-i.bak \
		-e 's/\(e\.g `v\).*\(`\)/\1'"$$(cat VERSION)"'\2/' \
		helm/habitat-operator/README.md
	rm -f helm/habitat-operator/README.md.bak
	sed \
		-i.bak \
		-e "s/\(habitat-operator:v\).*/\1$$(cat VERSION)/g" \
		test/e2e/v1beta1/resources/operator/deployment.yml
	rm -f test/e2e/v1beta1/resources/operator/deployment.yml.bak
	sed \
		-i.bak \
		-e 's/\(`\*: cut \)[.[:digit:]]*\( release`\)/\1'"$$(cat VERSION)"'\2/' \
		release.md
	rm -f release.md.bak

codegen:
	CODEGEN_PKG=../../../k8s.io/code-generator hack/update-codegen.sh

.PHONY: build test linux image e2e clean-test update-version
