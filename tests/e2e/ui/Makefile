render-e2e-tests-ui: load-assert-job load-elasticsarch-image
	SKIP_ES_EXTERNAL=$(SKIP_ES_EXTERNAL) ./tests/e2e/ui/render.sh

run-e2e-tests-ui: TEST_SUITE_NAME=ui
run-e2e-tests-ui: run-suite-tests
