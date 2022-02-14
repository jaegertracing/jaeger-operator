render-e2e-tests-outside-cluster: set-assert-e2e-img-name
	SKIP_ES_EXTERNAL=$(SKIP_ES_EXTERNAL) ./tests/e2e/outside-cluster/render.sh

run-e2e-tests-outside-cluster: TEST_SUITE_NAME=outside-cluster
run-e2e-tests-outside-cluster: run-suite-tests
