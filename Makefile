VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GO_FLAGS ?= GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on
KUBERNETES_CONFIG ?= "$(HOME)/.kube/config"
WATCH_NAMESPACE ?= ""
BIN_DIR ?= "build/_output/bin"
IMPORT_LOG=import.log
FMT_LOG=fmt.log

OPERATOR_NAME ?= jaeger-operator
NAMESPACE ?= "$(USER)"
BUILD_IMAGE ?= "$(NAMESPACE)/$(OPERATOR_NAME):latest"
OUTPUT_BINARY ?= "$(BIN_DIR)/$(OPERATOR_NAME)"
VERSION_PKG ?= "github.com/jaegertracing/jaeger-operator/pkg/version"
JAEGER_VERSION ?= "$(shell grep jaeger= versions.txt | awk -F= '{print $$2}')"
OPERATOR_VERSION ?= "$(shell git describe --tags)"
STORAGE_NAMESPACE ?= "${shell kubectl get sa default -o jsonpath='{.metadata.namespace}' || oc project -q}"
KAFKA_NAMESPACE ?= "kafka"
KAFKA_EXAMPLE ?= "https://raw.githubusercontent.com/strimzi/strimzi-kafka-operator/0.16.2/examples/kafka/kafka-persistent-single.yaml"
KAFKA_YAML ?= "https://github.com/strimzi/strimzi-kafka-operator/releases/download/0.16.2/strimzi-cluster-operator-0.16.2.yaml"
ES_OPERATOR_NAMESPACE ?= openshift-logging
ES_OPERATOR_BRANCH ?= release-4.4
ES_OPERATOR_IMAGE ?= quay.io/openshift/origin-elasticsearch-operator:4.4
SDK_VERSION=v0.18.2
GOPATH ?= "$(HOME)/go"

PROMETHEUS_OPERATOR_TAG ?= v0.39.0
PROMETHEUS_BUNDLE ?= https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/${PROMETHEUS_OPERATOR_TAG}/bundle.yaml

LD_FLAGS ?= "-X $(VERSION_PKG).version=$(OPERATOR_VERSION) -X $(VERSION_PKG).buildDate=$(VERSION_DATE) -X $(VERSION_PKG).defaultJaeger=$(JAEGER_VERSION)"

UNIT_TEST_PACKAGES := $(shell go list ./cmd/... ./pkg/... | grep -v elasticsearch/v1 | grep -v kafka/v1beta1 | grep -v client/versioned)

TEST_OPTIONS = $(VERBOSE) -kubeconfig $(KUBERNETES_CONFIG) -namespacedMan ../../deploy/test/namespace-manifests.yaml -globalMan ../../deploy/test/global-manifests.yaml -root .

.DEFAULT_GOAL := build

.PHONY: check
check:
	@echo Checking...
	@GOPATH=${GOPATH} .ci/format.sh > $(FMT_LOG)
	@[ ! -s "$(FMT_LOG)" ] || (echo "Go fmt, license check, or import ordering failures, run 'make format'" | cat - $(FMT_LOG) && false)

.PHONY: ensure-generate-is-noop
ensure-generate-is-noop: generate format
	@git diff -s --exit-code pkg/apis/jaegertracing/v1/zz_generated.*.go || (echo "Build failed: a model has been changed but the generated resources aren't up to date. Run 'make generate' and update your PR." && exit 1)
	@git diff -s --exit-code pkg/client/versioned || (echo "Build failed: the versioned clients aren't up to date. Run 'make generate'." && exit 1)


.PHONY: format
format:
	@echo Formatting code...
	@GOPATH=${GOPATH} .ci/format.sh

.PHONY: lint
lint:
	@echo Linting...
	@GOPATH=${GOPATH} ./.ci/lint.sh

.PHONY: security
security:
	@echo Security...
	@${GOPATH}/bin/gosec -quiet -exclude=G104 ./... 2>/dev/null

.PHONY: build
build: format
	@echo Building...
	@${GO_FLAGS} go build -o $(OUTPUT_BINARY) -ldflags $(LD_FLAGS)
# compile the tests without running them
	@${GO_FLAGS} go test -c ./test/e2e/...

.PHONY: docker
docker:
	@[ ! -z "$(PIPELINE)" ] || docker build --file build/Dockerfile -t "$(BUILD_IMAGE)" .

.PHONY: push
push:
ifeq ($(CI),true)
	@echo Skipping push, as the build is running within a CI environment
else
	@echo "Pushing image $(BUILD_IMAGE)..."
	@docker push $(BUILD_IMAGE) > /dev/null
endif

.PHONY: unit-tests
unit-tests:
	@echo Running unit tests...
	@go test $(VERBOSE) $(UNIT_TEST_PACKAGES) -cover -coverprofile=cover.out -ldflags $(LD_FLAGS)

.PHONY: e2e-tests
e2e-tests: prepare-e2e-tests e2e-tests-smoke e2e-tests-cassandra e2e-tests-es e2e-tests-self-provisioned-es e2e-tests-streaming e2e-tests-examples1 e2e-tests-examples2 e2e-tests-examples-openshift e2e-tests-generate

.PHONY: prepare-e2e-tests
prepare-e2e-tests: build docker push
	@mkdir -p deploy/test
	@cp deploy/service_account.yaml deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml

	@cat deploy/role.yaml >> deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml

	@# ClusterRoleBinding is created in test codebase because we don't know service account namespace
	@cat deploy/role_binding.yaml >> deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml

	@sed "s~image: jaegertracing\/jaeger-operator\:.*~image: $(BUILD_IMAGE)~gi" test/operator.yaml >> deploy/test/namespace-manifests.yaml

	@cp deploy/crds/jaegertracing.io_jaegers_crd.yaml deploy/test/global-manifests.yaml
	@echo "---" >> deploy/test/global-manifests.yaml
	@cat deploy/cluster_role.yaml >> deploy/test/global-manifests.yaml

.PHONY: e2e-tests-smoke
e2e-tests-smoke: prepare-e2e-tests
	@echo Running Smoke end-to-end tests...
	@BUILD_IMAGE=$(BUILD_IMAGE) go test -tags=smoke ./test/e2e/... $(TEST_OPTIONS)

.PHONY: e2e-tests-generate
e2e-tests-generate: prepare-e2e-tests
	@echo Running generate end-to-end tests...
	@BUILD_IMAGE=$(BUILD_IMAGE) go test -tags=generate ./test/e2e/... $(TEST_OPTIONS)

.PHONY: e2e-tests-cassandra
e2e-tests-cassandra: prepare-e2e-tests cassandra
	@echo Running Cassandra end-to-end tests...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) go test -tags=cassandra ./test/e2e/... $(TEST_OPTIONS)

.PHONY: e2e-tests-es
e2e-tests-es: prepare-e2e-tests es
	@echo Running Elasticsearch end-to-end tests...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) go test -tags=elasticsearch ./test/e2e/... $(TEST_OPTIONS)

.PHONY: e2e-tests-self-provisioned-es
e2e-tests-self-provisioned-es: prepare-e2e-tests deploy-es-operator
	@echo Running Self provisioned Elasticsearch end-to-end tests...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) ES_OPERATOR_NAMESPACE=$(ES_OPERATOR_NAMESPACE) ES_OPERATOR_IMAGE=$(ES_OPERATOR_IMAGE) go test -tags=self_provisioned_elasticsearch ./test/e2e/... $(TEST_OPTIONS)

.PHONY: e2e-tests-self-provisioned-es-kafka
e2e-tests-self-provisioned-es-kafka: prepare-e2e-tests deploy-kafka-operator deploy-es-operator
	@echo Running Self provisioned Elasticsearch and Kafka end-to-end tests...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) ES_OPERATOR_NAMESPACE=$(ES_OPERATOR_NAMESPACE) ES_OPERATOR_IMAGE=$(ES_OPERATOR_IMAGE) go test -tags=self_provisioned_elasticsearch_kafka ./test/e2e/... $(TEST_OPTIONS)

.PHONY: e2e-tests-token-propagation-es
e2e-tests-token-propagation-es: prepare-e2e-tests deploy-es-operator
	@echo Running Token Propagation Elasticsearch end-to-end tests...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) ES_OPERATOR_NAMESPACE=$(ES_OPERATOR_NAMESPACE) TEST_TIMEOUT=5 ES_OPERATOR_IMAGE=$(ES_OPERATOR_IMAGE) go test -tags=token_propagation_elasticsearch ./test/e2e/... $(TEST_OPTIONS)

.PHONY: e2e-tests-streaming
e2e-tests-streaming: prepare-e2e-tests es kafka
	@echo Running Streaming end-to-end tests...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) KAFKA_NAMESPACE=$(KAFKA_NAMESPACE) go test -tags=streaming ./test/e2e/... $(TEST_OPTIONS)

.PHONY: e2e-tests-examples1
e2e-tests-examples1: prepare-e2e-tests cassandra
	@echo Running Example end-to-end tests part 1...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) KAFKA_NAMESPACE=$(KAFKA_NAMESPACE) go test -tags=examples1 ./test/e2e/... $(TEST_OPTIONS)

.PHONY: e2e-tests-examples2
e2e-tests-examples2: prepare-e2e-tests es kafka
	@echo Running Example end-to-end tests part 2...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) KAFKA_NAMESPACE=$(KAFKA_NAMESPACE) go test -tags=examples2 ./test/e2e/... $(TEST_OPTIONS)

.PHONY: e2e-tests-examples-openshift
e2e-tests-examples-openshift: prepare-e2e-tests deploy-es-operator
	@echo Running OpenShift Example end-to-end tests...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) KAFKA_NAMESPACE=$(KAFKA_NAMESPACE) go test -tags=examples_openshift ./test/e2e/... $(TEST_OPTIONS)

.PHONY: run
run: crd
	@rm -rf /tmp/_cert*
	@POD_NAMESPACE=default OPERATOR_NAME=${OPERATOR_NAME} operator-sdk run local --watch-namespace="${WATCH_NAMESPACE}" --operator-flags "start ${CLI_FLAGS}" --go-ldflags ${LD_FLAGS}

.PHONY: run-debug
run-debug: run
run-debug: CLI_FLAGS = --log-level=debug --tracing-enabled=true

.PHONY: set-max-map-count
set-max-map-count:
	# This is not required in OCP 4.1. The node tuning operator configures the property automatically
	# when label tuned.openshift.io/elasticsearch=true label is present on the ES pod. The label
	# is configured by ES operator.
	@minishift ssh -- 'sudo sysctl -w vm.max_map_count=262144' > /dev/null 2>&1 || true

.PHONY: set-node-os-linux
set-node-os-linux:
	# Elasticsearch requires labeled nodes. These labels are by default present in OCP 4.2
	@kubectl label nodes --all kubernetes.io/os=linux --overwrite

.PHONY: deploy-es-operator
deploy-es-operator: set-node-os-linux set-max-map-count deploy-prometheus-operator
ifeq ($(OLM),true)
	@echo Skipping es-operator deployment, assuming it has been installed via OperatorHub
else
	@kubectl create namespace ${ES_OPERATOR_NAMESPACE} 2>&1 | grep -v "already exists" || true
	@kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/01-service-account.yaml -n ${ES_OPERATOR_NAMESPACE}
	@kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/02-role.yaml
	@kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/03-role-bindings.yaml
	@kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/04-crd.yaml -n ${ES_OPERATOR_NAMESPACE}
	@kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/05-deployment.yaml -n ${ES_OPERATOR_NAMESPACE}
	@kubectl set image deployment/elasticsearch-operator elasticsearch-operator=${ES_OPERATOR_IMAGE} -n ${ES_OPERATOR_NAMESPACE}
endif

.PHONY: undeploy-es-operator
undeploy-es-operator:
ifeq ($(OLM),true)
	@echo Skipping es-operator undeployment, as it should have been installed via OperatorHub
else
	@kubectl delete -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/05-deployment.yaml -n ${ES_OPERATOR_NAMESPACE} --ignore-not-found=true || true
	@kubectl delete -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/04-crd.yaml -n ${ES_OPERATOR_NAMESPACE} --ignore-not-found=true || true
	@kubectl delete -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/03-role-bindings.yaml --ignore-not-found=true || true
	@kubectl delete -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/02-role.yaml --ignore-not-found=true || true
	@kubectl delete -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/${ES_OPERATOR_BRANCH}/manifests/01-service-account.yaml -n ${ES_OPERATOR_NAMESPACE} --ignore-not-found=true || true
	@kubectl delete namespace ${ES_OPERATOR_NAMESPACE} --ignore-not-found=true 2>&1 || true
endif

.PHONY: es
es: storage
	@kubectl create -f ./test/elasticsearch.yml --namespace $(STORAGE_NAMESPACE) 2>&1 | grep -v "already exists" || true

.PHONY: cassandra
cassandra: storage
	@kubectl create -f ./test/cassandra.yml --namespace $(STORAGE_NAMESPACE) 2>&1 | grep -v "already exists" || true

.PHONY: storage
storage:
	@echo Creating namespace $(STORAGE_NAMESPACE)
	@kubectl create namespace $(STORAGE_NAMESPACE) 2>&1 | grep -v "already exists" || true

.PHONY: deploy-kafka-operator
deploy-kafka-operator:
	@echo Creating namespace $(KAFKA_NAMESPACE)
	@kubectl create namespace $(KAFKA_NAMESPACE) 2>&1 | grep -v "already exists" || true
ifeq ($(OLM),true)
	@echo Skipping kafka-operator deployment, assuming it has been installed via OperatorHub
else
	@kubectl create clusterrolebinding strimzi-cluster-operator-namespaced --clusterrole=strimzi-cluster-operator-namespaced --serviceaccount ${KAFKA_NAMESPACE}:strimzi-cluster-operator 2>&1 | grep -v "already exists" || true
	@kubectl create clusterrolebinding strimzi-cluster-operator-entity-operator-delegation --clusterrole=strimzi-entity-operator --serviceaccount ${KAFKA_NAMESPACE}:strimzi-cluster-operator 2>&1 | grep -v "already exists" || true
	@kubectl create clusterrolebinding strimzi-cluster-operator-topic-operator-delegation --clusterrole=strimzi-topic-operator --serviceaccount ${KAFKA_NAMESPACE}:strimzi-cluster-operator 2>&1 | grep -v "already exists" || true
	@curl --fail --location $(KAFKA_YAML) --output deploy/test/kafka-operator.yaml --create-dirs
	@sed 's/namespace: .*/namespace: $(KAFKA_NAMESPACE)/' deploy/test/kafka-operator.yaml | kubectl -n $(KAFKA_NAMESPACE) apply -f - 2>&1 | grep -v "already exists" || true
	@kubectl set env deployment strimzi-cluster-operator -n ${KAFKA_NAMESPACE} STRIMZI_NAMESPACE="*"
endif

.PHONY: undeploy-kafka-operator
undeploy-kafka-operator:
ifeq ($(OLM),true)
	@echo Skiping kafka-operator undeploy
else
	@kubectl delete --namespace $(KAFKA_NAMESPACE) -f deploy/test/kafka-operator.yaml --ignore-not-found=true 2>&1 || true
	@kubectl delete clusterrolebinding strimzi-cluster-operator-namespaced --ignore-not-found=true || true
	@kubectl delete clusterrolebinding strimzi-cluster-operator-entity-operator-delegation --ignore-not-found=true || true
	@kubectl delete clusterrolebinding strimzi-cluster-operator-topic-operator-delegation --ignore-not-found=true || true
endif
	@kubectl delete namespace $(KAFKA_NAMESPACE) --ignore-not-found=true 2>&1 || true

.PHONY: kafka
kafka: deploy-kafka-operator
	@echo Creating namespace $(KAFKA_NAMESPACE)
	@kubectl create namespace $(KAFKA_NAMESPACE) 2>&1 | grep -v "already exists" || true
	@curl --fail --location $(KAFKA_EXAMPLE) --output deploy/test/kafka-example.yaml --create-dirs
	@sed -i 's/size: 100Gi/size: 10Gi/g' deploy/test/kafka-example.yaml
	@kubectl -n $(KAFKA_NAMESPACE) apply --dry-run=true -f deploy/test/kafka-example.yaml
	@kubectl -n $(KAFKA_NAMESPACE) apply -f deploy/test/kafka-example.yaml 2>&1 | grep -v "already exists" || true

.PHONY: undeploy-kafka
undeploy-kafka: undeploy-kafka-operator
	@kubectl delete --namespace $(KAFKA_NAMESPACE) -f deploy/test/kafka-example.yaml 2>&1 || true


.PHONY: deploy-prometheus-operator
deploy-prometheus-operator:
ifeq ($(OLM),true)
	@echo Skipping prometheus-operator deployment, assuming it has been installed via OperatorHub
else
	@kubectl apply -f ${PROMETHEUS_BUNDLE}
endif

.PHONY: undeploy-prometheus-operator
undeploy-prometheus-operator:
ifeq ($(OLM),true)
	@echo Skipping prometheus-operator undeployment, as it should have been installed via OperatorHub
else
	@kubectl delete -f ${PROMETHEUS_BUNDLE} --ignore-not-found=true || true
endif

.PHONY: clean
clean: undeploy-kafka undeploy-es-operator undeploy-prometheus-operator
	@rm -f deploy/test/*.yaml 
	@if [ -d deploy/test ]; then rmdir deploy/test ; fi
	@kubectl delete -f ./test/cassandra.yml --ignore-not-found=true -n $(STORAGE_NAMESPACE) || true
	@kubectl delete -f ./test/elasticsearch.yml --ignore-not-found=true -n $(STORAGE_NAMESPACE) || true
	@kubectl delete -f deploy/crds/jaegertracing.io_jaegers_crd.yaml --ignore-not-found=true || true
	@kubectl delete -f deploy/operator.yaml --ignore-not-found=true || true
	@kubectl delete -f deploy/role_binding.yaml --ignore-not-found=true || true
	@kubectl delete -f deploy/role.yaml --ignore-not-found=true || true
	@kubectl delete -f deploy/service_account.yaml --ignore-not-found=true || true

.PHONY: crd
crd:
	@kubectl create -f deploy/crds/jaegertracing.io_jaegers_crd.yaml 2>&1 | grep -v "already exists" || true

.PHONY: ingress
ingress:
	@minikube addons enable ingress

.PHONY: generate 
generate: internal-generate format

.PHONY: internal-generate
internal-generate:
	@GOPATH=${GOPATH} ./.ci/generate.sh

.PHONY: test
test: unit-tests e2e-tests

.PHONY: all
all: check format lint security build test

.PHONY: ci
ci: ensure-generate-is-noop check format lint security build unit-tests

.PHONY: scorecard
scorecard:
	@operator-sdk scorecard --cr-manifest deploy/examples/simplest.yaml --csv-path deploy/olm-catalog/jaeger.clusterserviceversion.yaml --init-timeout 30

.PHONY: install-sdk
install-sdk:
	@echo Installing SDK ${SDK_VERSION}
	@SDK_VERSION=$(SDK_VERSION) GOPATH=$(GOPATH) ./.ci/install-sdk.sh

.PHONY: install-tools
install-tools:
	@${GO_FLAGS} go install \
		golang.org/x/lint/golint \
		github.com/securego/gosec/cmd/gosec \
		golang.org/x/tools/cmd/goimports \
		sigs.k8s.io/controller-tools/cmd/controller-gen \
		k8s.io/code-generator/cmd/client-gen \
		k8s.io/kube-openapi/cmd/openapi-gen

.PHONY: install
install: install-sdk install-tools

.PHONY: deploy
deploy: ingress crd
	@kubectl apply -f deploy/service_account.yaml
	@kubectl apply -f deploy/cluster_role.yaml
	@kubectl apply -f deploy/cluster_role_binding.yaml
	@sed "s~image: jaegertracing\/jaeger-operator\:.*~image: $(BUILD_IMAGE)~gi" deploy/operator.yaml | kubectl apply -f -

.PHONY: operatorhub
operatorhub: check-operatorhub-pr-template
	@./.ci/operatorhub.sh

.PHONY: check-operatorhub-pr-template
check-operatorhub-pr-template:
	@curl https://raw.githubusercontent.com/operator-framework/community-operators/master/docs/pull_request_template.md -o .ci/.operatorhub-pr-template.md -s > /dev/null 2>&1
	@git diff -s --exit-code .ci/.operatorhub-pr-template.md || (echo "Build failed: the PR template for OperatorHub has changed. Sync it and try again." && exit 1)

.PHONY: local-jaeger-container
local-jaeger-container:
	@echo "Starting local container with Jaeger. Check http://localhost:16686"
	@docker run -d --rm -p 16686:16686 -p 6831:6831/udp --name jaeger jaegertracing/all-in-one:1.15 > /dev/null

.PHONY: changelog
changelog:
	@echo "Set env variable OAUTH_TOKEN before invoking, https://github.com/settings/tokens/new?description=GitHub%20Changelog%20Generator%20token"
	@docker run --rm  -v "${PWD}:/app" pavolloffay/gch:latest --oauth-token ${OAUTH_TOKEN} --owner jaegertracing --repo jaeger-operator
