render-e2e-tests-generate: build load-assert-job
	$(VECHO)JAEGER_VERSION=${JAEGER_VERSION} ./tests/e2e/generate/render.sh

run-e2e-tests-generate:
	$(VECHO)KAFKA_OLM=$(KAFKA_OLM) ./hack/run-e2e-test-suite.sh generate $(USE_KIND_CLUSTER) $(JAEGER_OLM)
