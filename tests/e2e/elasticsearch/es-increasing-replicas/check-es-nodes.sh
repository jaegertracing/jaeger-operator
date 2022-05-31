#!/bin/bash

export ROOT_DIR=$(realpath $(dirname ${BASH_SOURCE[0]})/../../../../../)
source $ROOT_DIR/hack/common.sh

namespace=$1

n=0
while true; do
    echo "Checking if the number of ES instances is the expected"
    kubectl get deployment -l component=elasticsearch -o yaml -n $namespace  | $YQ -e e '.items | length == 2' && break
    n=$((n+1))
    sleep 5
done
