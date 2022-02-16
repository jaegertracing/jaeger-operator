include $(shell find tests/e2e -name "Makefile" ! -wholename "tests/e2e/Makefile")

export ELASTIC_IMG ?= docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.6
export OPERATOR_IMAGE_NEXT ?= ${IMG_PREFIX}/jaeger-operator:next
export ASSERT_IMG ?= ${IMG_PREFIX}/asserts-e2e:$(shell date +%s)
USE_KIND_CLUSTER ?= true

.PHONY: set-assert-e2e-img-name
set-assert-e2e-img-name:
	$(eval ASSERT_IMG=$(shell ASSERT_IMG=$(ASSERT_IMG) ./tests/build-utils/get-assert-e2e-img.sh))

.PHONY: generate-e2e-files
generate-e2e-files: set-image-controller
	$(VECHO)mkdir -p tests/_build/crds tests/_build/manifests
	$(VECHO)$(KUSTOMIZE) build config/default -o tests/_build/manifests/01-jaeger-operator.yaml
	$(VECHO)$(KUSTOMIZE) build config/crd -o tests/_build/crds/


# Build/pull container images used for the testing ###########################
.PHONY: load-operator-image
load-operator-image: docker
ifeq ($(USE_KIND_CLUSTER),true)
	$(ECHO) Loading Jager Operator image in KIND cluster
	$(VECHO)$(KIND) load docker-image $(IMG)
endif

build-assert-job: Dockerfile.asserts $(shell find tests/assert-jobs -type f)
	$(ECHO) Building E2E asserts image
	$(VECHO)docker build -t $(ASSERT_IMG)  -f Dockerfile.asserts .
	$(VECHO)echo $(ASSERT_IMG) > $@


.PHONY: load-assert-job
load-assert-job: build-assert-job set-assert-e2e-img-name
ifeq ($(USE_KIND_CLUSTER),true)
	$(VECHO)./hack/load-kind-image.sh $(ASSERT_IMG) $(KIND)
else
	$(ECHO)Pushing the E2E Test asserts Docker image to the remote repository
	$(VECHO)if [ "$(shell docker manifest inspect $(ASSERT_IMG) 2>/dev/null)" != "" ]; then echo "The image is in the registry. Not pushing"; else docker push $(shell cat build-assert-job); fi;
endif


build-e2e-upgrade-image: Dockerfile $(shell find apis bundle cmd config controllers pkg  -type f)
	@# Instead of calling the "docker" make target as a dependency, it is needed to invoke it
	@# as a target step. The reason is: it is needed to use "JAEGER_VERSION" and overwrite it. This causes
	@# an error in Make
	$(VECHO)$(MAKE) docker JAEGER_VERSION=$(shell .ci/get_test_upgrade_version.sh ${JAEGER_VERSION}) IMG="$(OPERATOR_IMAGE_NEXT)"
	@# Create a file with the name of the target to help make to calculate the timestamp
	$(VECHO)touch $@


.PHONY: load-e2e-upgrade-image
load-e2e-upgrade-image: build-e2e-upgrade-image
ifeq ($(USE_KIND_CLUSTER),true)
	$(VECHO)./hack/load-kind-image.sh $(OPERATOR_IMAGE_NEXT) $(KIND)
endif

.PHONY: load-elasticsarch-image
load-elasticsarch-image:
ifneq ($(SKIP_ES_EXTERNAL),true)
ifeq ($(USE_KIND_CLUSTER),true)
	$(VECHO)./hack/pull-docker-image.sh $(ELASTIC_IMG)
	$(VECHO)./hack/load-kind-image.sh $(ELASTIC_IMG) $(KIND)
else
	$(ECHO) The Elasticsearch Docker image will be pulled from the registry by the cluster
endif
else
	$(ECHO) Elasticsearch image not loaded because SKIP_ES_EXTERNAL is true
endif
# End build/pull container images used for the testing #######################

.PHONY: run-e2e-tests
run-e2e-tests: list-test-suites
	$(VECHO) for suite in $(TEST_SUITES); do \
		$(MAKE) run-e2e-tests-$$suite ; \
	done

.PHONY: render-e2e-tests
render-e2e-tests: generate-e2e-files list-test-suites
	$(VECHO) for suite in $(TEST_SUITES); do \
		$(MAKE) render-e2e-tests-$$suite ; \
	done

.PHONY: list-test-suites
list-test-suites:
	$(VECHO)$(eval TEST_SUITES := $(shell find tests/e2e -name Makefile | grep  -E -o '(\w|\-)*' | grep -v 'test' | grep -v 'Makefile' | grep -v 'e2e'))

run-suite-tests: start-kind kuttl build load-operator-image generate-e2e-files
	$(VECHO)$(MAKE) render-e2e-tests-$(TEST_SUITE_NAME)

ifeq ($(USE_KIND_CLUSTER),true)
	$(VECHO)kubectl wait --timeout=5m --for=condition=available deployment ingress-nginx-controller -n ingress-nginx
	$(VECHO)kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=5m
	$(VECHO)$(MAKE) cert-manager
endif

ifeq ($(OLM),true)
	$(ECHO) Skipping Jaeger Operator installation because OLM=true
else
	$(ECHO) Installing Jaeger Operator...
	$(VECHO)kubectl create namespace observability 2>&1 | grep -v "already exists" || true
	$(VECHO)kubectl apply -f ./tests/_build/manifests/01-jaeger-operator.yaml -n observability
	$(VECHO)kubectl wait --timeout=5m --for=condition=available deployment jaeger-operator -n observability
endif
	$(ECHO) Running $(TEST_SUITE_NAME) E2E tests
	$(VECHO)cd tests/e2e/$(TEST_SUITE_NAME)/_build && $(KUTTL) test $(KUTTL_OPTIONS)

ifeq ($(USE_KIND_CLUSTER),true)
	$(VECHO)$(MAKE) stop-kind
endif
