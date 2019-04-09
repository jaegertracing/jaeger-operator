#!/usr/bin/env bash
set -x
# Confirm we're working
kubectl get all --all-namespaces
kubectl get ingress --all-namespaces
minikube ip

## FIXME hack to workaround docker credentials issue - deploy image directly to minikube
sed -i 's/imagePullPolicy: Always/imagePullPolicy: Never/g' test/operator.yaml
sed -i 's/@docker push/#@docker push/g' Makefile
#eval $(minikube docker-env)

make e2e-tests-smoke
