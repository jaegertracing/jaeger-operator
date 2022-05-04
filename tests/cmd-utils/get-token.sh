#!/bin/bash

if [ "$#" -lt 2 ]; then
    echo "$0 <namespace> <Jaeger name> <output file (optional)>"
    exit 1
fi

export NAMESPACE=$1
export JAEGER_NAME=$2
export OUTPUT_FILE=$3

set -e

export ROOT_DIR=$(realpath $(dirname ${BASH_SOURCE[0]})/../../)
source $ROOT_DIR/hack/common.sh

# Ensure the tools are installed
$ROOT_DIR/hack/install/install-gomplate.sh > /dev/null
$ROOT_DIR/hack/install/install-yq.sh > /dev/null


if [ -z "$SERVICE_ACCOUNT_NAME" ]; then
    export SERVICE_ACCOUNT_NAME="automation-sa"
fi

$GOMPLATE -f $TEMPLATES_DIR/openshift/configure-jaeger-sa.yaml.template -o /tmp/jaeger-sa.yaml
kubectl apply -f /tmp/jaeger-sa.yaml -n $NAMESPACE > /dev/null

# This takes some time
sleep 5

SECRET_NAME=$(kubectl get sa $SERVICE_ACCOUNT_NAME -o yaml -n $NAMESPACE | $YQ eval '.secrets[] | select( .name == "*-token-*")'.name)
SECRET=$(kubectl get secret $SECRET_NAME -n $NAMESPACE -o jsonpath='{.data.token}' |  base64 -d)

if [ ! -z $OUTPUT_FILE ]; then
    echo -n $SECRET > $OUTPUT_FILE
else
    echo $SECRET
fi
