render-e2e-tests-elasticsearch: load-assert-job load-elasticsarch-image
	SKIP_ES_EXTERNAL=$(SKIP_ES_EXTERNAL) ./tests/e2e/elasticsearch/render.sh

run-e2e-tests-elasticsearch: TEST_SUITE_NAME=elasticsearch
run-e2e-tests-elasticsearch: run-suite-tests
