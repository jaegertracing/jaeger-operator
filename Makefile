include tests/e2e/Makefile

# When the VERBOSE variable is set to 1, all the commands are shown
ifeq ("$(VERBOSE)","true")
echo_prefix=">>>>"
else
VECHO = @
endif

VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
GOARCH ?= $(go env GOARCH)
GOOS ?= $(go env GOOS)
GO_FLAGS ?= GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 GO111MODULE=on
WATCH_NAMESPACE ?= ""
BIN_DIR ?= bin
FMT_LOG=fmt.log

OPERATOR_NAME ?= jaeger-operator
IMG_PREFIX ?= quay.io/${USER}
OPERATOR_VERSION ?= "$(shell grep -v '\#' versions.txt | grep operator | awk -F= '{print $$2}')"
VERSION ?= "$(shell git describe --tags | sed 's/^v//')"
IMG ?= ${IMG_PREFIX}/${OPERATOR_NAME}:${VERSION}
BUNDLE_IMG ?= ${IMG_PREFIX}/${OPERATOR_NAME}-bundle:$(addprefix v,${VERSION})
OUTPUT_BINARY ?= "$(BIN_DIR)/jaeger-operator"
VERSION_PKG ?= "github.com/jaegertracing/jaeger-operator/pkg/version"
JAEGER_VERSION ?= "$(shell grep jaeger= versions.txt | awk -F= '{print $$2}')"
# Kafka and kafka operator variables
STORAGE_NAMESPACE ?= "${shell kubectl get sa default -o jsonpath='{.metadata.namespace}' || oc project -q}"
KAFKA_NAMESPACE ?= "kafka"
KAFKA_EXAMPLE ?= "https://raw.githubusercontent.com/strimzi/strimzi-kafka-operator/0.23.0/examples/kafka/kafka-persistent-single.yaml"
KAFKA_YAML ?= "https://github.com/strimzi/strimzi-kafka-operator/releases/download/0.23.0/strimzi-cluster-operator-0.23.0.yaml"
# Istio binary path and version
ISTIO_VERSION ?= 1.11.2
ISTIO_PATH = ./tests/_build/
ISTIOCTL="${ISTIO_PATH}istio/bin/istioctl"
GOPATH ?= "$(HOME)/go"
GOROOT ?= "$(shell go env GOROOT)"
ECHO ?= @echo $(echo_prefix)
SED ?= "sed"
CERTMANAGER_VERSION ?= 1.6.1
OPERATOR_SDK_VERSION ?= 1.17.0

USE_KIND_CLUSTER ?= true
export OLM ?= false
SKIP_ES_EXTERNAL ?= false

PROMETHEUS_OPERATOR_TAG ?= v0.39.0
PROMETHEUS_BUNDLE ?= https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/${PROMETHEUS_OPERATOR_TAG}/bundle.yaml

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

LD_FLAGS ?= "-X $(VERSION_PKG).version=$(VERSION) -X $(VERSION_PKG).buildDate=$(VERSION_DATE) -X $(VERSION_PKG).defaultJaeger=$(JAEGER_VERSION)"

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23
# Options for kuttl testing
KUBE_VERSION ?= 1.20
KIND_CONFIG ?= kind-$(KUBE_VERSION).yaml

SCORECARD_TEST_IMG ?= quay.io/operator-framework/scorecard-test:v$(OPERATOR_SDK_VERSION)

.DEFAULT_GOAL := build

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:maxDescLen=0,generateEmbeddedObjectMeta=true"

# If we are running in CI, run go test in verbose mode
ifeq (,$(CI))
GOTEST_OPTS=
else
GOTEST_OPTS=-v
endif

all: manager

.PHONY: check
check:
	$(ECHO) Checking...
	$(VECHO)GOPATH=${GOPATH} .ci/format.sh > $(FMT_LOG)
	$(VECHO)[ ! -s "$(FMT_LOG)" ] || (echo "Go fmt, license check, or import ordering failures, run 'make format'" | cat - $(FMT_LOG) && false)

ensure-generate-is-noop: VERSION=$(OPERATOR_VERSION)
ensure-generate-is-noop: set-image-controller generate bundle
	$(VECHO)# on make bundle config/manager/kustomization.yaml includes changes, which should be ignored for the below check
	$(VECHO)git restore config/manager/kustomization.yaml
	$(VECHO)git diff -s --exit-code api/v1/zz_generated.*.go || (echo "Build failed: a model has been changed but the generated resources aren't up to date. Run 'make generate' and update your PR." && exit 1)
	$(VECHO)git diff -s --exit-code bundle config || (echo "Build failed: the bundle, config files has been changed but the generated bundle, config files aren't up to date. Run 'make bundle' and update your PR." && git diff && exit 1)
	$(VECHO)git diff -s --exit-code docs/api.md || (echo "Build failed: the api.md file has been changed but the generated api.md file isn't up to date. Run 'make api-docs' and update your PR." && git diff && exit 1)


.PHONY: format
format:
	$(ECHO) Formatting code...
	$(VECHO)GOPATH=${GOPATH} .ci/format.sh

PHONY: lint
lint:
	$(ECHO) Linting...
	$(VECHO)GOPATH=${GOPATH} ./.ci/lint.sh

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: security
security:
	$(ECHO) Security...
	$(VECHO)${GOPATH}/bin/gosec -quiet -exclude=G104 ./... 2>/dev/null

.PHONY: build
build: format
	$(ECHO) Building...
	$(VECHO)${GO_FLAGS} go build -ldflags $(LD_FLAGS) -o $(OUTPUT_BINARY) main.go

.PHONY: docker
docker:
	$(VECHO)[ ! -z "$(PIPELINE)" ] || docker build --build-arg=GOPROXY=${GOPROXY} --build-arg=JAEGER_VERSION=${JAEGER_VERSION} --build-arg=TARGETARCH=$(GOARCH) --build-arg VERSION_DATE=${VERSION_DATE}  --build-arg VERSION_PKG=${VERSION_PKG} -t "$(IMG)" . ${DOCKER_BUILD_OPTIONS}

.PHONY: dockerx
dockerx:
	$(VECHO)[ ! -z "$(PIPELINE)" ] || docker buildx build --push --progress=plain --build-arg=JAEGER_VERSION=${JAEGER_VERSION} --build-arg=GOPROXY=${GOPROXY} --build-arg VERSION_DATE=${VERSION_DATE} --build-arg VERSION_PKG=${VERSION_PKG} --platform=$(PLATFORMS) $(IMAGE_TAGS) .

.PHONY: push
push:
ifeq ($(CI),true)
	$(ECHO) Skipping push, as the build is running within a CI environment
else
	$(ECHO) "Pushing image $(IMG)..."
	$(VECHO)docker push $(IMG) > /dev/null
endif

.PHONY: unit-tests
unit-tests: envtest
	@echo Running unit tests...
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ${GOTEST_OPTS} ./... -cover -coverprofile=cover.out -ldflags $(LD_FLAGS)

.PHONY: set-max-map-count
set-max-map-count:
	# This is not required in OCP 4.1. The node tuning operator configures the property automatically
	# when label tuned.openshift.io/elasticsearch=true label is present on the ES pod. The label
	# is configured by ES operator.
	$(VECHO)minishift ssh -- 'sudo sysctl -w vm.max_map_count=262144' > /dev/null 2>&1 || true

.PHONY: set-node-os-linux
set-node-os-linux:
	# Elasticsearch requires labeled nodes. These labels are by default present in OCP 4.2
	$(VECHO)kubectl label nodes --all kubernetes.io/os=linux --overwrite

cert-manager: cmctl
	# Consider using cmctl to install the cert-manager once install command is not experimental
	kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v${CERTMANAGER_VERSION}/cert-manager.yaml
	cmctl check api --wait=5m

cmctl:
ifeq (, $(shell which cmctl))
	@{ \
	curl -L -o /tmp/cmctl.tar.gz https://github.com/jetstack/cert-manager/releases/download/v$(CERTMANAGER_VERSION)/cmctl-`go env GOOS`-`go env GOARCH`.tar.gz ;\
	cd /tmp ;\
	tar xzf cmctl.tar.gz ;\
	mv cmctl $(GOBIN) ;\
	}
CTL=$(GOBIN)/cmctl
else
CTL=$(shell which cmctl)
endif

.PHONY: es
es: storage
ifeq ($(SKIP_ES_EXTERNAL),true)
	$(ECHO) Skipping creation of external Elasticsearch instance
else
	$(VECHO)kubectl create -f ./tests/elasticsearch.yml --namespace $(STORAGE_NAMESPACE) 2>&1 | grep -v "already exists" || true
endif

.PHONY: istio
istio:
	$(ECHO) Install istio with minimal profile
	$(VECHO)mkdir -p ${ISTIO_PATH}
	[ -f "${ISTIOCTL}" ] || (curl -L https://istio.io/downloadIstio | ISTIO_VERSION=${ISTIO_VERSION} TARGET_ARCH=x86_64 sh - && mv ./istio-${ISTIO_VERSION} ${ISTIO_PATH}/istio/)
	$(VECHO)${ISTIOCTL} install --set profile=minimal -y

.PHONY: undeploy-istio
undeploy-istio:
	$(VECHO)[ -f "${ISTIOCTL}" ] && (${ISTIOCTL} manifest generate --set profile=demo | kubectl delete --ignore-not-found=true -f -) || true
	$(VECHO)kubectl delete namespace istio-system --ignore-not-found=true || true
	$(VECHO)rm -rf ${ISTIO_PATH}

.PHONY: cassandra
cassandra: storage
	$(VECHO)kubectl create -f ./tests/cassandra.yml --namespace $(STORAGE_NAMESPACE) 2>&1 | grep -v "already exists" || true

.PHONY: storage
storage:
	$(ECHO) Creating namespace $(STORAGE_NAMESPACE)
	$(VECHO)kubectl create namespace $(STORAGE_NAMESPACE) 2>&1 | grep -v "already exists" || true

.PHONY: deploy-kafka-operator
deploy-kafka-operator:
	$(ECHO) Creating namespace $(KAFKA_NAMESPACE)
	$(VECHO)kubectl create namespace $(KAFKA_NAMESPACE) 2>&1 | grep -v "already exists" || true
ifeq ($(OLM),true)
	$(ECHO) Skipping kafka-operator deployment, assuming it has been installed via OperatorHub
else
	$(VECHO)kubectl create clusterrolebinding strimzi-cluster-operator-namespaced --clusterrole=strimzi-cluster-operator-namespaced --serviceaccount ${KAFKA_NAMESPACE}:strimzi-cluster-operator 2>&1 | grep -v "already exists" || true
	$(VECHO)kubectl create clusterrolebinding strimzi-cluster-operator-entity-operator-delegation --clusterrole=strimzi-entity-operator --serviceaccount ${KAFKA_NAMESPACE}:strimzi-cluster-operator 2>&1 | grep -v "already exists" || true
	$(VECHO)kubectl create clusterrolebinding strimzi-cluster-operator-topic-operator-delegation --clusterrole=strimzi-topic-operator --serviceaccount ${KAFKA_NAMESPACE}:strimzi-cluster-operator 2>&1 | grep -v "already exists" || true
	$(VECHO)curl --fail --location $(KAFKA_YAML) --output tests/_build/kafka-operator.yaml --create-dirs
	$(VECHO)${SED} -i 's/namespace: .*/namespace: $(KAFKA_NAMESPACE)/' tests/_build/kafka-operator.yaml
	$(VECHO) kubectl -n $(KAFKA_NAMESPACE) apply -f tests/_build/kafka-operator.yaml | grep -v "already exists" || true
	$(VECHO)kubectl set env deployment strimzi-cluster-operator -n ${KAFKA_NAMESPACE} STRIMZI_NAMESPACE="*"
endif

.PHONY: undeploy-kafka-operator
undeploy-kafka-operator:
ifeq ($(OLM),true)
	$(ECHO) Skiping kafka-operator undeploy
else
	$(VECHO)kubectl delete --namespace $(KAFKA_NAMESPACE) -f tests/_build/kafka-operator.yaml --ignore-not-found=true 2>&1 || true
	$(VECHO)kubectl delete clusterrolebinding strimzi-cluster-operator-namespaced --ignore-not-found=true || true
	$(VECHO)kubectl delete clusterrolebinding strimzi-cluster-operator-entity-operator-delegation --ignore-not-found=true || true
	$(VECHO)kubectl delete clusterrolebinding strimzi-cluster-operator-topic-operator-delegation --ignore-not-found=true || true
endif

.PHONY: kafka
kafka: deploy-kafka-operator
ifeq ($(SKIP_KAFKA),true)
	$(ECHO) Skipping Kafka/external ES related tests
else
	$(ECHO) Creating namespace $(KAFKA_NAMESPACE)
	$(VECHO)mkdir -p tests/_build/
	$(VECHO)kubectl create namespace $(KAFKA_NAMESPACE) 2>&1 | grep -v "already exists" || true
	$(VECHO)curl --fail --location $(KAFKA_EXAMPLE) --output tests/_build/kafka-example.yaml --create-dirs
	$(VECHO)${SED} -i 's/size: 100Gi/size: 10Gi/g' tests/_build/kafka-example.yaml
	$(VECHO)kubectl -n $(KAFKA_NAMESPACE) apply --dry-run=client -f  tests/_build/kafka-example.yaml
	$(VECHO)kubectl -n $(KAFKA_NAMESPACE) apply -f tests/_build/kafka-example.yaml 2>&1 | grep -v "already exists" || true
endif

.PHONY: undeploy-kafka
undeploy-kafka: undeploy-kafka-operator
	$(VECHO)kubectl delete --namespace $(KAFKA_NAMESPACE) -f tests/_build/kafka-example.yaml 2>&1 || true


.PHONY: deploy-prometheus-operator
deploy-prometheus-operator:
ifeq ($(OLM),true)
	$(ECHO) Skipping prometheus-operator deployment, assuming it has been installed via OperatorHub
else
	$(VECHO)kubectl apply -f ${PROMETHEUS_BUNDLE}
endif

.PHONY: undeploy-prometheus-operator
undeploy-prometheus-operator:
ifeq ($(OLM),true)
	$(ECHO) Skipping prometheus-operator undeployment, as it should have been installed via OperatorHub
else
	$(VECHO)kubectl delete -f ${PROMETHEUS_BUNDLE} --ignore-not-found=true || true
endif

.PHONY: clean
clean: undeploy-kafka undeploy-prometheus-operator undeploy-istio
	$(VECHO)kubectl delete namespace $(KAFKA_NAMESPACE) --ignore-not-found=true 2>&1 || true
	$(VECHO)if [ -d tests/_build ]; then rm -rf tests/_build ; fi
	$(VECHO)kubectl delete -f ./tests/cassandra.yml --ignore-not-found=true -n $(STORAGE_NAMESPACE) || true
	$(VECHO)kubectl delete -f ./tests/elasticsearch.yml --ignore-not-found=true -n $(STORAGE_NAMESPACE) || true

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen api-docs ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: test
test: unit-tests run-e2e-tests

.PHONY: all
all: check format lint security build test

.PHONY: ci
ci: ensure-generate-is-noop check format lint security build unit-tests

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: operatorhub
operatorhub: check-operatorhub-pr-template
	$(VECHO)./.ci/operatorhub.sh

.PHONY: check-operatorhub-pr-template
check-operatorhub-pr-template:
	$(VECHO)curl https://raw.githubusercontent.com/operator-framework/community-operators/master/docs/pull_request_template.md -o .ci/.operatorhub-pr-template.md -s > /dev/null 2>&1
	$(VECHO)git diff -s --exit-code .ci/.operatorhub-pr-template.md || (echo "Build failed: the PR template for OperatorHub has changed. Sync it and try again." && exit 1)

.PHONY: changelog
changelog:
	$(ECHO) "Set env variable OAUTH_TOKEN before invoking, https://github.com/settings/tokens/new?description=GitHub%20Changelog%20Generator%20token"
	$(VECHO)docker run --rm  -v "${PWD}:/app" pavolloffay/gch:latest --oauth-token ${OAUTH_TOKEN} --owner jaegertracing --repo jaeger-operator


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)

ENVTEST = $(shell pwd)/bin/setup-envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell git rev-parse --show-toplevel)
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
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --manifests --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle

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

.PHONY: start-kind
start-kind: kind
ifeq ($(USE_KIND_CLUSTER),true)
	$(ECHO) Starting KIND cluster...
# Instead of letting KUTTL create the Kind cluster (using the CLI or in the kuttl-tests.yaml
# file), the cluster is created here. There are multiple reasons to do this:
# 	* The kubectl command will not work outside KUTTL
#	* Some KUTTL versions are not able to start properly a Kind cluster
#	* The cluster will be removed after running KUTTL (this can be disabled). Sometimes,
#		the cluster teardown is not done properly and KUTTL can not be run with the --start-kind flag
# When the Kind cluster is not created by Kuttl, the
# kindContainers parameter from kuttl-tests.yaml has not effect so, it is needed to load the
# container images here.
	$(VECHO)$(KIND) create cluster --config $(KIND_CONFIG) 2>&1 | grep -v "already exists" || true
	$(VECHO)kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.0.1/deploy/static/provider/kind/deploy.yaml
else
	$(ECHO)KIND cluster creation disabled. Skipping...
endif

stop-kind:
	$(ECHO)Stopping the kind cluster
	$(VECHO)kind delete cluster

.PHONY: install-git-hooks
install-git-hooks:
	$(VECHO)cp scripts/git-hooks/pre-commit .git/hooks

# Generates the released manifests
release-artifacts: set-image-controller
	mkdir -p dist
	$(KUSTOMIZE) build config/default -o dist/jaeger-operator.yaml

# Set the controller image parameters
set-image-controller: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}

.PHONY: tools
tools: kustomize controller-gen operator-sdk

.PHONY: install-tools
install-tools: operator-sdk
	$(VECHO)${GO_FLAGS} ./.ci/vgot.sh \
		golang.org/x/lint/golint \
		golang.org/x/tools/cmd/goimports \
		github.com/securego/gosec/cmd/gosec@v0.0.0-20191008095658-28c1128b7336

.PHONY: kustomize
kustomize:
	./hack/install/install-kustomize.sh
	$(eval KUSTOMIZE=$(shell echo ${PWD}/bin/kustomize))

.PHONY: kuttl
kuttl:
	./hack/install/install-kuttl.sh
	$(eval KUTTL=$(shell echo ${PWD}/bin/kubectl-kuttl))

.PHONY: kind
kind:
	./hack/install/install-kind.sh
	$(eval KIND=$(shell echo ${PWD}/bin/kind))

.PHONY: prepare-release
prepare-release:
	$(VECHO)./.ci/prepare-release.sh

scorecard-tests: operator-sdk
	echo "Operator sdk is " $(OPERATOR_SDK)
	$(OPERATOR_SDK) scorecard bundle -w 600s || (echo "scorecard test failed" && exit 1)

scorecard-tests-local: kind
	$(VECHO)$(KIND) create cluster --config $(KIND_CONFIG) 2>&1 | grep -v "already exists" || true
	$(VECHO)docker pull $(SCORECARD_TEST_IMG)
	$(VECHO)$(KIND) load docker-image $(SCORECARD_TEST_IMG)
	$(VECHO)kubectl wait --timeout=5m --for=condition=available deployment/coredns -n kube-system
	$(VECHO)$(MAKE) scorecard-tests

OPERATOR_SDK = $(shell pwd)/bin/operator-sdk
.PHONY: operator-sdk
operator-sdk:
	@{ \
	set -e ;\
	[ -d bin ] || mkdir bin ;\
	curl -L -o $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk_`go env GOOS`_`go env GOARCH`;\
	chmod +x $(OPERATOR_SDK) ;\
	}

BIN_LOCAL = $(shell pwd)/bin
CRDOC = $(BIN_LOCAL)/crdoc
api-docs: crdoc kustomize
	@{ \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ; \
	$(KUSTOMIZE) build config/crd -o $$TMP_DIR/crd-output.yaml ;\
	$(CRDOC) --resources $$TMP_DIR/crd-output.yaml --output docs/api.md ;\
	}


# Find or download crdoc
crdoc:
ifeq (, $(shell which $(CRDOC)))
	@GOBIN=$(BIN_LOCAL) go install fybrik.io/crdoc@v0.5.2
endif
