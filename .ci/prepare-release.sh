#!/bin/bash

if [[ "${OPERATOR_VERSION}x" == "x" ]]; then
    echo "OPERATOR_VERSION isn't set. Skipping process."
    exit 1
fi


PREVIOUS_VERSION=$(grep operator= versions.txt | awk -F= '{print $2}')

# change the versions.txt, bump only operator version.
sed "s~operator=${PREVIOUS_VERSION}~operator=${OPERATOR_VERSION}~gi" -i versions.txt

# changes to deploy/operator.yaml
sed "s~replaces: jaeger-operator.v.*~replaces: jaeger-operator.v${PREVIOUS_VERSION}~i" -i config/manifests/bases/jaeger-operator.clusterserviceversion.yaml

VERSION=${OPERATOR_VERSION} USER=jaegertracing make bundle


