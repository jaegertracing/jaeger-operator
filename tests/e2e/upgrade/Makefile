render-e2e-tests-upgrade: TEST_SUITE_NAME=upgrade
render-e2e-tests-upgrade: generate-kuttl-config generate-e2e-files
	IMG=$(IMG) JAEGER_VERSION=${JAEGER_VERSION} ./tests/e2e/upgrade/render.sh

run-e2e-tests-upgrade: TEST_SUITE_NAME=upgrade
run-e2e-tests-upgrade: run-suite-tests
