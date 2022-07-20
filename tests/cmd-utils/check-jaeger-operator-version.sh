#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "$0 <namespace> <expected version>"
    exit 1
fi

export ROOT_DIR=$(realpath $(dirname ${BASH_SOURCE[0]})/../../)
source $ROOT_DIR/hack/common.sh

namespace=$1
expected_version=$2

SLEEP_TIME=5

check_version() {
    POD=""
    while [ -z "$POD" ]; do
        POD=$(kubectl get pods -n $namespace -l name=jaeger-operator -o yaml | $YQ e ".items[0].metadata.name")
        if [ -z "$POD" ]; then
            echo "No pods found for the Jaeger Operator. Trying again in $SLEEP_TIME seconds..."
            time $SLEEP_TIME
        fi
    done
    export VERSION=$(kubectl exec $POD -n $namespace -c jaeger-operator -- ./jaeger-operator version | $YQ -P ".jaeger-operator"| grep -Eo '[0-9]+\.[0-9]+\.[0-9]+')
}


check_version
while [ "$VERSION" != "$expected_version" ]
do
    if [ -z "$VERSION" ]; then
        echo "Version was not found! Check if your Jaeger Operator image was built properly"
    else
        echo "Version mismatch: found $VERSION expected $expected_version"
    fi
    sleep $SLEEP_TIME
    check_version
done

echo "Version asserted properly! Expected $expected and found $VERSION"
