#!/usr/bin/env bash
set -x

[[ -z "$TEST_GROUP" ]] && { echo "TEST_GROUP is undefined, exiting" ; exit 1; }

## Since we're running MiniKube with --vm-driver none, change imagePullPolicy to get the image locally
sed -i 's/imagePullPolicy: Always/imagePullPolicy: Never/g' test/operator.yaml

if [ "${TEST_GROUP}" = "es" ]; then
    echo "Running elasticsearch tests"
    make es
    make e2e-tests-es
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
else
    echo "Unknown TEST_GROUP [${TEST_GROUP}]"; exit 1
fi

