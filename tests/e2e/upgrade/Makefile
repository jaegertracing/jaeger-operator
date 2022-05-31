render-e2e-tests-upgrade: load-assert-job load-e2e-upgrade-image
	SKIP_ES_EXTERNAL=$(SKIP_ES_EXTERNAL) IMG=$(IMG) JAEGER_VERSION=${JAEGER_VERSION} ./tests/e2e/upgrade/render.sh

run-e2e-tests-upgrade: TEST_SUITE_NAME=upgrade
run-e2e-tests-upgrade: run-suite-tests
