render-e2e-tests-upgrade: load-assert-job load-e2e-upgrade-image
	$(VECHO)SKIP_ES_EXTERNAL=$(SKIP_ES_EXTERNAL) IMG=$(IMG) JAEGER_OPERATOR_VERSION=$(OPERATOR_VERSION) JAEGER_VERSION=${JAEGER_VERSION} ./tests/e2e/upgrade/render.sh

run-e2e-tests-upgrade:
	$(VECHO)KAFKA_OLM=$(KAFKA_OLM) ./hack/run-e2e-test-suite.sh upgrade $(USE_KIND_CLUSTER) $(JAEGER_OLM)
