render-e2e-tests-elasticsearch: load-assert-job load-elasticsarch-image
	$(VECHO) SKIP_ES_EXTERNAL=$(SKIP_ES_EXTERNAL) \
		KAFKA_VERSION=$(KAFKA_VERSION) \
		SKIP_KAFKA=$(SKIP_KAFKA) \
		./tests/e2e/elasticsearch/render.sh

run-e2e-tests-elasticsearch:
	$(VECHO)KAFKA_OLM=$(KAFKA_OLM) ./hack/run-e2e-test-suite.sh elasticsearch $(USE_KIND_CLUSTER) $(JAEGER_OLM)
