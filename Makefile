VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
CI_COMMIT_SHA ?= $(shell git rev-parse HEAD)
GO_FLAGS ?= GOOS=linux GOARCH=amd64 CGO_ENABLED=0
KUBERNETES_CONFIG ?= "$(HOME)/.kube/config"
WATCH_NAMESPACE ?= default
BIN_DIR ?= "_output/bin"

OPERATOR_NAME ?= jaeger-operator
NAMESPACE ?= "$(USER)"
BUILD_IMAGE ?= "$(NAMESPACE)/$(OPERATOR_NAME):latest"
OUTPUT_BINARY ?= "$(BIN_DIR)/$(OPERATOR_NAME)"
VERSION_PKG ?= "github.com/jaegertracing/jaeger-operator/pkg/cmd/version"

LD_FLAGS ?= "-X $(VERSION_PKG).gitCommit=$(CI_COMMIT_SHA) -X $(VERSION_PKG).buildDate=$(VERSION_DATE)"
PACKAGES := $(shell go list ./cmd/... ./pkg/...)

.DEFAULT_GOAL := build

check:
	@echo Checking...
	@$(foreach file, $(shell go fmt $(PACKAGES) 2>&1), echo "Some files need formatting. Failing." || exit 1)

format:
	@echo Formatting code...
	@go fmt $(PACKAGES)

lint:
	@echo Linting...
	@golint $(PACKAGES)
	@gosec -quiet -exclude=G104 $(PACKAGES) 2>/dev/null

build: format
	@echo Building...
	@${GO_FLAGS} go build -o $(OUTPUT_BINARY) -ldflags $(LD_FLAGS)

docker:
	@docker build -t "$(BUILD_IMAGE)" .

push:
	@echo Pushing image $(BUILD_IMAGE)...
	@docker push $(BUILD_IMAGE) > /dev/null

unit-tests:
	@echo Running unit tests...
	@go test $(PACKAGES) -cover -coverprofile=cover.out

e2e-tests: build docker push
	@echo Running end-to-end tests...
	@cp deploy/rbac.yaml deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml
	@cat deploy/operator.yaml | sed "s~image: jaegertracing\/jaeger-operator\:.*~image: $(BUILD_IMAGE)~gi" >> deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml
	@cat test/elasticsearch.yml >> deploy/test/namespace-manifests.yaml
	@go test ./test/e2e/... -kubeconfig $(KUBERNETES_CONFIG) -namespacedMan ../../deploy/test/namespace-manifests.yaml -globalMan ../../deploy/crd.yaml -root .

run:
	@kubectl create -f deploy/crd.yaml > /dev/null 2>&1 || true
	@OPERATOR_NAME=$(OPERATOR_NAME) KUBERNETES_CONFIG=$(KUBERNETES_CONFIG) WATCH_NAMESPACE=$(WATCH_NAMESPACE) ./_output/bin/jaeger-operator start

es:
	@kubectl create -f ./test/elasticsearch.yml

ingress:
	# see https://kubernetes.github.io/ingress-nginx/deploy/#verify-installation
	@kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/nginx-0.18.0/deploy/mandatory.yaml
	@minikube addons enable ingress

generate:
	@operator-sdk generate k8s

test: unit-tests e2e-tests
all: check format lint build test
ci: check format lint build unit-tests
