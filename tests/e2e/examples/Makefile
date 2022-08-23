render-e2e-tests-examples: load-assert-job load-elasticsarch-image load-elasticsarch-image
	$(VECHO)./tests/e2e/examples/render.sh

run-e2e-tests-examples:
	$(VECHO)KAFKA_OLM=$(KAFKA_OLM) ./hack/run-e2e-test-suite.sh examples $(USE_KIND_CLUSTER) $(JAEGER_OLM)
