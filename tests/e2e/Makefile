include $(shell find tests/e2e -name "Makefile" ! -wholename "tests/e2e/Makefile")

export VERTX_IMG ?= jaegertracing/vertx-create-span:operator-e2e-tests
export ELASTIC_IMG ?= docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.6
export OPERATOR_IMAGE_NEXT ?= ${IMG_PREFIX}/jaeger-operator:next
export ASSERT_IMG ?= ${IMG_PREFIX}/asserts-e2e:$(shell date +%s)
USE_KIND_CLUSTER ?= true

.PHONY: prepare-e2e-tests
prepare-e2e-tests: kuttl build prepare-e2e-images generate-e2e-files

.PHONY: set-assert-e2e-img-name
set-assert-e2e-img-name:
	$(eval ASSERT_IMG=$(shell ASSERT_IMG=$(ASSERT_IMG) ./tests/build-utils/get-assert-e2e-img.sh))

.PHONY: generate-e2e-files
generate-e2e-files: set-image-controller build-assert-job
	$(VECHO)mkdir -p tests/_build/crds tests/_build/manifests
	$(VECHO)$(KUSTOMIZE) build config/default -o tests/_build/manifests/01-jaeger-operator.yaml
	$(VECHO)$(KUSTOMIZE) build config/crd -o tests/_build/crds/

build-assert-job: Dockerfile.asserts $(shell find tests/assert-jobs -type f)
	docker build -t $(ASSERT_IMG)  -f Dockerfile.asserts . --cache-from	type=local,src=/tmp/.buildx-cache --cache-to type=local,dest=/tmp/.buildx-cache-new,mode=max --load
	$(VECHO)echo $(ASSERT_IMG) > $@


docker-e2e-upgrade-image: Dockerfile $(shell find apis bundle cmd config controllers pkg  -type f)
	@# Instead of calling the "docker" make target as a dependency, it is needed to invoke it
	@# as a target step. The reason is: it is needed to use "JAEGER_VERSION" and overwrite it. This causes
	@# an error in Make
	$(VECHO)$(MAKE) docker JAEGER_VERSION=$(shell .ci/get_test_upgrade_version.sh ${JAEGER_VERSION}) IMG="$(OPERATOR_IMAGE_NEXT)"
	@# Create a file with the name of the target to help make to calculate the timestamp
	$(VECHO)touch $@

.PHONY: prepare-e2e-images
prepare-e2e-images: docker push build-assert-job docker-e2e-upgrade-image set-assert-e2e-img-name
ifeq ($(USE_KIND_CLUSTER),true)
	$(MAKE) cert-manager
	$(ECHO) Loading the Jaeger Operator Docker image in the KIND cluster...
	$(VECHO)$(KIND) load docker-image ${IMG}

	$(ECHO) Loading the VERTX Docker image in the KIND cluster
	$(VECHO)docker pull $(VERTX_IMG)
	$(VECHO)$(KIND) load docker-image $(VERTX_IMG)

	$(ECHO) Loading the Elasticsearch Docker image in the KIND cluster
	$(VECHO)docker pull $(ELASTIC_IMG)
	$(VECHO)$(KIND) load docker-image $(ELASTIC_IMG)

	$(ECHO) Loading the Operator Docker image for the E2E upgrade test in the KIND cluster
	$(VECHO)$(KIND) load docker-image $(OPERATOR_IMAGE_NEXT)

	$(ECHO) Loading the E2E Test asserts Docker image in the KIND cluster...
	$(VECHO)$(KIND) load docker-image $(ASSERT_IMG)
else
	$(ECHO)KIND cluster creation disabled. Skipping...

	$(ECHO)Pushing the E2E Test asserts Docker image to the remote repository
	$(VECHO)if [ "$(shell docker manifest inspect $(ASSERT_IMG) 2>/dev/null)" != "" ]; then echo "The image is in the registry. Not pushing"; else docker push $(shell cat build-assert-job); fi;
endif


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

run-suite-tests: start-kind prepare-e2e-tests
	$(VECHO)$(MAKE) render-e2e-tests-$(TEST_SUITE_NAME)

ifeq ($(USE_KIND_CLUSTER),true)
	$(VECHO)kubectl wait --timeout=5m --for=condition=available deployment ingress-nginx-controller -n ingress-nginx
	$(VECHO)kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=5m
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

clean-generated-files:
	$(VECHO)git clean -Xdf tests/e2e

generate-kuttl-config: set-assert-e2e-img-name clean-generated-files
	$(VECHO)gomplate -f tests/templates/kuttl-test.yaml -o tests/e2e/$(TEST_SUITE_NAME)/kuttl-test.yaml
