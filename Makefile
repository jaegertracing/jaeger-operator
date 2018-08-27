VERSION_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
CI_COMMIT_SHA ?= $(shell git rev-parse HEAD)
GO_FLAGS ?= GOOS=linux GOARCH=amd64 CGO_ENABLED=0
OPERATOR_NAME=jaeger-operator
NAMESPACE="$(USER)"
BUILD_IMAGE="$(NAMESPACE)/$(OPERATOR_NAME):latest"

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
	@operator-sdk build $(BUILD_IMAGE) > /dev/null

push:
	@echo Pushing image...
	@docker push $(BUILD_IMAGE) > /dev/null

unit-tests:
	@echo Running unit tests...
	@go test $(PACKAGES) -cover -coverprofile=cover.out

e2e-tests: build push
	@echo Running end-to-end tests...
	@operator-sdk test --test-location ./test/e2e

run:
	@kubectl create -f deploy/crd.yaml > /dev/null 2>&1 || true
	@OPERATOR_NAME=$(OPERATOR_NAME) operator-sdk up local

es:
	@kubectl create -f ./test/elasticsearch.yml

ingress:
	# see https://kubernetes.github.io/ingress-nginx/deploy/#verify-installation
	@kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/nginx-0.18.0/deploy/mandatory.yaml
	@minikube addons enable ingress

generate:
	@operator-sdk generate k8s

test: unit-tests e2e-tests
all: check format lint test
ci: all