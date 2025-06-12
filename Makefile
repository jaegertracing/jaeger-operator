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
GOPATH ?= "$(HOME)/go"
GOROOT ?= "$(shell go env GOROOT)"
WATCH_NAMESPACE ?= ""
BIN_DIR ?= bin
FMT_LOG=fmt.log
ECHO ?= @echo $(echo_prefix)
SED ?= "sed"
# Jaeger Operator build variables
OPERATOR_NAME ?= jaeger-operator
IMG_PREFIX ?= quay.io/${USER}
OPERATOR_VERSION ?= "$(shell grep -v '\#' versions.txt | grep operator | awk -F= '{print $$2}')"
VERSION ?=  "$(shell grep operator= versions.txt | awk -F= '{print $$2}')"
IMG ?= ${IMG_PREFIX}/${OPERATOR_NAME}:${VERSION}
BUNDLE_IMG ?= ${IMG_PREFIX}/${OPERATOR_NAME}-bundle:$(addprefix v,${VERSION})
OUTPUT_BINARY ?= "$(BIN_DIR)/jaeger-operator"
VERSION_PKG ?= "github.com/jaegertracing/jaeger-operator/pkg/version"
export JAEGER_VERSION ?= "$(shell grep jaeger= versions.txt | awk -F= '{print $$2}')"
# agent was removed in jaeger 1.62.0, and the new versions of jaeger doesn't distribute the images anymore
# for that reason the last version of the agent is 1.62.0 and is pined here so we can update jaeger and maintain
# the latest agent image.
export JAEGER_AGENT_VERSION ?= "1.62.0"

# Kafka and Kafka Operator variables
STORAGE_NAMESPACE ?= "${shell kubectl get sa default -o jsonpath='{.metadata.namespace}' || oc project -q}"
KAFKA_NAMESPACE ?= "kafka"
KAFKA_VERSION ?= 0.32.0
KAFKA_EXAMPLE ?= "https://raw.githubusercontent.com/strimzi/strimzi-kafka-operator/${KAFKA_VERSION}/examples/kafka/kafka-persistent-single.yaml"
KAFKA_YAML ?= "https://github.com/strimzi/strimzi-kafka-operator/releases/download/${KAFKA_VERSION}/strimzi-cluster-operator-${KAFKA_VERSION}.yaml"
# Prometheus Operator variables
PROMETHEUS_OPERATOR_TAG ?= v0.39.0
PROMETHEUS_BUNDLE ?= https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/${PROMETHEUS_OPERATOR_TAG}/bundle.yaml
# Metrics server variables
METRICS_SERVER_TAG ?= v0.6.1
METRICS_SERVER_YAML ?= https://github.com/kubernetes-sigs/metrics-server/releases/download/${METRICS_SERVER_TAG}/components.yaml
# Ingress controller variables
INGRESS_CONTROLLER_TAG ?= v1.0.1
INGRESS_CONTROLLER_YAML ?= https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-${INGRESS_CONTROLLER_TAG}/deploy/static/provider/kind/deploy.yaml
## Location to install tool dependencies
LOCALBIN ?= $(shell pwd)/bin
# Cert manager version to use
CERTMANAGER_VERSION ?= 1.6.1
CMCTL ?= $(LOCALBIN)/cmctl
# Operator SDK
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk
OPERATOR_SDK_VERSION ?= 1.32.0
# Minimum Kubernetes and OpenShift versions
MIN_KUBERNETES_VERSION ?= 1.19.0
MIN_OPENSHIFT_VERSION ?= 4.12
# Use a KIND cluster for the E2E tests
USE_KIND_CLUSTER ?= true
 # Is Jaeger Operator installed via OLM?
JAEGER_OLM ?= false
# Is Kafka Operator installed via OLM?
KAFKA_OLM ?= false
# Is Prometheus Operator installed via OLM?
PROMETHEUS_OLM ?= false
# Istio binary path and version
ISTIOCTL ?= $(LOCALBIN)/istioctl
# Tools
CRDOC ?= $(LOCALBIN)/crdoc
KIND ?= $(LOCALBIN)/kind
KUSTOMIZE ?= $(LOCALBIN)/kustomize


$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

LD_FLAGS ?= "-X $(VERSION_PKG).version=$(VERSION) -X $(VERSION_PKG).buildDate=$(VERSION_DATE) -X $(VERSION_PKG).defaultJaeger=$(JAEGER_VERSION) -X $(VERSION_PKG).defaultAgent=$(JAEGER_AGENT_VERSION)"

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST ?= $(LOCALBIN)/setup-envtest
ENVTEST_K8S_VERSION = 1.30
# Options for KIND version to use
export KUBE_VERSION ?= 1.30
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
check: install-tools
	$(ECHO) Checking...
	$(VECHO)./.ci/format.sh > $(FMT_LOG)
	$(VECHO)[ ! -s "$(FMT_LOG)" ] || (echo "Go fmt, license check, or import ordering failures, run 'make format'" | cat - $(FMT_LOG) && false)

ensure-generate-is-noop: VERSION=$(OPERATOR_VERSION)
ensure-generate-is-noop: set-image-controller generate bundle
	$(VECHO)# on make bundle config/manager/kustomization.yaml includes changes, which should be ignored for the below check
	$(VECHO)git restore config/manager/kustomization.yaml
	$(VECHO)git diff -s --exit-code api/v1/zz_generated.*.go || (echo "Build failed: a model has been changed but the generated resources aren't up to date. Run 'make generate' and update your PR." && exit 1)
	$(VECHO)git diff -s --exit-code bundle config || (echo "Build failed: the bundle, config files has been changed but the generated bundle, config files aren't up to date. Run 'make bundle' and update your PR." && git diff && exit 1)
	$(VECHO)git diff -s --exit-code docs/api.md || (echo "Build failed: the api.md file has been changed but the generated api.md file isn't up to date. Run 'make api-docs' and update your PR." && git diff && exit 1)


.PHONY: format
format: install-tools
	$(ECHO) Formatting code...
	$(VECHO)./.ci/format.sh

PHONY: lint
lint: install-tools
	$(ECHO) Linting...
	$(VECHO)$(LOCALBIN)/golangci-lint -v run

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: build
build: format
	$(ECHO) Building...
	$(VECHO)./hack/install/install-dependencies.sh
	$(VECHO)${GO_FLAGS} go build -ldflags $(LD_FLAGS) -o $(OUTPUT_BINARY) main.go

.PHONY: docker
docker:
	$(VECHO)[ ! -z "$(PIPELINE)" ] || docker build --build-arg=GOPROXY=${GOPROXY} --build-arg=VERSION=${VERSION} --build-arg=JAEGER_VERSION=${JAEGER_VERSION} --build-arg=JAEGER_AGENT_VERSION=${JAEGER_AGENT_VERSION} --build-arg=TARGETARCH=$(GOARCH) --build-arg VERSION_DATE=${VERSION_DATE}  --build-arg VERSION_PKG=${VERSION_PKG} -t "$(IMG)" . ${DOCKER_BUILD_OPTIONS}

.PHONY: dockerx
dockerx:
	$(VECHO)[ ! -z "$(PIPELINE)" ] || docker buildx build --push --progress=plain --build-arg=VERSION=${VERSION} --build-arg=JAEGER_VERSION=${JAEGER_VERSION} --build-arg=JAEGER_AGENT_VERSION=${JAEGER_AGENT_VERSION} --build-arg=GOPROXY=${GOPROXY} --build-arg VERSION_DATE=${VERSION_DATE} --build-arg VERSION_PKG=${VERSION_PKG} --platform=$(PLATFORMS) $(IMAGE_TAGS) .

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
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -p 1 ${GOTEST_OPTS} ./... -cover -coverprofile=cover.out -ldflags $(LD_FLAGS)

.PHONY: set-node-os-linux
set-node-os-linux:
	# Elasticsearch requires labeled nodes. These labels are by default present in OCP 4.2
	$(VECHO)kubectl label nodes --all kubernetes.io/os=linux --overwrite

cert-manager: cmctl
	# Consider using cmctl to install the cert-manager once install command is not experimental
	kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v${CERTMANAGER_VERSION}/cert-manager.yaml
	$(CMCTL) check api --wait=5m

undeploy-cert-manager:
	kubectl delete --ignore-not-found=true -f https://github.com/jetstack/cert-manager/releases/download/v${CERTMANAGER_VERSION}/cert-manager.yaml

cmctl: $(CMCTL)
$(CMCTL): $(LOCALBIN)
	./hack/install/install-cmctl.sh $(CERTMANAGER_VERSION)

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
	$(VECHO)./hack/install/install-istio.sh
	$(VECHO)${ISTIOCTL} install --set profile=minimal -y

.PHONY: undeploy-istio
undeploy-istio:
	$(VECHO)${ISTIOCTL} manifest generate --set profile=demo | kubectl delete --ignore-not-found=true -f - || true
	$(VECHO)kubectl delete namespace istio-system --ignore-not-found=true || true

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
ifeq ($(KAFKA_OLM),true)
	$(ECHO) Skipping kafka-operator deployment, assuming it has been installed via OperatorHub
else
	$(VECHO)curl --fail --location https://github.com/strimzi/strimzi-kafka-operator/releases/download/0.32.0/strimzi-0.32.0.tar.gz --output tests/_build/kafka-operator.tar.gz --create-dirs
	$(VECHO)tar xf tests/_build/kafka-operator.tar.gz
	$(VECHO)${SED} -i 's/namespace: .*/namespace: ${KAFKA_NAMESPACE}/' strimzi-${KAFKA_VERSION}/install/cluster-operator/*RoleBinding*.yaml
	$(VECHO)kubectl create -f strimzi-${KAFKA_VERSION}/install/cluster-operator/020-RoleBinding-strimzi-cluster-operator.yaml -n ${KAFKA_NAMESPACE}
	$(VECHO)kubectl create -f strimzi-${KAFKA_VERSION}/install/cluster-operator/023-RoleBinding-strimzi-cluster-operator.yaml -n ${KAFKA_NAMESPACE}
	$(VECHO)kubectl create -f strimzi-${KAFKA_VERSION}/install/cluster-operator/031-RoleBinding-strimzi-cluster-operator-entity-operator-delegation.yaml -n ${KAFKA_NAMESPACE}
	$(VECHO)kubectl apply -f strimzi-${KAFKA_VERSION}/install/cluster-operator/ -n ${KAFKA_NAMESPACE}
endif

.PHONY: undeploy-kafka-operator
undeploy-kafka-operator:
ifeq ($(KAFKA_OLM),true)
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
ifeq ($(PROMETHEUS_OLM),true)
	$(ECHO) Skipping prometheus-operator deployment, assuming it has been installed via OperatorHub
else
	$(VECHO)kubectl apply -f ${PROMETHEUS_BUNDLE}
endif

.PHONY: undeploy-prometheus-operator
undeploy-prometheus-operator:
ifeq ($(PROMETHEUS_OLM),true)
	$(ECHO) Skipping prometheus-operator undeployment, as it should have been installed via OperatorHub
else
	$(VECHO)kubectl delete -f ${PROMETHEUS_BUNDLE} --ignore-not-found=true || true
endif

.PHONY: clean
clean: undeploy-kafka undeploy-prometheus-operator undeploy-istio undeploy-cert-manager
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
all: check format lint build test

.PHONY: ci
ci: install-tools ensure-generate-is-noop check format lint build unit-tests

##@ Deployment

ignore-not-found ?= false

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	kubectl create namespace observability 2>&1 | grep -v "already exists" || true
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	./hack/enable-operator-features.sh
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
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
	$(VECHO)docker run --rm  -v "${PWD}:/app" pavolloffay/gch:latest --oauth-token ${OAUTH_TOKEN} --branch main --owner jaegertracing --repo jaeger-operator


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(VECHO)./hack/install/install-controller-gen.sh

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(ENVTEST) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: bundle
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	$(SED) -i "s#containerImage: quay.io/jaegertracing/jaeger-operator:$(OPERATOR_VERSION)#containerImage: quay.io/jaegertracing/jaeger-operator:$(VERSION)#g" config/manifests/bases/jaeger-operator.clusterserviceversion.yaml
	$(SED) -i 's/minKubeVersion: .*/minKubeVersion: $(MIN_KUBERNETES_VERSION)/' config/manifests/bases/jaeger-operator.clusterserviceversion.yaml
	$(SED) -i 's/com.redhat.openshift.versions=.*/com.redhat.openshift.versions=v$(MIN_OPENSHIFT_VERSION)/' bundle.Dockerfile
	$(SED) -i 's/com.redhat.openshift.versions: .*/com.redhat.openshift.versions: v$(MIN_OPENSHIFT_VERSION)/' bundle/metadata/annotations.yaml

	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --manifests --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	$(OPERATOR_SDK) bundle validate ./bundle
	./hack/ignore-createdAt-bundle.sh

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	docker push $(BUNDLE_IMG)

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
# When the Kind cluster is not created by Kuttl, the kindContainers parameter
# from kuttl-tests.yaml has not effect so, it is needed to load the container
# images here.
	$(VECHO)$(KIND) create cluster --config $(KIND_CONFIG) 2>&1 | grep -v "already exists" || true
# Install metrics-server for HPA
	$(ECHO)"Installing the metrics-server in the kind cluster"
	$(VECHO)kubectl apply -f $(METRICS_SERVER_YAML)
	$(VECHO)kubectl patch deployment -n kube-system metrics-server --type "json" -p '[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": --kubelet-insecure-tls}]'
# Install the ingress-controller
	$(ECHO)"Installing the Ingress controller in the kind cluster"
	$(VECHO)kubectl apply -f $(INGRESS_CONTROLLER_YAML)
# Check the deployments were done properly
	$(ECHO)"Checking the metrics-server was deployed properly"
	$(VECHO)kubectl wait --for=condition=available deployment/metrics-server -n kube-system --timeout=5m
	$(ECHO)"Checking the Ingress controller deployment was done successfully"
	$(VECHO)kubectl wait --for=condition=available deployment ingress-nginx-controller -n ingress-nginx --timeout=5m
else
	$(ECHO)"KIND cluster creation disabled. Skipping..."
endif

stop-kind:
	$(ECHO)"Stopping the kind cluster"
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
	$(VECHO)./hack/install/install-golangci-lint.sh
	$(VECHO)./hack/install/install-goimports.sh

.PHONY: kustomize
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): $(LOCALBIN)
	./hack/install/install-kustomize.sh

.PHONY: kind
kind: $(KIND)
$(KIND): $(LOCALBIN)
	./hack/install/install-kind.sh

.PHONY: prepare-release
prepare-release:
	$(VECHO)./.ci/prepare-release.sh

scorecard-tests: operator-sdk
	echo "Operator sdk is $(OPERATOR_SDK)"
	$(OPERATOR_SDK) scorecard bundle -w 10m || (echo "scorecard test failed" && exit 1)

scorecard-tests-local: kind
	$(VECHO)$(KIND) create cluster --config $(KIND_CONFIG) 2>&1 | grep -v "already exists" || true
	$(VECHO)docker pull $(SCORECARD_TEST_IMG)
	$(VECHO)$(KIND) load docker-image $(SCORECARD_TEST_IMG)
	$(VECHO)kubectl wait --timeout=5m --for=condition=available deployment/coredns -n kube-system
	$(VECHO)$(MAKE) scorecard-tests

.PHONY: operator-sdk
operator-sdk: $(OPERATOR_SDK)
$(OPERATOR_SDK): $(LOCALBIN)
	test -s $(OPERATOR_SDK) || curl -sLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk_`go env GOOS`_`go env GOARCH`
	@chmod +x $(OPERATOR_SDK)

api-docs: crdoc kustomize
	@{ \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ; \
	$(KUSTOMIZE) build config/crd -o $$TMP_DIR/crd-output.yaml ;\
	$(CRDOC) --resources $$TMP_DIR/crd-output.yaml --output docs/api.md ;\
	}

.PHONY: crdoc
crdoc: $(CRDOC)
$(CRDOC): $(LOCALBIN)
	test -s $(CRDOC) || GOBIN=$(LOCALBIN) go install fybrik.io/crdoc@v0.5.2
	@chmod +x $(CRDOC)
