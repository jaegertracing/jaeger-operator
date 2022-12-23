include $(shell find tests/e2e -name "Makefile" ! -wholename "tests/e2e/Makefile")

export VERTX_IMG ?= jaegertracing/vertx-create-span:operator-e2e-tests
export ELASTIC_IMG ?= docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.6
export OPERATOR_IMAGE_NEXT ?= ${IMG_PREFIX}/jaeger-operator:next
# get "tests/assert-jobs" directory commit id and use that as build tag to avoid frequent build and push container image into a remote registry
export ASSERT_IMG ?= ${IMG_PREFIX}/asserts-e2e:$(shell git log -n 1 --pretty=format:%h ./tests/assert-jobs/)
export MULTI_ARCH_ASSERT_IMG ?= false

# Use a KIND cluster for the E2E tests
USE_KIND_CLUSTER ?= true
# Skip E2E tests where ES external instance is used
SKIP_ES_EXTERNAL ?= false
# Skip E2e tests where kafka/streaming deployment strategy is used
export SKIP_KAFKA ?= false
export E2E_TESTS_TIMEOUT ?= 330


.PHONY: prepare-e2e-tests
prepare-e2e-tests: start-kind load-operator-image set-image-controller build-assert-job

.PHONY: set-assert-e2e-img-name
set-assert-e2e-img-name:
	$(eval ASSERT_IMG=$(shell ASSERT_IMG=$(ASSERT_IMG) ./tests/build-utils/get-assert-e2e-img.sh))


# Build/pull container images used for the testing ###########################
.PHONY: load-operator-image
load-operator-image: docker kind
ifneq ($(JAEGER_OLM),true)
ifeq ($(USE_KIND_CLUSTER),true)
	$(ECHO) Loading Jager Operator image in KIND cluster
	$(VECHO)$(KIND) load docker-image $(IMG)
else
	$(MAKE) push
endif
	@echo "JAEGER_OLM is true. Not needed to push or load the image anywhere"
endif

build-assert-job: Dockerfile.asserts $(shell find tests/assert-jobs -type f)
	$(ECHO) Building E2E asserts image
	$(VECHO) DOCKER_BUILD_OPTIONS="$(DOCKER_BUILD_OPTIONS)" \
		PLATFORMS="$(PLATFORMS)" \
		MULTI_ARCH_ASSERT_IMG=$(MULTI_ARCH_ASSERT_IMG) \
		./tests/build-utils/build-assert-e2e-img.sh $(ASSERT_IMG) $@


.PHONY: load-assert-job
load-assert-job: build-assert-job set-assert-e2e-img-name
	$(VECHO) USE_KIND_CLUSTER=$(USE_KIND_CLUSTER) \
		MULTI_ARCH_ASSERT_IMG=$(MULTI_ARCH_ASSERT_IMG) \
		./tests/build-utils/push-assert-e2e-img.sh


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
	$(VECHO)./hack/load-kind-image.sh $(OPERATOR_IMAGE_NEXT)
endif

.PHONY: load-elasticsarch-image
load-elasticsarch-image:
ifneq ($(SKIP_ES_EXTERNAL),true)
ifeq ($(USE_KIND_CLUSTER),true)
	$(VECHO)./hack/pull-docker-image.sh $(ELASTIC_IMG)
	$(VECHO)./hack/load-kind-image.sh $(ELASTIC_IMG)
else
	$(ECHO) The Elasticsearch Docker image will be pulled from the registry by the cluster
endif
else
	$(ECHO) Elasticsearch image not loaded because SKIP_ES_EXTERNAL is true
endif
# End build/pull container images used for the testing #######################

.PHONY: render-e2e-tests
render-e2e-tests: list-test-suites
	$(VECHO) for suite in $(TEST_SUITES); do \
		$(MAKE) render-e2e-tests-$$suite ; \
	done

.PHONY: run-e2e-tests
run-e2e-tests: list-test-suites
	$(VECHO) for suite in $(TEST_SUITES); do \
		$(MAKE) run-e2e-tests-$$suite ; \
	done


.PHONY: list-test-suites
list-test-suites:
	$(eval TEST_SUITES := $(shell find tests/e2e -name Makefile | grep  -E -o '(\w|\-)*' | grep -v 'test' | grep -v 'Makefile' | grep -v 'e2e'))

.PHONE: e2e-test-suites
e2e-test-suites: list-test-suites
	@echo "Test suites to run"
	@for test_suite in $(TEST_SUITES) ; do \
		echo -e "\t $$test_suite" ; \
	done

	@echo "You can run a test suite with make run-e2e-tests-<suite name>. E.g: make run-e2e-tests-elasticsearch"

