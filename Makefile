# When the VERBOSE variable is set to 1, all the commands are shown
ifeq ("$(VERBOSE)","1")
echo_prefix=">>>>"
else
VECHO = @
endif
SED ?= "sed"
ISTIOCTL="./tests/_build/istio/istio/bin/istioctl"

# Current Operator version
VERSION ?= "$(shell git describe --tags | sed 's/^v//')"
VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
VERSION_PKG ?= "github.com/jaegertracing/jaeger-operator/pkg/version"
JAEGER_VERSION ?= $(shell grep -v '\#' versions.txt | grep jaeger | awk -F= '{print $$2}')
LD_FLAGS ?= "-X $(VERSION_PKG).version=$(VERSION) -X $(VERSION_PKG).buildDate=$(VERSION_DATE) -X $(VERSION_PKG).defaultJaeger=$(JAEGER_VERSION)"
UNIT_TEST_PACKAGES := $(shell go list ./pkg/... | grep -v elasticsearch/v1 | grep -v kafka/v1beta2 | grep -v client/versioned)

ISTIO_VERSION ?= 1.11.2

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

KUBE_VERSION ?= 1.20
KIND_CONFIG ?= kind-$(KUBE_VERSION).yaml

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# jaegertracing.io/jaeger-operator-bundle:$VERSION and jaegertracing.io/jaeger-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= jaegertracing.io/jaeger-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false,maxDescLen=0,generateEmbeddedObjectMeta=true"

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# If we are running in CI, run go test in verbose mode
ifeq (,$(CI))
GOTEST_OPTS=
else
GOTEST_OPTS=-v
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run -ldflags ${LD_FLAGS} ./main.go

docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

ENVTEST = $(shell pwd)/bin/setup-envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

.PHONY: test
test: generate fmt vet envtest
	@echo Running unit tests...
	@ KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ${GOTEST_OPTS} ./... -cover -coverprofile=cover.out -ldflags $(LD_FLAGS)

# Build the container image, used only for local dev purposes
container:
	docker build -t ${IMG} --build-arg VERSION_PKG=${VERSION_PKG} --build-arg VERSION=${VERSION} --build-arg VERSION_DATE=${VERSION_DATE} --build-arg JAEGER_VERSION=${JAEGER_VERSION} .

# Push the container image, used only for local dev purposes
container-push:
	docker push ${IMG}


.PHONY: istio
istio:
	mkdir -p tests/_build/istio
	[ -f "${ISTIOCTL}" ] || (curl -L https://istio.io/downloadIstio | ISTIO_VERSION=${ISTIO_VERSION} TARGET_ARCH=x86_64 sh - && mv ./istio-${ISTIO_VERSION} ./tests/_build/istio/istio)
	${ISTIOCTL} install --set profile=minimal -y

# end-to-tests
.PHONY: kuttl-e2e
kuttl-e2e:
	$(KUTTL) test

.PHONY: prepare-kuttl-e2e
prepare-kuttl-e2e: kuttl set-test-image-vars set-image-controller prepare-kuttl-images generate-kuttl-files build render-kuttl-templates start-kind

.PHONY: generate-kuttl-files
generate-kuttl-files:
	mkdir -p tests/_build/crds tests/_build/manifests
	$(KUSTOMIZE) build config/default -o tests/_build/manifests/01-jaeger-operator.yaml
	$(KUSTOMIZE) build config/crd -o tests/_build/crds/

.PHONY: prepare-kuttl-images
prepare-kuttl-images: container build-assert-job
	$(VECHO)docker pull jaegertracing/vertx-create-span:operator-e2e-tests
	$(VECHO)docker pull docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.6
	# Image for the upgrade E2E test
	$(VECHO)docker build --build-arg=GOPROXY=${GOPROXY}  --build-arg VERSION_PKG=${VERSION_PKG} --build-arg=JAEGER_VERSION=$(shell .ci/get_test_upgrade_version.sh ${JAEGER_VERSION}) --file Dockerfile -t "local/jaeger-operator:next" .

.PHONY: start-kind
start-kind: 
	$(VECHO)kind create cluster --config $(KIND_CONFIG)
	$(VECHO)kind load docker-image local/jaeger-operator:e2e
	$(VECHO)kind load docker-image local/asserts:e2e
	$(VECHO)kind load docker-image jaegertracing/vertx-create-span:operator-e2e-tests
	$(VECHO)kind load docker-image local/jaeger-operator:next
	$(VECHO)kind load docker-image docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.6


.PHONY: render-kuttl-templates
render-kuttl-templates:
# This is needed for the generate test
	$(VECHO)JAEGER_VERSION=${JAEGER_VERSION} gomplate -f tests/e2e/generate/jaeger-template.yaml.template -o tests/e2e/generate/jaeger-deployment.yaml
	$(VECHO)JAEGER_VERSION=${JAEGER_VERSION} gomplate -f tests/e2e/upgrade/deployment-assert.yaml.template -o tests/e2e/upgrade/00-assert.yaml
	$(VECHO)JAEGER_VERSION=$(shell .ci/get_test_upgrade_version.sh ${JAEGER_VERSION}) gomplate -f tests/e2e/upgrade/deployment-assert.yaml.template -o tests/e2e/upgrade/01-assert.yaml
	$(VECHO)JAEGER_VERSION=${JAEGER_VERSION} gomplate -f tests/e2e/upgrade/deployment-assert.yaml.template -o tests/e2e/upgrade/02-assert.yaml
	$(VECHO)${SED} "s~local/jaeger-operator:e2e~local/jaeger-operator:next~gi" tests/_build/manifests/01-jaeger-operator.yaml > tests/e2e/upgrade/operator-upgrade.yaml
# This is needed for the streaming tests
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/streaming-simple/01-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/streaming-simple/01-assert.yaml
	$(VECHO)REPLICAS=1 CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-kafka-cluster.yaml.template -o tests/e2e/streaming-simple/02-assert.yaml
	$(VECHO)REPLICAS=1 CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-zookeeper-cluster.yaml.template -o tests/e2e/streaming-simple/03-assert.yaml
	$(VECHO)CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-entity-operator.yaml.template -o tests/e2e/streaming-simple/04-assert.yaml
	$(VECHO)JAEGER_SERVICE=simple-streaming JAEGER_OPERATION=smoketestoperation JAEGER_NAME=simple-streaming gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/streaming-simple/06-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/streaming-simple/06-assert.yaml
# streaming-with-tls
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/streaming-with-tls/01-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/streaming-with-tls/01-assert.yaml
	$(VECHO)REPLICAS=1 CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-kafka-cluster.yaml.template -o tests/e2e/streaming-with-tls/02-assert.yaml
	$(VECHO)REPLICAS=1 CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-zookeeper-cluster.yaml.template -o tests/e2e/streaming-with-tls/03-assert.yaml
	$(VECHO)CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-entity-operator.yaml.template -o tests/e2e/streaming-with-tls/04-assert.yaml
	$(VECHO)JAEGER_SERVICE=streaming-with-tls JAEGER_OPERATION=smoketestoperation JAEGER_NAME=tls-streaming gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/streaming-with-tls/07-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/streaming-with-tls/07-assert.yaml
# streaming-with-autoprovisioning
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/streaming-with-autoprovisioning/01-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/streaming-with-autoprovisioning/01-assert.yaml
	$(VECHO)REPLICAS=3 CLUSTER_NAME=auto-provisioned gomplate -f tests/templates/assert-zookeeper-cluster.yaml.template -o tests/e2e/streaming-with-autoprovisioning/02-assert.yaml
	$(VECHO)REPLICAS=3 CLUSTER_NAME=auto-provisioned gomplate -f tests/templates/assert-kafka-cluster.yaml.template -o tests/e2e/streaming-with-autoprovisioning/03-assert.yaml
	$(VECHO)CLUSTER_NAME=auto-provisioned gomplate -f tests/templates/assert-entity-operator.yaml.template -o tests/e2e/streaming-with-autoprovisioning/04-assert.yaml
	$(VECHO)JAEGER_SERVICE=streaming-with-autoprovisioning JAEGER_OPERATION=smoketestoperation JAEGER_NAME=auto-provisioned gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/streaming-with-autoprovisioning/06-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/streaming-with-autoprovisioning/06-assert.yaml
# examples-agent-as-daemonset
	$(VECHO)gomplate -f examples/agent-as-daemonset.yaml -o tests/e2e/examples-agent-as-daemonset/00-install.yaml
	$(VECHO)JAEGER_NAME=agent-as-daemonset gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-agent-as-daemonset/00-assert.yaml
	$(VECHO)JAEGER_SERVICE=agent-as-daemonset JAEGER_OPERATION=smoketestoperation JAEGER_NAME=agent-as-daemonset gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-agent-as-daemonset/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-agent-as-daemonset/02-assert.yaml
# examples-with-cassandra
	$(VECHO)gomplate -f tests/templates/cassandra-install.yaml.template -o tests/e2e/examples-with-cassandra/00-install.yaml
	$(VECHO)gomplate -f tests/templates/cassandra-assert.yaml.template -o tests/e2e/examples-with-cassandra/00-assert.yaml
	$(VECHO)gomplate -f examples/with-cassandra.yaml -o tests/e2e/examples-with-cassandra/01-install.yaml
	$(VECHO)${SED} -i "s~cassandra.default.svc~cassandra~gi" tests/e2e/examples-with-cassandra/01-install.yaml
	$(VECHO)JAEGER_NAME=with-cassandra gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-with-cassandra/01-assert.yaml
	$(VECHO)JAEGER_SERVICE=with-cassandra JAEGER_OPERATION=smoketestoperation JAEGER_NAME=with-cassandra gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-with-cassandra/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-with-cassandra/02-assert.yaml
# examples-business-application-injected-sidecar
	$(VECHO)cat examples/business-application-injected-sidecar.yaml tests/e2e/examples-business-application-injected-sidecar/livenessProbe.yaml >  tests/e2e/examples-business-application-injected-sidecar/00-install.yaml
	$(VECHO)gomplate -f  examples/simplest.yaml -o tests/e2e/examples-business-application-injected-sidecar/01-install.yaml
	$(VECHO)JAEGER_NAME=simplest gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-business-application-injected-sidecar/01-assert.yaml
	$(VECHO)JAEGER_SERVICE=simplest JAEGER_OPERATION=smoketestoperation JAEGER_NAME=simplest gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-business-application-injected-sidecar/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-business-application-injected-sidecar/02-assert.yaml
# istio
	$(VECHO)cat examples/business-application-injected-sidecar.yaml tests/e2e/istio/livelinessprobe.template > tests/e2e/istio/03-install.yaml
# cassandra
	$(VECHO)gomplate -f tests/templates/cassandra-install.yaml.template -o tests/e2e/cassandra/00-install.yaml
	$(VECHO)gomplate -f tests/templates/cassandra-assert.yaml.template -o tests/e2e/cassandra/00-assert.yaml
	$(VECHO)INSTANCE_NAME=with-cassandra  gomplate -f tests/templates/cassandra-jaeger-install.yaml.template -o tests/e2e/cassandra/01-install.yaml
	$(VECHO)INSTANCE_NAME=with-cassandra  gomplate -f tests/templates/cassandra-jaeger-assert.yaml.template -o tests/e2e/cassandra/01-assert.yaml
# cassandra spark
	$(VECHO) gomplate -f tests/templates/cassandra-install.yaml.template -o tests/e2e/cassandra-spark/00-install.yaml
	$(VECHO) gomplate -f tests/templates/cassandra-assert.yaml.template -o tests/e2e/cassandra-spark/00-assert.yaml
	$(VECHO)INSTANCE_NAME=test-spark-deps DEP_SCHEDULE=true CASSANDRA_MODE=prod gomplate -f tests/templates/cassandra-jaeger-install.yaml.template -o tests/e2e/cassandra-spark/01-install.yaml
# es-spark-dependencies
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/es-spark-dependencies/00-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/es-spark-dependencies/00-assert.yaml
# es-simple-prod
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/es-simple-prod/00-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/es-simple-prod/00-assert.yaml
	$(VECHO)JAEGER_NAME=simple-prod gomplate -f tests/templates/production-jaeger-install.yaml.template -o tests/e2e/es-simple-prod/01-install.yaml
	$(VECHO)JAEGER_NAME=simple-prod gomplate -f tests/templates/production-jaeger-assert.yaml.template -o tests/e2e/es-simple-prod/01-assert.yaml
	$(VECHO)JAEGER_SERVICE=simple-prod JAEGER_OPERATION=smoketestoperation JAEGER_NAME=simple-prod gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/es-simple-prod/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/es-simple-prod/02-assert.yaml
# es-index-cleaner
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/es-index-cleaner/00-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/es-index-cleaner/00-assert.yaml
	$(VECHO)JAEGER_NAME=test-es-index-cleaner-with-prefix gomplate -f tests/templates/production-jaeger-install.yaml.template -o tests/e2e/es-index-cleaner/jaeger-deployment
	$(VECHO)gomplate -f tests/e2e/es-index-cleaner/es-index.template -o tests/e2e/es-index-cleaner/es-index
	$(VECHO)cat tests/e2e/es-index-cleaner/jaeger-deployment tests/e2e/es-index-cleaner/es-index >> tests/e2e/es-index-cleaner/01-install.yaml
	$(VECHO)JAEGER_NAME=test-es-index-cleaner-with-prefix gomplate -f tests/templates/production-jaeger-assert.yaml.template -o tests/e2e/es-index-cleaner/01-assert.yaml
	$(VECHO)$(SED) "s~enabled: false~enabled: true~gi" tests/e2e/es-index-cleaner/01-install.yaml > tests/e2e/es-index-cleaner/03-install.yaml
	$(VECHO)gomplate -f tests/e2e/es-index-cleaner/01-install.yaml -o tests/e2e/es-index-cleaner/05-install.yaml
	$(VECHO)PREFIX=my-prefix gomplate -f tests/e2e/es-index-cleaner/es-index.template -o tests/e2e/es-index-cleaner/es-index2
	$(VECHO)cat tests/e2e/es-index-cleaner/jaeger-deployment tests/e2e/es-index-cleaner/es-index2 >> tests/e2e/es-index-cleaner/07-install.yaml
	$(VECHO)$(SED) "s~enabled: false~enabled: true~gi" tests/e2e/es-index-cleaner/07-install.yaml > tests/e2e/es-index-cleaner/09-install.yaml
	$(VECHO)gomplate -f tests/e2e/es-index-cleaner/04-wait-es-index-cleaner.yaml -o tests/e2e/es-index-cleaner/11-wait-es-index-cleaner.yaml
	$(VECHO)gomplate -f tests/e2e/es-index-cleaner/05-install.yaml -o tests/e2e/es-index-cleaner/12-install.yaml


set-test-image-vars:
	$(eval IMG=local/jaeger-operator:e2e)

.PHONY: build-assert-job
build-assert-job:
	@docker build -t local/asserts:e2e  -f Dockerfile.asserts .

kuttl:
ifeq (, $(shell which kubectl-kuttl))
	echo ${PATH}
	ls -l /usr/local/bin
	which kubectl-kuttl

	@{ \
	set -e ;\
	echo "" ;\
	echo "ERROR: kuttl not found." ;\
	echo "Please check https://kuttl.dev/docs/cli.html for installation instructions and try again." ;\
	echo "" ;\
	exit 1 ;\
	}
else
KUTTL=$(shell which kubectl-kuttl)
endif


# Set the controller image parameters
set-image-controller: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}

kind:
ifeq (, $(shell which kind))
	@{ \
	set -e ;\
	echo "" ;\
	echo "ERROR: kind not found." ;\
	echo "Please check https://kind.sigs.k8s.io/docs/user/quick-start/#installation for installation instructions and try again." ;\
	echo "" ;\
	exit 1 ;\
	}
else
KIND=$(shell which kind)
endif

tools: kustomize controller-gen operator-sdk