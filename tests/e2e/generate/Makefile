render-e2e-tests-generate: TEST_SUITE_NAME=generate
render-e2e-tests-generate: generate-kuttl-config
	JAEGER_VERSION=${JAEGER_VERSION} ./tests/e2e/generate/render.sh

run-e2e-tests-generate: TEST_SUITE_NAME=generate
run-e2e-tests-generate: run-suite-tests
