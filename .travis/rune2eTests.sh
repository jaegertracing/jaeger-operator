#!/usr/bin/env bash

# Confirm we're working
kubectl get all --all-namespaces
kubectl get ingress --all-namespaces
minikube ip

if [ "${DOCKER_PASSWORD}x" != "x" -a "${DOCKER_USERNAME}x" != "x" ]; then
    echo "Performing a 'docker login'"
    echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
    export NAMESPACE=${DOCKER_USERNAME}
fi


make e2e-tests-smoke
