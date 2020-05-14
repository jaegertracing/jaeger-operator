#!/usr/bin/env bash
set -x

[[ -z "$TEST_GROUP" ]] && { echo "TEST_GROUP is undefined, exiting" ; exit 1; }

## Since we're running MiniKube with --vm-driver none, change imagePullPolicy to get the image locally
sed -i 's/imagePullPolicy: Always/imagePullPolicy: Never/g' test/operator.yaml
## remove this once #947 is fixed
export VERBOSE='-v -timeout 20m'
if [ "${TEST_GROUP}" = "es" ]; then
    echo "Running elasticsearch tests"
    make es
    make e2e-tests-es
elif [ "${TEST_GROUP}" = "es-otel" ]; then
    echo "Running elasticsearch tests with OTEL collector"
    export USE_OTEL_COLLECTOR=true
    make es
    make e2e-tests-es
elif [ "${TEST_GROUP}" = "es-self-provisioned" ]; then
    echo "Running self provisioned elasticsearch tests"
    make e2e-tests-self-provisioned-es
    res=$?
    if [[ ${res} -ne 0 ]]; then
        kubectl log deploy/elasticsearch-operator -n openshift-logging
    fi
    exit ${res}
elif [ "${TEST_GROUP}" = "smoke" ]
then
    echo "Running Smoke Tests"
    make e2e-tests-smoke
elif [ "${TEST_GROUP}" = "cassandra" ]
then
    echo "Running Cassandra Tests"
    make cassandra
    make e2e-tests-cassandra
elif [ "${TEST_GROUP}" = "streaming" ]
then
    echo "Running Streaming Tests"
    make e2e-tests-streaming
elif [ "${TEST_GROUP}" = "streaming-otel" ]
then
    echo "Running Streaming Tests with OTEL collector"
    export USE_OTEL_COLLECTOR=true
    make e2e-tests-streaming
elif [ "${TEST_GROUP}" = "examples1" ]
then
    echo "Running Examples1 Tests"
    make e2e-tests-examples1
elif [ "${TEST_GROUP}" = "examples2" ]
then
    echo "Running Examples2 Tests"
    make e2e-tests-examples2
elif [ "${TEST_GROUP}" = "es-token-propagation" ]
then
    echo "Running token propagation tests"
    make e2e-tests-token-propagation-es
elif [ "${TEST_GROUP}" = "generate" ]
then
    echo "Running CLI manifest generation tests"
    make e2e-tests-generate
else
    echo "Unknown TEST_GROUP [${TEST_GROUP}]"; exit 1
fi

