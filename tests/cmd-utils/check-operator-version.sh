#!/bin/bash

if [ "$#" -ne 3 ]; then
    echo "$0 <namespace> <expected version>"
    exit 1
fi

export ROOT_DIR=$(realpath $(dirname ${BASH_SOURCE[0]})/../../)
source $ROOT_DIR/hack/common.sh

jaeger=$1
namespace=$2
expected_version=$3

check_version() {
    export POD=$(get pods -n $namespace -l app.kubernetes.io/name=jaeger-operator -o yaml | yq e ".items[0].metadata.name")
    export VERSION=$(kubectl exec $POD -n $namespace -c jaeger-operator -- ./jaeger-operator version)
}


check_version
while [ "$VERSION" != "$expected_version" ]
do
    echo "Version mismatch: found $VERSION expected $expected_version"
    sleep 5
    check_version
done

echo "Version asserted properly! Expected $expected and found $VERSION"
