#!/bin/bash
kubectl apply -f ./.ci/minikube-ingress.yaml
kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=150s