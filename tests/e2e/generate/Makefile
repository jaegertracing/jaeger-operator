render-e2e-tests-generate: set-assert-e2e-img-name
	JAEGER_VERSION=${JAEGER_VERSION} ./tests/e2e/generate/render.sh

run-e2e-tests-generate: TEST_SUITE_NAME=generate
run-e2e-tests-generate: run-suite-tests
