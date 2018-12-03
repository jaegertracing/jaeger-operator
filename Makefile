VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GO_FLAGS ?= GOOS=linux GOARCH=amd64 CGO_ENABLED=0
KUBERNETES_CONFIG ?= "$(HOME)/.kube/config"
WATCH_NAMESPACE ?= default
BIN_DIR ?= "build/_output/bin"

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

.PHONY: ensure-generate-is-noop
ensure-generate-is-noop: generate
	@git diff -s --exit-code pkg/apis/io/v1alpha1/zz_generated.deepcopy.go || (echo "Build failed: a model has been changed but the deep copy functions aren't up to date. Run 'make generate' and update your PR." && exit 1)

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
e2e-tests: deploy-cassandra deploy-es crd build docker push
	mkdir -p deploy/test
	@echo Running end-to-end tests...

	@cp deploy/role_binding.yaml deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml

	@cat deploy/role.yaml >> deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml

	@cat deploy/service_account.yaml >> deploy/test/namespace-manifests.yaml
	@echo "---" >> deploy/test/namespace-manifests.yaml

	@cat deploy/operator.yaml | sed "s~image: jaegertracing\/jaeger-operator\:.*~image: $(BUILD_IMAGE)~gi" >> deploy/test/namespace-manifests.yaml
	@go test ./test/e2e/... -kubeconfig $(KUBERNETES_CONFIG) -namespacedMan ../../deploy/test/namespace-manifests.yaml -globalMan ../../deploy/crds/io_v1alpha1_jaeger_crd.yaml -root .

.PHONY: run
run: crd
	@bash -c 'trap "exit 0" INT; OPERATOR_NAME=${OPERATOR_NAME} KUBERNETES_CONFIG=${KUBERNETES_CONFIG} WATCH_NAMESPACE=${WATCH_NAMESPACE} go run -ldflags ${LD_FLAGS} main.go start'

.PHONY: run-openshift
run-openshift: crd
	@bash -c 'trap "exit 0" INT; OPERATOR_NAME=${OPERATOR_NAME} KUBERNETES_CONFIG=${KUBERNETES_CONFIG} WATCH_NAMESPACE=${WATCH_NAMESPACE} go run -ldflags ${LD_FLAGS} main.go start --platform=openshift'

.PHONY: deploy-es
deploy-es:
	@kubectl create -f ./test/elasticsearch.yml || true

.PHONY: deploy-cassandra
deploy-cassandra:
	@kubectl create -f ./test/cassandra.yml || true

.PHONY: clean-deploy
clean-deploy:
	@kubectl delete -f ./test/cassandra.yml || true
	@kubectl delete -f ./test/elasticsearch.yml || true
	@kubectl delete -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/nginx-0.18.0/deploy/mandatory.yaml || true

.PHONY: crd
crd:
	@kubectl create -f deploy/crds/io_v1alpha1_jaeger_crd.yaml | grep -v "already exists" || true

.PHONY: deploy-ingress
deploy-ingress:
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
