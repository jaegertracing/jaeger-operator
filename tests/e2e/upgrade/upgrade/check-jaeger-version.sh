#!/bin/bash

if [ "$#" -ne 3 ]; then
    echo "$0 <Jaeger name> <namespace> <expected version>"
    exit 1
fi

check_version() {
    export VERSION=$(kubectl get jaeger $jaeger -o jsonpath="{.status.version}" -n $namespace)
}


jaeger=$1
namespace=$2
expected_version=$3

check_version
while [ "$VERSION" != "$expected_version" ]
do
    echo "Version mismatch: found $VERSION expected $expected_version"
    sleep 5
    check_version
done

echo "Version asserted properly! Expected $expected and found $VERSION"
