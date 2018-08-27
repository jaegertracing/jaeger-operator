#!/bin/bash

kubectl cluster-info
kubectl get deployment
kubectl get pods
kubectl describe pods

if [ -f deploy/test/namespace-manifests.yaml ]; then
    echo "Test namespace manifests:"
    cat deploy/test/namespace-manifests.yaml
else
    echo "Test namespace manifest does not exist."
fi
