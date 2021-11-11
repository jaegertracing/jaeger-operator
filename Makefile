# When the VERBOSE variable is set to 1, all the commands are shown
ifeq ("$(VERBOSE)","1")
echo_prefix=">>>>"
else
VECHO = @
endif

VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x
GOARCH ?= $(go env GOARCH)
GOOS ?= $(go env GOOS)
GO_FLAGS ?= GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 GO111MODULE=on
KUBERNETES_CONFIG ?= "$(HOME)/.kube/config"
WATCH_NAMESPACE ?= ""
BIN_DIR ?= "build/_output/bin"
IMPORT_LOG=import.log
FMT_LOG=fmt.log

OPERATOR_NAME ?= jaeger-operator
NAMESPACE ?= "$(USER)"
BUILD_IMAGE ?= "$(NAMESPACE)/$(OPERATOR_NAME):latest"
IMAGE_TAGS ?= "--tag $(BUILD_IMAGE)"
OUTPUT_BINARY ?= "$(BIN_DIR)/$(OPERATOR_NAME)"
VERSION_PKG ?= "github.com/jaegertracing/jaeger-operator/pkg/version"
JAEGER_VERSION ?= "$(shell grep jaeger= versions.txt | awk -F= '{print $$2}')"
OPERATOR_VERSION ?= "$(shell git describe --tags)"
STORAGE_NAMESPACE ?= "${shell kubectl get sa default -o jsonpath='{.metadata.namespace}' || oc project -q}"
KAFKA_NAMESPACE ?= "kafka"
KAFKA_EXAMPLE ?= "https://raw.githubusercontent.com/strimzi/strimzi-kafka-operator/0.23.0/examples/kafka/kafka-persistent-single.yaml"
KAFKA_YAML ?= "https://github.com/strimzi/strimzi-kafka-operator/releases/download/0.23.0/strimzi-cluster-operator-0.23.0.yaml"
ES_OPERATOR_NAMESPACE ?= openshift-logging
ES_OPERATOR_BRANCH ?= release-4.4
ES_OPERATOR_IMAGE ?= quay.io/openshift/origin-elasticsearch-operator:4.4
SDK_VERSION=v0.18.2
ISTIO_VERSION ?= 1.11.2
ISTIOCTL="./deploy/test/istio/bin/istioctl"
GOPATH ?= "$(HOME)/go"
GOROOT ?= "$(shell go env GOROOT)"

ECHO ?= @echo $(echo_prefix)
SED ?= "sed"

PROMETHEUS_OPERATOR_TAG ?= v0.39.0
PROMETHEUS_BUNDLE ?= https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/${PROMETHEUS_OPERATOR_TAG}/bundle.yaml

LD_FLAGS ?= "-X $(VERSION_PKG).version=$(OPERATOR_VERSION) -X $(VERSION_PKG).buildDate=$(VERSION_DATE) -X $(VERSION_PKG).defaultJaeger=$(JAEGER_VERSION)"

UNIT_TEST_PACKAGES := $(shell go list ./cmd/... ./pkg/... | grep -v elasticsearch/v1 | grep -v kafka/v1beta2 | grep -v client/versioned)

TEST_OPTIONS = $(VERBOSE) -kubeconfig $(KUBERNETES_CONFIG) -namespacedMan ../../deploy/test/namespace-manifests.yaml -globalMan ../../deploy/test/global-manifests.yaml -root .

KUBE_VERSION ?= 1.21
KIND_CONFIG ?= kind-$(KUBE_VERSION).yaml

.DEFAULT_GOAL := build

.PHONY: check
check:
	$(ECHO) Checking...
	$(VECHO)GOPATH=${GOPATH} .ci/format.sh > $(FMT_LOG)
	$(VECHO)[ ! -s "$(FMT_LOG)" ] || (echo "Go fmt, license check, or import ordering failures, run 'make format'" | cat - $(FMT_LOG) && false)

.PHONY: ensure-generate-is-noop
ensure-generate-is-noop: generate format
	$(VECHO)git diff  pkg/apis/jaegertracing/v1/zz_generated.*.go
	$(VECHO)git diff -s --exit-code pkg/apis/jaegertracing/v1/zz_generated.*.go || (echo "Build failed: a model has been changed but the generated resources aren't up to date. Run 'make generate' and update your PR." && exit 1)
	$(VECHO)git diff -s --exit-code pkg/client/versioned || (echo "Build failed: the versioned clients aren't up to date. Run 'make generate'." && exit 1)


.PHONY: format
format:
	$(ECHO) Formatting code...
	$(VECHO)GOPATH=${GOPATH} .ci/format.sh

.PHONY: lint
lint:
	$(ECHO) Linting...
	$(VECHO)GOPATH=${GOPATH} ./.ci/lint.sh

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
	$(VECHO)${GO_FLAGS} go build -o $(OUTPUT_BINARY) -ldflags $(LD_FLAGS)

.PHONY: docker
docker:
	$(VECHO)[ ! -z "$(PIPELINE)" ] || docker build --build-arg=GOPROXY=${GOPROXY} --build-arg=JAEGER_VERSION=${JAEGER_VERSION} --build-arg=TARGETARCH=$(GOARCH) --file build/Dockerfile -t "$(BUILD_IMAGE)" .

.PHONY: dockerx
dockerx:
	$(VECHO)[ ! -z "$(PIPELINE)" ] || docker buildx build --push --progress=plain --build-arg=JAEGER_VERSION=${JAEGER_VERSION} --build-arg=GOPROXY=${GOPROXY} --platform=$(PLATFORMS) --file build/Dockerfile $(IMAGE_TAGS) .

.PHONY: push
push:
ifeq ($(CI),true)
	$(ECHO) Skipping push, as the build is running within a CI environment
else
	$(ECHO) "Pushing image $(BUILD_IMAGE)..."
	$(VECHO)docker push $(BUILD_IMAGE) > /dev/null
endif

.PHONY: unit-tests
unit-tests:
	$(ECHO) Running unit tests...
	$(VECHO)go test $(VERBOSE) $(UNIT_TEST_PACKAGES) -cover -coverprofile=cover.out -ldflags $(LD_FLAGS)

.PHONY: run
run: crd
	$(VECHO)rm -rf /tmp/_cert*
	$(VECHO)POD_NAMESPACE=default OPERATOR_NAME=${OPERATOR_NAME} operator-sdk run local --watch-namespace="${WATCH_NAMESPACE}" --operator-flags "start ${CLI_FLAGS}" --go-ldflags ${LD_FLAGS}

.PHONY: run-debug
run-debug: run
run-debug: CLI_FLAGS = --log-level=debug --tracing-enabled=true

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

.PHONY: crd
crd:
	$(VECHO)kubectl create -f deploy/crds/jaegertracing.io_jaegers_crd.yaml 2>&1 | grep -v "already exists" || true

.PHONY: ingress
ingress:
	$(VECHO)minikube addons enable ingress

.PHONY: generate
generate: internal-generate format

.PHONY: internal-generate
internal-generate:
	$(VECHO)GOPATH=${GOPATH} GOROOT=${GOROOT} ./.ci/generate.sh

.PHONY: test
test: unit-tests e2e-tests

.PHONY: all
all: check format lint security build test

.PHONY: ci
ci: ensure-generate-is-noop check format lint security build unit-tests

.PHONY: scorecard
scorecard:
	$(VECHO)operator-sdk scorecard --cr-manifest deploy/examples/simplest.yaml --csv-path deploy/olm-catalog/jaeger.clusterserviceversion.yaml --init-timeout 30

.PHONY: install-sdk
install-sdk:
	$(ECHO) Installing SDK ${SDK_VERSION}
	$(VECHO)SDK_VERSION=$(SDK_VERSION) GOPATH=$(GOPATH) ./.ci/install-sdk.sh

.PHONY: install-tools
install-tools:
	$(VECHO)${GO_FLAGS} ./.ci/vgot.sh \
		golang.org/x/lint/golint \
		golang.org/x/tools/cmd/goimports \
		github.com/securego/gosec/cmd/gosec@v0.0.0-20191008095658-28c1128b7336 \
		sigs.k8s.io/controller-tools/cmd/controller-gen@v0.5.0 \
		k8s.io/code-generator/cmd/client-gen@v0.18.6 \
		k8s.io/kube-openapi/cmd/openapi-gen@v0.0.0-20200410145947-61e04a5be9a6
	./.ci/install-gomplate.sh

.PHONY: install
install: install-sdk install-tools

.PHONY: deploy
deploy: ingress crd
	$(VECHO)kubectl apply -f deploy/service_account.yaml
	$(VECHO)kubectl apply -f deploy/cluster_role.yaml
	$(VECHO)kubectl apply -f deploy/cluster_role_binding.yaml
	$(VECHO)${SED} "s~image: jaegertracing\/jaeger-operator\:.*~image: $(BUILD_IMAGE)~gi" deploy/operator.yaml | kubectl apply -f -

.PHONY: operatorhub
operatorhub: check-operatorhub-pr-template
	$(VECHO)./.ci/operatorhub.sh

.PHONY: check-operatorhub-pr-template
check-operatorhub-pr-template:
	$(VECHO)curl https://raw.githubusercontent.com/operator-framework/community-operators/master/docs/pull_request_template.md -o .ci/.operatorhub-pr-template.md -s > /dev/null 2>&1
	$(VECHO)git diff -s --exit-code .ci/.operatorhub-pr-template.md || (echo "Build failed: the PR template for OperatorHub has changed. Sync it and try again." && exit 1)

.PHONY: local-jaeger-container
local-jaeger-container:
	$(ECHO) "Starting local container with Jaeger. Check http://localhost:16686"
	$(VECHO)docker run -d --rm -p 16686:16686 -p 6831:6831/udp --name jaeger jaegertracing/all-in-one:1.22 > /dev/null

.PHONY: changelog
changelog:
	$(ECHO) "Set env variable OAUTH_TOKEN before invoking, https://github.com/settings/tokens/new?description=GitHub%20Changelog%20Generator%20token"
	$(VECHO)docker run --rm  -v "${PWD}:/app" pavolloffay/gch:latest --oauth-token ${OAUTH_TOKEN} --owner jaegertracing --repo jaeger-operator


# e2e tests using kuttl

kuttl:
ifeq (, $(shell which kubectl-kuttl))
	echo ${PATH}
	ls -l /usr/local/bin
	which kubectl-kuttl

	$(VECHO){ \
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

kind:
ifeq (, $(shell which kind))
	$(VECHO){ \
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

.PHONY: prepare-e2e-tests
prepare-e2e-tests: BUILD_IMAGE="local/jaeger-operator:e2e"
prepare-e2e-tests: prepare-e2e-images generate-e2e-files
	$(VECHO)mkdir -p  tests/_build/manifests
	$(VECHO)mkdir -p  tests/_build/crds

.PHONY: prepare-e2e-images
prepare-e2e-images: docker build-assert-job
	$(ECHO) Building the container images needed to run the E2E tests
	$(VECHO)docker pull jaegertracing/vertx-create-span:operator-e2e-tests
	$(VECHO)docker pull docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.6
	@# Image for the upgrade E2E test
	$(VECHO)docker build --build-arg=GOPROXY=${GOPROXY}  --build-arg=JAEGER_VERSION=$(shell .ci/get_test_upgrade_version.sh ${JAEGER_VERSION}) --file build/Dockerfile -t "local/jaeger-operator:next" .

.PHONY: generate-e2e-files
generate-e2e-files: build
	$(ECHO) Generating the files needed to run the E2E tests
	@# Generate the Jaeger manifest
	$(VECHO)cp deploy/service_account.yaml tests/_build/manifests/01-jaeger-operator.yaml
	$(ECHO) "---" >> tests/_build/manifests/01-jaeger-operator.yaml

	$(VECHO)cat deploy/role.yaml >> tests/_build/manifests/01-jaeger-operator.yaml
	$(ECHO) "---" >> tests/_build/manifests/01-jaeger-operator.yaml

	$(VECHO)cat deploy/cluster_role.yaml >> tests/_build/manifests/01-jaeger-operator.yaml
	$(ECHO) "---" >> tests/_build/manifests/01-jaeger-operator.yaml

	$(VECHO)${SED} "s~namespace: .*~namespace: jaeger-operator-system~gi" deploy/cluster_role_binding.yaml >> tests/_build/manifests/01-jaeger-operator.yaml
	$(ECHO) "---" >> tests/_build/manifests/01-jaeger-operator.yaml

	$(VECHO)${SED} "s~image: jaegertracing\/jaeger-operator\:.*~image: $(BUILD_IMAGE)~gi" deploy/operator.yaml >> tests/_build/manifests/01-jaeger-operator.yaml
	$(VECHO)${SED} "s~imagePullPolicy: Always~imagePullPolicy: Never~gi" tests/_build/manifests/01-jaeger-operator.yaml -i
	$(VECHO)${SED} "0,/fieldPath: metadata.namespace/s/fieldPath: metadata.namespace/fieldPath: metadata.annotations['olm.targetNamespaces']/gi" tests/_build/manifests/01-jaeger-operator.yaml -i

	$(VECHO)cp deploy/crds/jaegertracing.io_jaegers_crd.yaml tests/_build/crds/jaegertracing.io_jaegers_crd.yaml

	@# Generate all the files for the steps performed by the E2E tests
	@# generate
	$(VECHO)@JAEGER_VERSION=${JAEGER_VERSION} gomplate -f tests/e2e/generate/jaeger-template.yaml.template -o tests/e2e/generate/jaeger-deployment.yaml
	@# upgrade
	$(VECHO)JAEGER_VERSION=${JAEGER_VERSION} gomplate -f tests/e2e/upgrade/deployment-assert.yaml.template -o tests/e2e/upgrade/00-assert.yaml
	$(VECHO)JAEGER_VERSION=$(shell .ci/get_test_upgrade_version.sh ${JAEGER_VERSION}) gomplate -f tests/e2e/upgrade/deployment-assert.yaml.template -o tests/e2e/upgrade/01-assert.yaml
	$(VECHO)JAEGER_VERSION=${JAEGER_VERSION} gomplate -f tests/e2e/upgrade/deployment-assert.yaml.template -o tests/e2e/upgrade/02-assert.yaml
	$(VECHO)${SED} "s~local/jaeger-operator:e2e~local/jaeger-operator:next~gi" tests/_build/manifests/01-jaeger-operator.yaml > tests/e2e/upgrade/operator-upgrade.yaml
	@# This is needed for the streaming tests
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/streaming-simple/01-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/streaming-simple/01-assert.yaml
	$(VECHO)REPLICAS=1 CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-kafka-cluster.yaml.template -o tests/e2e/streaming-simple/02-assert.yaml
	$(VECHO)REPLICAS=1 CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-zookeeper-cluster.yaml.template -o tests/e2e/streaming-simple/03-assert.yaml
	$(VECHO)CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-entity-operator.yaml.template -o tests/e2e/streaming-simple/04-assert.yaml
	$(VECHO)JAEGER_SERVICE=simple-streaming JAEGER_OPERATION=smoketestoperation JAEGER_NAME=simple-streaming gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/streaming-simple/06-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/streaming-simple/06-assert.yaml
	@# streaming-with-tls
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/streaming-with-tls/01-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/streaming-with-tls/01-assert.yaml
	$(VECHO)REPLICAS=1 CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-kafka-cluster.yaml.template -o tests/e2e/streaming-with-tls/02-assert.yaml
	$(VECHO)REPLICAS=1 CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-zookeeper-cluster.yaml.template -o tests/e2e/streaming-with-tls/03-assert.yaml
	$(VECHO)CLUSTER_NAME=my-cluster gomplate -f tests/templates/assert-entity-operator.yaml.template -o tests/e2e/streaming-with-tls/04-assert.yaml
	$(VECHO)JAEGER_SERVICE=streaming-with-tls JAEGER_OPERATION=smoketestoperation JAEGER_NAME=tls-streaming gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/streaming-with-tls/07-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/streaming-with-tls/07-assert.yaml
	@# streaming-with-autoprovisioning
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/streaming-with-autoprovisioning/01-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/streaming-with-autoprovisioning/01-assert.yaml
	$(VECHO)REPLICAS=3 CLUSTER_NAME=auto-provisioned gomplate -f tests/templates/assert-zookeeper-cluster.yaml.template -o tests/e2e/streaming-with-autoprovisioning/02-assert.yaml
	$(VECHO)REPLICAS=3 CLUSTER_NAME=auto-provisioned gomplate -f tests/templates/assert-kafka-cluster.yaml.template -o tests/e2e/streaming-with-autoprovisioning/03-assert.yaml
	$(VECHO)CLUSTER_NAME=auto-provisioned gomplate -f tests/templates/assert-entity-operator.yaml.template -o tests/e2e/streaming-with-autoprovisioning/04-assert.yaml
	$(VECHO)JAEGER_SERVICE=streaming-with-autoprovisioning JAEGER_OPERATION=smoketestoperation JAEGER_NAME=auto-provisioned gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/streaming-with-autoprovisioning/06-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/streaming-with-autoprovisioning/06-assert.yaml
	@# examples-agent-as-daemonset
	$(VECHO)gomplate -f examples/agent-as-daemonset.yaml -o tests/e2e/examples-agent-as-daemonset/00-install.yaml
	$(VECHO)JAEGER_NAME=agent-as-daemonset gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-agent-as-daemonset/00-assert.yaml
	$(VECHO)JAEGER_SERVICE=agent-as-daemonset JAEGER_OPERATION=smoketestoperation JAEGER_NAME=agent-as-daemonset gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-agent-as-daemonset/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-agent-as-daemonset/02-assert.yaml
	@# examples-with-cassandra
	$(VECHO)gomplate -f tests/templates/cassandra-install.yaml.template -o tests/e2e/examples-with-cassandra/00-install.yaml
	$(VECHO)gomplate -f tests/templates/cassandra-assert.yaml.template -o tests/e2e/examples-with-cassandra/00-assert.yaml
	$(VECHO)gomplate -f examples/with-cassandra.yaml -o tests/e2e/examples-with-cassandra/01-install.yaml
	$(VECHO)${SED} -i "s~cassandra.default.svc~cassandra~gi" tests/e2e/examples-with-cassandra/01-install.yaml
	$(VECHO)JAEGER_NAME=with-cassandra gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-with-cassandra/01-assert.yaml
	$(VECHO)JAEGER_SERVICE=with-cassandra JAEGER_OPERATION=smoketestoperation JAEGER_NAME=with-cassandra gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-with-cassandra/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-with-cassandra/02-assert.yaml
	@# examples-business-application-injected-sidecar
	$(VECHO)cat examples/business-application-injected-sidecar.yaml tests/e2e/examples-business-application-injected-sidecar/livenessProbe.yaml >  tests/e2e/examples-business-application-injected-sidecar/00-install.yaml
	$(VECHO)gomplate -f  examples/simplest.yaml -o tests/e2e/examples-business-application-injected-sidecar/01-install.yaml
	$(VECHO)JAEGER_NAME=simplest gomplate -f tests/templates/allinone-jaeger-assert.yaml.template -o tests/e2e/examples-business-application-injected-sidecar/01-assert.yaml
	$(VECHO)JAEGER_SERVICE=simplest JAEGER_OPERATION=smoketestoperation JAEGER_NAME=simplest gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/examples-business-application-injected-sidecar/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/examples-business-application-injected-sidecar/02-assert.yaml
	@# istio
	$(VECHO)cat examples/business-application-injected-sidecar.yaml tests/e2e/istio/livelinessprobe.template > tests/e2e/istio/03-install.yaml
	@# cassandra
	$(VECHO)gomplate -f tests/templates/cassandra-install.yaml.template -o tests/e2e/cassandra/00-install.yaml
	$(VECHO)gomplate -f tests/templates/cassandra-assert.yaml.template -o tests/e2e/cassandra/00-assert.yaml
	$(VECHO)INSTANCE_NAME=with-cassandra  gomplate -f tests/templates/cassandra-jaeger-install.yaml.template -o tests/e2e/cassandra/01-install.yaml
	$(VECHO)INSTANCE_NAME=with-cassandra  gomplate -f tests/templates/cassandra-jaeger-assert.yaml.template -o tests/e2e/cassandra/01-assert.yaml
	@# cassandra-spark
	$(VECHO) gomplate -f tests/templates/cassandra-install.yaml.template -o tests/e2e/cassandra-spark/00-install.yaml
	$(VECHO) gomplate -f tests/templates/cassandra-assert.yaml.template -o tests/e2e/cassandra-spark/00-assert.yaml
	$(VECHO)INSTANCE_NAME=test-spark-deps DEP_SCHEDULE=true CASSANDRA_MODE=prod gomplate -f tests/templates/cassandra-jaeger-install.yaml.template -o tests/e2e/cassandra-spark/01-install.yaml
	@# es-spark-dependencies
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/es-spark-dependencies/00-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/es-spark-dependencies/00-assert.yaml
	@# es-simple-prod
	$(VECHO)gomplate -f tests/templates/elasticsearch-install.yaml.template -o tests/e2e/es-simple-prod/00-install.yaml
	$(VECHO)gomplate -f tests/templates/elasticsearch-assert.yaml.template -o tests/e2e/es-simple-prod/00-assert.yaml
	$(VECHO)JAEGER_NAME=simple-prod gomplate -f tests/templates/production-jaeger-install.yaml.template -o tests/e2e/es-simple-prod/01-install.yaml
	$(VECHO)JAEGER_NAME=simple-prod gomplate -f tests/templates/production-jaeger-assert.yaml.template -o tests/e2e/es-simple-prod/01-assert.yaml
	$(VECHO)JAEGER_SERVICE=simple-prod JAEGER_OPERATION=smoketestoperation JAEGER_NAME=simple-prod gomplate -f tests/templates/smoke-test.yaml.template -o tests/e2e/es-simple-prod/02-smoke-test.yaml
	$(VECHO)gomplate -f tests/templates/smoke-test-assert.yaml.template -o tests/e2e/es-simple-prod/02-assert.yaml
	@# es-index-cleaner
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

start-kind: prepare-e2e-tests
# Instead of letting KUTTL create the Kind cluster (using the CLI or in the kuttl-tests.yaml
# file), the cluster is created here. There are multiple reasons to do this:
# 	* The kubectl command will not work outside KUTTL
#	* Some KUTTL versions are not able to start properly a Kind cluster
#	* The cluster will be removed after running KUTTL (this can be disabled). Sometimes,
#		the cluster teardown is not done properly and KUTTL can not be run with the --start-kind flag
# When the Kind cluster is not created by Kuttl, the
# kindContainers parameter from kuttl-tests.yaml has not effect so, it is needed to load the
# container images here.
	$(VECHO)kind create cluster --config $(KIND_CONFIG) 2>&1 | grep -v "already exists" || true
	$(VECHO)kind load docker-image local/jaeger-operator:e2e
	$(VECHO)kind load docker-image local/asserts:e2e
	$(VECHO)kind load docker-image jaegertracing/vertx-create-span:operator-e2e-tests
	$(VECHO)kind load docker-image local/jaeger-operator:next
	$(VECHO)kind load docker-image docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.6

.PHONY: build-assert-job
build-assert-job:
	$(VECHO)docker build -t local/asserts:e2e  -f Dockerfile.asserts .
	$(VECHO)docker build -t local/asserts:e2e  -f Dockerfile.asserts .

.PHONY: build-assert-job
install-git-hooks:
	$(VECHO)cp scripts/git-hooks/pre-commit .git/hooks

.PHONY: prepare-release
prepare-release:
	$(VECHO)./.ci/prepare-release.sh
