#!/usr/bin/env bash

# Confirm we're working
kubectl get all --all-namespaces
kubectl get ingress --all-namespaces
minikube ip

make e2e-tests-smoke
