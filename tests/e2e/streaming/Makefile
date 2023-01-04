render-e2e-tests-streaming: load-assert-job load-elasticsarch-image
	$(VECHO) KAFKA_VERSION=$(KAFKA_VERSION) \
		SKIP_KAFKA=$(SKIP_KAFKA) \
		SKIP_ES_EXTERNAL=$(SKIP_ES_EXTERNAL) \
		./tests/e2e/streaming/render.sh

run-e2e-tests-streaming:
	$(VECHO)KAFKA_OLM=$(KAFKA_OLM) ./hack/run-e2e-test-suite.sh streaming $(USE_KIND_CLUSTER) $(JAEGER_OLM)
