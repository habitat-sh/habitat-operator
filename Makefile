HUB :=
GITHUB_ORG := habitat-sh
DOCKER_ORG := habitat
IMAGE := $(if $(HUB),$(HUB)/)$(DOCKER_ORG)/habitat-operator
BIN_PATH := habitat-operator
TAG := $(shell git describe --tags --always)
TESTIMAGE :=
SUDO :=
VERSION :=

# determine if there are some uncommited changes
changes := $(shell git status --porcelain)
ifeq ($(changes),)
	VERSION := $(TAG)
else
	VERSION := $(TAG)-dirty
endif

.PHONY: build
build:
	go build -ldflags="-X github.com/$(GITHUB_ORG)/habitat-operator/pkg/version.VERSION=$(VERSION)" \
		github.com/$(GITHUB_ORG)/habitat-operator/cmd/habitat-operator

.PHONY: linux
linux: build
	# Compile statically linked binary for linux.
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s" \
		-ldflags="-X github.com/$(GITHUB_ORG)/habitat-operator/pkg/version.VERSION=$(VERSION)" \
		-o $(BIN_PATH) github.com/$(GITHUB_ORG)/habitat-operator/cmd/habitat-operator

.PHONY: print-version
print-version:
	@echo $(VERSION)

.PHONY: image
image: linux
	$(SUDO) docker build -t "$(IMAGE):$(TAG)" .

.PHONY: test
test:
	go test $(shell go list ./... | grep -v /vendor/ | grep -v /test/)
	# Run the RBAC sync tests
	go test github.com/habitat-sh/habitat-operator/test/sync/rbac

# requires minikube or any kubernetes cluster to be running
.PHONY: e2e
e2e: clean-test
	$(eval IP := $(shell kubectl config view --output=jsonpath='{.clusters[0].cluster.server}' --minify | grep --only-matching '[0-9.]\+' | head --lines 1))
	$(eval KUBECONFIG_PATH := $(shell mktemp --tmpdir operator-e2e.XXXXXXX))
	kubectl config view --minify --flatten > $(KUBECONFIG_PATH)
	@if test 'x$(TESTIMAGE)' = 'x'; then echo "TESTIMAGE must be passed."; exit 1; fi
	# control the order in which tests are run
	go test -v ./test/e2e/v1beta1/clusterwide/... --image "$(TESTIMAGE)" --kubeconfig $(KUBECONFIG_PATH) --ip "$(IP)"
	go test -v ./test/e2e/v1beta1/namespaced/... --image "$(TESTIMAGE)" --kubeconfig $(KUBECONFIG_PATH) --ip "$(IP)"

.PHONY: clean-test
clean-test:
	# Delete resources created for the clusterwide tests
	-kubectl delete namespace testing-clusterwide
	-kubectl delete clusterrolebinding habitat-operator-v1beta1
	-kubectl delete clusterrole habitat-operator-v1beta1
	-kubectl delete crd habitats.habitat.sh
	# Delete resources created for the namespaced tests
	-kubectl delete namespace testing-namespaced

.PHONY: update-version
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
	# The deployments artifact for clusterwide and namespaced tests should be updated
	# to have the newer updated image name
	for f in test/e2e/v1beta1/{clusterwide,namespaced}/resources/operator/deployment.yml; do \
		sed \
			-i.bak \
			-e "s/\(habitat-operator:v\).*/\1$$(cat VERSION)/g" $$f; \
		rm -f $$f.bak; \
	done

	sed \
		-i.bak \
		-e 's/\(`\*: cut \)[.[:digit:]]*\( release`\)/\1'"$$(cat VERSION)"'\2/' \
		doc/release-process.md
	rm -f doc/release-process.md.bak

.PHONY: codegen
codegen:
	hack/update-codegen.sh

