#!/bin/bash

NAMESPACE=${NAMESPACE:-"observability"} # same as the one in the Makefile
if [ "${1}x" != "x" ] ; then
    NAMESPACE="${1}"
fi

kubectl get namespace "${NAMESPACE}" > /dev/null 2>&1
if [ $? == 1 ]; then
    echo "Creating namespace ${NAMESPACE}"
    kubectl create namespace "${NAMESPACE}"
fi

kubectl config set-context $(kubectl config current-context) --namespace="${NAMESPACE}" > /dev/null
if [ $? != 0 ]; then
    echo "Failed to switch to the namespace '${NAMESPACE}'"
fi
