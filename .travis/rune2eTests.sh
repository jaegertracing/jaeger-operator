#!/usr/bin/env bash
set -x
# Confirm we're working
kubectl get all --all-namespaces
kubectl get ingress --all-namespaces
minikube ip

## FIXME hack to workaround docker credentials issue - deploy image directly to minikube
sed -i 's/imagePullPolicy: Always/imagePullPolicy: Never/g' test/operator.yaml
sed -i 's/@docker push/#@docker push/g' Makefile

# Do these first to avoid race conditions
make cassandra
make es
#until kubectl --namespace default get statefulset elasticsearch --output=jsonpath='{.status.readyReplicas}' | grep --quiet 1; do sleep 5;echo "waiting for elasticsearch to be available"; kubectl get statefulsets --namespace default; done
#until kubectl --namespace default get statefulset cassandra --output=jsonpath='{.status.readyReplicas}' | grep --quiet 3; do sleep 5;echo "waiting for cassandra to be available"; kubectl get statefulsets --namespace default; done

make e2e-tests

