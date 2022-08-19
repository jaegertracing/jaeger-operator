render-e2e-tests-sidecar: load-assert-job
	$(VECHO)./tests/e2e/sidecar/render.sh

run-e2e-tests-sidecar:
	$(VECHO)KAFKA_OLM=$(KAFKA_OLM) ./hack/run-e2e-test-suite.sh sidecar $(USE_KIND_CLUSTER) $(JAEGER_OLM)
