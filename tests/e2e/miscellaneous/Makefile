render-e2e-tests-miscellaneous: set-assert-e2e-img-name load-assert-job
	$(VECHO)SKIP_ES_EXTERNAL=$(SKIP_ES_EXTERNAL) ./tests/e2e/miscellaneous/render.sh

run-e2e-tests-miscellaneous:
	$(VECHO)KAFKA_OLM=$(KAFKA_OLM) ./hack/run-e2e-test-suite.sh miscellaneous $(USE_KIND_CLUSTER) $(JAEGER_OLM)
