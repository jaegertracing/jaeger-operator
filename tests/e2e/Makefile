include tests/e2e/allinone/Makefile
include tests/e2e/cassandra/Makefile
include tests/e2e/elasticsearch/Makefile
include tests/e2e/examples/Makefile
include tests/e2e/generate/Makefile
include tests/e2e/istio/Makefile
include tests/e2e/outside-cluster/Makefile
include tests/e2e/sidecard/Makefile
include tests/e2e/smoke/Makefile
include tests/e2e/streaming/Makefile
include tests/e2e/upgrade/Makefile

.PHONY: prepare-e2e-tests
prepare-e2e-tests: IMG=local/jaeger-operator:e2e
prepare-e2e-tests: kuttl build prepare-e2e-images generate-e2e-files

.PHONY: build-assert-job
build-assert-job:
	$(VECHO)docker build -t local/asserts:e2e -f Dockerfile.asserts .

.PHONY: generate-e2e-files
generate-e2e-files: set-image-controller
	mkdir -p tests/_build/crds tests/_build/manifests
	$(KUSTOMIZE) build config/default -o tests/_build/manifests/01-jaeger-operator.yaml
	$(KUSTOMIZE) build config/crd -o tests/_build/crds/

.PHONY: prepare-e2e-images
prepare-e2e-images:
prepare-e2e-images: docker build-assert-job
	$(VECHO)docker pull jaegertracing/vertx-create-span:operator-e2e-tests
	$(VECHO)docker pull docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.6

	# Image for the upgrade E2E test
	$(VECHO)docker build --build-arg=GOPROXY=${GOPROXY}  --build-arg VERSION_PKG=${VERSION_PKG} --build-arg=JAEGER_VERSION=$(shell .ci/get_test_upgrade_version.sh ${JAEGER_VERSION}) --file Dockerfile -t "local/jaeger-operator:next" .

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
	$(eval TEST_SUITES := $(shell find tests/e2e -name Makefile | grep  -E -o '(\w|\-)*' | grep -v 'test' | grep -v 'Makefile' | grep -v 'e2e'))

.PHONE: e2e-test-suites
e2e-test-suites: list-test-suites
	@echo "Test suites to run"
	@for test_suite in $(TEST_SUITES) ; do \
		echo -e "\t $$test_suite" ; \
	done

	@echo "You can run a test suite with make run-e2e-tests-<suite name>. E.g: make run-e2e-tests-smoke"

run-suite-tests:
	$(VECHO)$(MAKE) prepare-e2e-tests render-e2e-tests-$(TEST_SUITE_NAME) start-kind
	$(VECHO)cd tests/e2e/$(TEST_SUITE_NAME) && $(KUTTL) test $(KUTTL_OPTIONS)
	$(VECHO)$(MAKE) stop-kind

generate-kuttl-config:
	$(VECHO)gomplate -f tests/templates/kuttl-test.yaml -o tests/e2e/$(TEST_SUITE_NAME)/kuttl-test.yaml
