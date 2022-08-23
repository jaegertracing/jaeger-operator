render-e2e-tests-ui: load-assert-job load-elasticsarch-image
	$(VECHO)SKIP_ES_EXTERNAL=$(SKIP_ES_EXTERNAL) ./tests/e2e/ui/render.sh

run-e2e-tests-ui:
	$(VECHO)KAFKA_OLM=$(KAFKA_OLM) ./hack/run-e2e-test-suite.sh ui $(USE_KIND_CLUSTER) $(JAEGER_OLM)
