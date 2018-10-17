VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GO_FLAGS ?= GOOS=linux GOARCH=amd64 CGO_ENABLED=0
KUBERNETES_CONFIG ?= "$(HOME)/.kube/config"
WATCH_NAMESPACE ?= default
BIN_DIR ?= "_output/bin"

OPERATOR_NAME ?= jaeger-operator
NAMESPACE ?= "$(USER)"
BUILD_IMAGE ?= "$(NAMESPACE)/$(OPERATOR_NAME):latest"
OUTPUT_BINARY ?= "$(BIN_DIR)/$(OPERATOR_NAME)"
VERSION_PKG ?= "github.com/jaegertracing/jaeger-operator/pkg/version"
JAEGER_VERSION ?= "$(shell grep -v '\#' jaeger.version)"
OPERATOR_VERSION ?= "$(shell git describe --tags)"

LD_FLAGS ?= "-X $(VERSION_PKG).version=$(OPERATOR_VERSION) -X $(VERSION_PKG).buildDate=$(VERSION_DATE) -X $(VERSION_PKG).defaultJaeger=$(JAEGER_VERSION)"
PACKAGES := $(shell go list ./cmd/... ./pkg/...)

.DEFAULT_GOAL := build

.PHONY: check
check:
	@echo Checking...
	@$(foreach file, $(shell go fmt $(PACKAGES) 2>&1), echo "Some files need formatting. Failing." || exit 1)

.PHONY: format
format:
	@echo Formatting code...
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
	@docker build -t "$(BUILD_IMAGE)" .

.PHONY: push
push:
	@echo Pushing image $(BUILD_IMAGE)...
	@docker push $(BUILD_IMAGE) > /dev/null

.PHONY: unit-tests
unit-tests:
	@echo Running unit tests...
	@go test $(PACKAGES) -cover -coverprofile=cover.out

.PHONY: e2e-tests
e2e-tests: es crd build docker push
	@echo Running end-to-end tests...
	@cp deploy/rbac.yaml deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml
	@cat deploy/operator.yaml | sed "s~image: jaegertracing\/jaeger-operator\:.*~image: $(BUILD_IMAGE)~gi" >> deploy/test/namespace-manifests.yaml
	@go test ./test/e2e/... -kubeconfig $(KUBERNETES_CONFIG) -namespacedMan ../../deploy/test/namespace-manifests.yaml -globalMan ../../deploy/crd.yaml -root .

.PHONY: crd
run: crd
	@bash -c 'trap "exit 0" INT; OPERATOR_NAME=${OPERATOR_NAME} KUBERNETES_CONFIG=${KUBERNETES_CONFIG} WATCH_NAMESPACE=${WATCH_NAMESPACE} go run -ldflags ${LD_FLAGS} main.go start'

.PHONY: es
es:
	@kubectl create -f ./test/elasticsearch.yml 2>&1 | grep -v "already exists" || true

.PHONY: cassandra
cassandra:
	@kubectl create -f ./test/cassandra.yml 2>&1 | grep -v "already exists" || true

.PHONY: crd
crd:
	@kubectl create -f deploy/crd.yaml 2>&1 | grep -v "already exists" || true

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
ci: check format lint build unit-tests
