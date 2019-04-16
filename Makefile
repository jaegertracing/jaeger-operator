VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GO_FLAGS ?= GOOS=linux GOARCH=amd64 CGO_ENABLED=0
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
JAEGER_VERSION ?= "$(shell grep -v '\#' jaeger.version)"
OPERATOR_VERSION ?= "$(shell git describe --tags)"
STORAGE_NAMESPACE ?= "${shell kubectl get sa default -o jsonpath='{.metadata.namespace}' || oc project -q}"
ES_OPERATOR_NAMESPACE = openshift-logging

LD_FLAGS ?= "-X $(VERSION_PKG).version=$(OPERATOR_VERSION) -X $(VERSION_PKG).buildDate=$(VERSION_DATE) -X $(VERSION_PKG).defaultJaeger=$(JAEGER_VERSION)"
PACKAGES := $(shell go list ./cmd/... ./pkg/...)

.DEFAULT_GOAL := build

.PHONY: check
check:
	@echo Checking...
	@go fmt $(PACKAGES) > $(FMT_LOG)
	@.travis/import-order-cleanup.sh stdout > $(IMPORT_LOG)
	@[ ! -s "$(FMT_LOG)" -a ! -s "$(IMPORT_LOG)" ] || (echo "Go fmt, license check, or import ordering failures, run 'make format'" | cat - $(FMT_LOG) $(IMPORT_LOG) && false)

.PHONY: ensure-generate-is-noop
ensure-generate-is-noop: generate
	@git diff -s --exit-code pkg/apis/jaegertracing/v1/zz_generated.deepcopy.go || (echo "Build failed: a model has been changed but the deep copy functions aren't up to date. Run 'make generate' and update your PR." && exit 1)

.PHONY: format
format:
	@echo Formatting code...
	@.travis/import-order-cleanup.sh inplace
	@go fmt $(PACKAGES)

.PHONY: lint
lint:
	@echo Linting...
	@golint $(PACKAGES)
	@gosec -quiet -exclude=G104 $(PACKAGES) 2>/dev/null

.PHONY: build
build: format
	@echo Building...
	@${GO_FLAGS} go build -o $(OUTPUT_BINARY) -ldflags $(LD_FLAGS)

.PHONY: docker
docker:
	@docker build --file build/Dockerfile -t "$(BUILD_IMAGE)" .

.PHONY: push
push:
	@echo Pushing image $(BUILD_IMAGE)...
	@docker push $(BUILD_IMAGE) > /dev/null

.PHONY: unit-tests
unit-tests:
	@echo Running unit tests...
	@go test $(PACKAGES) -cover -coverprofile=cover.out

.PHONY: e2e-tests
e2e-tests: prepare-e2e-tests e2e-tests-smoke e2e-tests-cassandra e2e-tests-es e2e-tests-self-provisioned-es

.PHONY: prepare-e2e-tests
prepare-e2e-tests: crd build docker push
	@mkdir -p deploy/test
	@cp test/role_binding.yaml deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml

	@cat test/role.yaml >> deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml

	@cat test/service_account.yaml >> deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml

	@cat test/operator.yaml | sed "s~image: jaegertracing\/jaeger-operator\:.*~image: $(BUILD_IMAGE)~gi" >> deploy/test/namespace-manifests.yaml

.PHONY: e2e-tests-smoke
e2e-tests-smoke: prepare-e2e-tests
	@echo Running Smoke end-to-end tests...
	@go test -tags=smoke ./test/e2e/... -kubeconfig $(KUBERNETES_CONFIG) -namespacedMan ../../deploy/test/namespace-manifests.yaml -globalMan ../../deploy/crds/jaegertracing_v1_jaeger_crd.yaml -root .

.PHONY: e2e-tests-cassandra
e2e-tests-cassandra: prepare-e2e-tests cassandra
	@echo Running Cassandra end-to-end tests...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) go test -tags=cassandra ./test/e2e/... -kubeconfig $(KUBERNETES_CONFIG) -namespacedMan ../../deploy/test/namespace-manifests.yaml -globalMan ../../deploy/crds/jaegertracing_v1_jaeger_crd.yaml -root .

.PHONY: e2e-tests-es
e2e-tests-es: prepare-e2e-tests es
	@echo Running Elasticsearch end-to-end tests...
	@STORAGE_NAMESPACE=$(STORAGE_NAMESPACE) go test -tags=elasticsearch ./test/e2e/... -kubeconfig $(KUBERNETES_CONFIG) -namespacedMan ../../deploy/test/namespace-manifests.yaml -globalMan ../../deploy/crds/jaegertracing_v1_jaeger_crd.yaml -root .

.PHONY: e2e-tests-self-provisioned-es
e2e-tests-self-provisioned-es: prepare-e2e-tests deploy-es-operator
	@echo Running Self provisioned Elasticsearch end-to-end tests...
	@go test -tags=self_provisioned_elasticsearch ./test/e2e/... -kubeconfig $(KUBERNETES_CONFIG) -namespacedMan ../../deploy/test/namespace-manifests.yaml -globalMan ../../deploy/crds/jaegertracing_v1_jaeger_crd.yaml -root .

.PHONY: run
run: crd
	@rm -rf /tmp/_cert*
	@bash -c 'trap "exit 0" INT; OPERATOR_NAME=${OPERATOR_NAME} KUBERNETES_CONFIG=${KUBERNETES_CONFIG} WATCH_NAMESPACE=${WATCH_NAMESPACE} go run -ldflags ${LD_FLAGS} main.go start'

.PHONY: set-max-map-count
set-max-map-count:
	@minishift ssh -- 'sudo sysctl -w vm.max_map_count=262144' > /dev/null 2>&1 || true

.PHONY: deploy-es-operator
deploy-es-operator: set-max-map-count
	@kubectl create namespace ${ES_OPERATOR_NAMESPACE} 2>&1 | grep -v "already exists" || true
	@kubectl apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/example/prometheus-operator-crd/prometheusrule.crd.yaml
	@kubectl apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/example/prometheus-operator-crd/servicemonitor.crd.yaml
	@kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/master/manifests/01-service-account.yaml -n ${ES_OPERATOR_NAMESPACE}
	@kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/master/manifests/02-role.yaml
	@kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/master/manifests/03-role-bindings.yaml
	@kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/master/manifests/04-crd.yaml -n ${ES_OPERATOR_NAMESPACE}
	@kubectl apply -f https://raw.githubusercontent.com/openshift/elasticsearch-operator/master/manifests/05-deployment.yaml -n ${ES_OPERATOR_NAMESPACE}

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

.PHONY: clean
clean:
	@rm -f deploy/test/*.yaml 
	@if [ -d deploy/test ]; then rmdir deploy/test ; fi
	@kubectl delete -f ./test/cassandra.yml --ignore-not-found=true -n $(STORAGE_NAMESPACE) || true
	@kubectl delete -f ./test/elasticsearch.yml --ignore-not-found=true -n $(STORAGE_NAMESPACE) || true
	@kubectl delete namespace ${ES_OPERATOR_NAMESPACE} || true

.PHONY: crd
crd:
	@kubectl create -f deploy/crds/jaegertracing_v1_jaeger_crd.yaml 2>&1 | grep -v "already exists" || true

.PHONY: ingress
ingress:
	# see https://kubernetes.github.io/ingress-nginx/deploy/#verify-installation
	@kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/nginx-0.18.0/deploy/mandatory.yaml
	@minikube addons enable ingress

.PHONY: generate
generate:
	@operator-sdk generate k8s

.PHONY: test
test: unit-tests e2e-tests

.PHONY: all
all: check format lint build test

.PHONY: ci
ci: ensure-generate-is-noop check format lint build unit-tests

.PHONY: scorecard
scorecard:
	@operator-sdk scorecard --cr-manifest deploy/examples/simplest.yaml --csv-path deploy/olm-catalog/jaeger.clusterserviceversion.yaml --init-timeout 30
