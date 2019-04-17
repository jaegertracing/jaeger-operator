#!/usr/bin/env bash
set -x
# Confirm we're working
kubectl get all --all-namespaces

## Since we're running MiniKube with --vm-driver none, change imagePullPolicy to get the image locally
sed -i 's/imagePullPolicy: Always/imagePullPolicy: Never/g' test/operator.yaml

# Do these first to avoid race conditions
make cassandra
make es

make e2e-tests

