#!/bin/bash

kubectl apply -f deploy/webhook-inject-sidecar.yaml
cat deploy/mutating-webhook-configuration.yaml | \
    CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n') \
    envsubst | \
    kubectl apply -f -
