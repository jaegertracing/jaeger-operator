#!/usr/bin/env bash
set -x

[[ -z "$TEST_GROUP" ]] && { echo "TEST_GROUP is undefined, exiting" ; exit 1; }

## Since we're running MiniKube with --vm-driver none, change imagePullPolicy to get the image locally
sed -i 's/imagePullPolicy: Always/imagePullPolicy: Never/g' test/operator.yaml
## remove this once #947 is fixed
export VERBOSE='-v -timeout 20m'
if [ "${TEST_GROUP}" = "es" ]; then
    echo "Running elasticsearch tests"
    which minikube
    minikube version
    kubectl version
    make es
    make e2e-tests-es
else
    echo "Unknown TEST_GROUP [${TEST_GROUP}]"; exit 1
fi

