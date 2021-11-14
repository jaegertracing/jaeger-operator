# When the VERBOSE variable is set to 1, all the commands are shown
ifeq ("$(VERBOSE)","1")
echo_prefix=">>>>"
else
VECHO = @
endif

# Current Operator version
VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x
GOARCH ?= $(go env GOARCH)
GOOS ?= $(go env GOOS)
GO_FLAGS ?= GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 GO111MODULE=on
BIN_DIR ?= bin
FMT_LOG=fmt.log

# Image URL to use all building/pushing image targets
OPERATOR_NAME ?= jaeger-operator
IMG_PREFIX ?= quay.io/${USER}
OPERATOR_VERSION ?= "$(shell grep -v '\#' versions.txt | grep operator | awk -F= '{print $$2}')"
VERSION ?= "$(shell git describe --tags | sed 's/^v//')"
IMG ?= ${IMG_PREFIX}/${OPERATOR_NAME}:$(addprefix v,${VERSION})
BUNDLE_IMG ?= ${IMG_PREFIX}/${OPERATOR_NAME}-bundle:$(addprefix v,${VERSION})
OUTPUT_BINARY ?= "$(BIN_DIR)/$(OPERATOR_NAME)"
VERSION_PKG ?= "github.com/jaegertracing/jaeger-operator/pkg/version"
JAEGER_VERSION ?= "$(shell grep jaeger= versions.txt | awk -F= '{print $$2}')"
# Kafka and kafka operator variables
KAFKA_NAMESPACE ?= "kafka"
KAFKA_EXAMPLE ?= "https://raw.githubusercontent.com/strimzi/strimzi-kafka-operator/0.23.0/examples/kafka/kafka-persistent-single.yaml"
KAFKA_YAML ?= "https://github.com/strimzi/strimzi-kafka-operator/releases/download/0.23.0/strimzi-cluster-operator-0.23.0.yaml"
ES_OPERATOR_NAMESPACE ?= openshift-logging
ES_OPERATOR_BRANCH ?= release-4.4
ES_OPERATOR_IMAGE ?= quay.io/openshift/origin-elasticsearch-operator:4.4
# Istio binary path and version
ISTIO_VERSION ?= 1.11.2
ISTIOCTL="./tests/_build/istio/istio/bin/istioctl"
GOPATH ?= "$(HOME)/go"
GOROOT ?= "$(shell go env GOROOT)"

ECHO ?= @echo $(echo_prefix)
SED ?= "sed"

PROMETHEUS_OPERATOR_TAG ?= v0.39.0
PROMETHEUS_BUNDLE ?= https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/${PROMETHEUS_OPERATOR_TAG}/bundle.yaml

LD_FLAGS ?= "-X $(VERSION_PKG).version=$(VERSION) -X $(VERSION_PKG).buildDate=$(VERSION_DATE) -X $(VERSION_PKG).defaultJaeger=$(JAEGER_VERSION)"

# Options for kuttl testing
KUBE_VERSION ?= 1.20
KIND_CONFIG ?= kind-$(KUBE_VERSION).yaml

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false,maxDescLen=0,generateEmbeddedObjectMeta=true"

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
ensure-generate-is-noop: USER=jaegertracing
ensure-generate-is-noop: set-image-controller generate bundle
	@# on make bundle config/manager/kustomization.yaml includes changes, which should be ignored for the below check
	@git restore config/manager/kustomization.yaml
	@git diff -s --exit-code api/v1/zz_generated.*.go || (echo "Build failed: a model has been changed but the generated resources aren't up to date. Run 'make generate' and update your PR." && exit 1)
	@git diff -s --exit-code bundle config || (echo "Build failed: the bundle, config files has been changed but the generated bundle, config files aren't up to date. Run 'make bundle' and update your PR." && git diff && exit 1)


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
	$(MAKE) gobuild

.PHONY: gobuild
gobuild:
	$(ECHO) Building...
	$(VECHO)${GO_FLAGS} go build -ldflags $(LD_FLAGS) -o $(OUTPUT_BINARY) main.go 

.PHONY: docker
docker:
	$(VECHO)[ ! -z "$(PIPELINE)" ] || docker build --build-arg=GOPROXY=${GOPROXY} --build-arg=JAEGER_VERSION=${JAEGER_VERSION} --build-arg=TARGETARCH=$(GOARCH) --build-arg VERSION_DATE=${VERSION_DATE}  --build-arg VERSION_PKG=${VERSION_PKG} -t "$(IMG)" .

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
unit-tests:
	@echo Running unit tests...
	go test ${GOTEST_OPTS} ./... -cover -coverprofile=cover.out -ldflags $(LD_FLAGS)

.PHONY: run
run: manifests generate format vet
	$(VECHO)rm -rf /tmp/_cert*
	go run -ldflags ${LD_FLAGS} ./main.go

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

.PHONY: deploy-es-operator
deploy-es-operator: set-node-os-linux set-max-map-count deploy-prometheus-operator
ifeq ($(OLM),true)
	$(ECHO) Skipping es-operator deployment, assuming it has been installed via OperatorHub
else
	$(VECHO)kubectl create namespace ${ES_OPERATOR_NAMESPACE} 2>&1 | grep -v "already exists" || true
	$(VECHO)kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/01-service-account.yaml -n ${ES_OPERATOR_NAMESPACE}
	$(VECHO)kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/02-role.yaml
	$(VECHO)kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/03-role-bindings.yaml
	$(VECHO)kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/04-crd.yaml -n ${ES_OPERATOR_NAMESPACE}
	$(VECHO)kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/05-deployment.yaml -n ${ES_OPERATOR_NAMESPACE}
	$(VECHO)kubectl set image deployment/elasticsearch-operator elasticsearch-operator=${ES_OPERATOR_IMAGE} -n ${ES_OPERATOR_NAMESPACE}
endif

.PHONY: undeploy-es-operator
undeploy-es-operator:
ifeq ($(OLM),true)
	$(ECHO) Skipping es-operator undeployment, as it should have been installed via OperatorHub
else
	$(VECHO)kubectl delete -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/05-deployment.yaml -n ${ES_OPERATOR_NAMESPACE} --ignore-not-found=true || true
	$(VECHO)kubectl delete -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/04-crd.yaml -n ${ES_OPERATOR_NAMESPACE} --ignore-not-found=true || true
	$(VECHO)kubectl delete -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/03-role-bindings.yaml --ignore-not-found=true || true
	$(VECHO)kubectl delete -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/02-role.yaml --ignore-not-found=true || true
	$(VECHO)kubectl delete -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/01-service-account.yaml -n ${ES_OPERATOR_NAMESPACE} --ignore-not-found=true || true
	$(VECHO)kubectl delete namespace ${ES_OPERATOR_NAMESPACE} --ignore-not-found=true 2>&1 || true
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
	$(VECHO)mkdir -p deploy/test
	$(VECHO)[ -f "${ISTIOCTL}" ] || (curl -L https://istio.io/downloadIstio | ISTIO_VERSION=${ISTIO_VERSION} TARGET_ARCH=x86_64 sh - && mv ./istio-${ISTIO_VERSION} ./deploy/test/istio)
	$(VECHO)${ISTIOCTL} install --set profile=minimal -y

.PHONY: undeploy-istio
undeploy-istio:
	$(VECHO)[ -f "${ISTIOCTL}" ] && (${ISTIOCTL} manifest generate --set profile=demo | kubectl delete --ignore-not-found=true -f -) || true
	$(VECHO)kubectl delete namespace istio-system --ignore-not-found=true || true
	$(VECHO)rm -rf deploy/test/istio

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
	$(VECHO)curl --fail --location $(KAFKA_YAML) --output deploy/test/kafka-operator.yaml --create-dirs
	$(VECHO)${SED} -i 's/namespace: .*/namespace: $(KAFKA_NAMESPACE)/' deploy/test/kafka-operator.yaml
	$(VECHO) kubectl -n $(KAFKA_NAMESPACE) apply -f deploy/test/kafka-operator.yaml | grep -v "already exists" || true
	$(VECHO)kubectl set env deployment strimzi-cluster-operator -n ${KAFKA_NAMESPACE} STRIMZI_NAMESPACE="*"
endif

.PHONY: undeploy-kafka-operator
undeploy-kafka-operator:
ifeq ($(OLM),true)
	$(ECHO) Skiping kafka-operator undeploy
else
	$(VECHO)kubectl delete --namespace $(KAFKA_NAMESPACE) -f deploy/test/kafka-operator.yaml --ignore-not-found=true 2>&1 || true
	$(VECHO)kubectl delete clusterrolebinding strimzi-cluster-operator-namespaced --ignore-not-found=true || true
	$(VECHO)kubectl delete clusterrolebinding strimzi-cluster-operator-entity-operator-delegation --ignore-not-found=true || true
	$(VECHO)kubectl delete clusterrolebinding strimzi-cluster-operator-topic-operator-delegation --ignore-not-found=true || true
endif
	$(VECHO)kubectl delete namespace $(KAFKA_NAMESPACE) --ignore-not-found=true 2>&1 || true

.PHONY: kafka
kafka: deploy-kafka-operator
ifeq ($(SKIP_KAFKA),true)
	$(ECHO) Skipping Kafka/external ES related tests
else
	$(ECHO) Creating namespace $(KAFKA_NAMESPACE)
	$(VECHO)kubectl create namespace $(KAFKA_NAMESPACE) 2>&1 | grep -v "already exists" || true
	$(VECHO)curl --fail --location $(KAFKA_EXAMPLE) --output deploy/test/kafka-example.yaml --create-dirs
	$(VECHO)${SED} -i 's/size: 100Gi/size: 10Gi/g' deploy/test/kafka-example.yaml
	$(VECHO)kubectl -n $(KAFKA_NAMESPACE) apply --dry-run=true -f deploy/test/kafka-example.yaml
	$(VECHO)kubectl -n $(KAFKA_NAMESPACE) apply -f deploy/test/kafka-example.yaml 2>&1 | grep -v "already exists" || true
endif

.PHONY: undeploy-kafka
undeploy-kafka: undeploy-kafka-operator
	$(VECHO)kubectl delete --namespace $(KAFKA_NAMESPACE) -f deploy/test/kafka-example.yaml 2>&1 || true


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
clean: undeploy-kafka undeploy-es-operator undeploy-prometheus-operator undeploy-istio
	$(VECHO)rm -f deploy/test/*.yaml
	$(VECHO)if [ -d deploy/test ]; then rmdir deploy/test ; fi
	$(VECHO)kubectl delete -f ./tests/cassandra.yml --ignore-not-found=true -n $(STORAGE_NAMESPACE) || true
	$(VECHO)kubectl delete -f ./tests/elasticsearch.yml --ignore-not-found=true -n $(STORAGE_NAMESPACE) || true
	$(VECHO)kubectl delete -f deploy/crds/jaegertracing.io_jaegers_crd.yaml --ignore-not-found=true || true
	$(VECHO)kubectl delete -f deploy/operator.yaml --ignore-not-found=true || true
	$(VECHO)kubectl delete -f deploy/role_binding.yaml --ignore-not-found=true || true
	$(VECHO)kubectl delete -f deploy/role.yaml --ignore-not-found=true || true
	$(VECHO)kubectl delete -f deploy/service_account.yaml --ignore-not-found=true || true

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: test
test: unit-tests e2e-tests

.PHONY: all
all: check format lint security build test

.PHONY: ci
ci: ensure-generate-is-noop check format lint security build unit-tests

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
	$(KUSTOMIZE) build config/default | kubectl delete -f -

.PHONY: changelog
changelog:
	$(ECHO) "Set env variable OAUTH_TOKEN before invoking, https://github.com/settings/tokens/new?description=GitHub%20Changelog%20Generator%20token"
	$(VECHO)docker run --rm  -v "${PWD}:/app" pavolloffay/gch:latest --oauth-token ${OAUTH_TOKEN} --owner jaegertracing --repo jaeger-operator


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

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

# end-to-tests
.PHONY: prepare-e2e-tests
prepare-e2e-tests: kuttl set-test-image-vars set-image-controller prepare-e2e-images generate-e2e-files build render-e2e-templates

.PHONY: generate-e2e-files
generate-e2e-files:
	mkdir -p tests/_build/crds tests/_build/manifests
	$(KUSTOMIZE) build config/default -o tests/_build/manifests/01-jaeger-operator.yaml
	$(KUSTOMIZE) build config/crd -o tests/_build/crds/

.PHONY: prepare-e2e-images
prepare-e2e-images: docker build-assert-job
	$(VECHO)docker pull jaegertracing/vertx-create-span:operator-e2e-tests
	$(VECHO)docker pull docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.6
	# Image for the upgrade E2E test
	$(VECHO)docker build --build-arg=GOPROXY=${GOPROXY}  --build-arg VERSION_PKG=${VERSION_PKG} --build-arg=JAEGER_VERSION=$(shell .ci/get_test_upgrade_version.sh ${JAEGER_VERSION}) --file Dockerfile -t "local/jaeger-operator:next" .

.PHONY: render-e2e-templates
render-e2e-templates:
# This files are needed for the examples
# examples-simplest
	$(VECHO)gomplate -f examples/simplest.yaml -o tests/e2e/examples-simplest/00-install.yaml
	$(VECHO)JAEGER_NAME=simplest gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-simplest/00-assert.yaml
	$(VECHO)JAEGER_SERVICE=smoketest JAEGER_OPERATION=smoketestoperation JAEGER_NAME=simplest gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-simplest/01-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-simplest/01-assert.yaml
# examples-with-badger
	$(VECHO)gomplate -f examples/with-badger.yaml -o tests/e2e/examples-with-badger/00-install.yaml
	$(VECHO)JAEGER_NAME=with-badger gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-with-badger/00-assert.yaml
	$(VECHO)JAEGER_SERVICE=with-badger JAEGER_OPERATION=smoketestoperation JAEGER_NAME=with-badger gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-with-badger/01-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-with-badger/01-assert.yaml
# examples-with-badger-and-volume
	$(VECHO)gomplate -f examples/with-badger-and-volume.yaml -o tests/e2e/examples-with-badger-and-volume/00-install.yaml
	$(VECHO)JAEGER_NAME=with-badger-and-volume gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-with-badger-and-volume/00-assert.yaml
	$(VECHO)JAEGER_SERVICE=with-badger-and-volume JAEGER_OPERATION=smoketestoperation JAEGER_NAME=with-badger-and-volume gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-with-badger-and-volume/01-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-with-badger-and-volume/01-assert.yaml
# examples-service-types
	$(VECHO)gomplate -f examples/service-types.yaml -o tests/e2e/examples-service-types/00-install.yaml
	$(VECHO)JAEGER_NAME=service-types gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-service-types/00-assert.yaml
	$(VECHO)JAEGER_SERVICE=service-types JAEGER_OPERATION=smoketestoperation JAEGER_NAME=service-types gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-service-types/01-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-service-types/01-assert.yaml
# examples-simple-prod
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/examples-simple-prod/00-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/examples-simple-prod/00-assert.yaml
	$(VECHO)gomplate -f examples/simple-prod.yaml -o tests/e2e/examples-simple-prod/01-install.yaml
	$(VECHO)${SED} -i "s~server-urls: http://elasticsearch.default.svc:9200~server-urls: http://elasticsearch:9200~gi" tests/e2e/examples-simple-prod/01-install.yaml
	$(VECHO)JAEGER_NAME=simple-prod gomplate -f tests/templates/production-jaeger-assert.yaml.template -o tests/e2e/examples-simple-prod/01-assert.yaml
	$(VECHO)JAEGER_SERVICE=simple-prod JAEGER_OPERATION=smoketestoperation JAEGER_NAME=simple-prod gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-simple-prod/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-simple-prod/02-assert.yaml
# examples-simple-prod-with-volumes
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/examples-simple-prod-with-volumes/00-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/examples-simple-prod-with-volumes/00-assert.yaml
	$(VECHO)gomplate -f examples/simple-prod-with-volumes.yaml -o tests/e2e/examples-simple-prod-with-volumes/01-install.yaml
	$(VECHO)${SED} -i "s~server-urls: http://elasticsearch.default.svc:9200~server-urls: http://elasticsearch:9200~gi" tests/e2e/examples-simple-prod-with-volumes/01-install.yaml
	$(VECHO)JAEGER_NAME=simple-prod gomplate -f tests/templates/production-jaeger-assert.yaml.template -o tests/e2e/examples-simple-prod-with-volumes/01-assert.yaml
	$(VECHO)JAEGER_SERVICE=simple-prod-with-volumes JAEGER_OPERATION=smoketestoperation JAEGER_NAME=simple-prod gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-simple-prod-with-volumes/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-simple-prod-with-volumes/02-assert.yaml
# examples-with-sampling
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/examples-with-sampling/00-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/examples-with-sampling/00-assert.yaml
	$(VECHO)gomplate -f examples/with-sampling.yaml -o tests/e2e/examples-with-sampling/01-install.yaml
	$(VECHO)${SED} -i "s~server-urls: http://elasticsearch.default.svc:9200~server-urls: http://elasticsearch:9200~gi" tests/e2e/examples-with-sampling/01-install.yaml
	$(VECHO)JAEGER_NAME=with-sampling gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-with-sampling/01-assert.yaml
	$(VECHO)JAEGER_SERVICE=with-sampling JAEGER_OPERATION=smoketestoperation JAEGER_NAME=with-sampling gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-with-sampling/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-with-sampling/02-assert.yaml
# This is needed for the generate test
	$(VECHO)@JAEGER_VERSION=${JAEGER_VERSION} gomplate -f tests/e2e/generate/jaeger-template.yaml.template -o tests/e2e/generate/jaeger-deployment.yaml
# This is needed for the upgrade test
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

# end-to-tests
.PHONY: e2e-tests
e2e-tests: prepare-e2e-tests start-kind run-e2e-tests

.PHONY: run-e2e-tests
run-e2e-tests:
	$(VECHO)$(KUTTL) test

.PHONY: start-kind
start-kind: 
	$(VECHO)kind create cluster --config $(KIND_CONFIG)
	$(VECHO)kind load docker-image local/jaeger-operator:e2e
	$(VECHO)kind load docker-image local/asserts:e2e
	$(VECHO)kind load docker-image jaegertracing/vertx-create-span:operator-e2e-tests
	$(VECHO)kind load docker-image local/jaeger-operator:next
	$(VECHO)kind load docker-image docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.6

.PHONY: build-assert-job
build-assert-job:
	$(VECHO)docker build -t local/asserts:e2e  -f Dockerfile.asserts .


.PHONY: build-assert-job
install-git-hooks:
	$(VECHO)cp scripts/git-hooks/pre-commit .git/hooks

set-test-image-vars:
	$(eval IMG=local/jaeger-operator:e2e)

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