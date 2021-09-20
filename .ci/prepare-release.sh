#!/bin/bash

if [[ "${OPERATOR_VERSION}x" == "x" ]]; then
    echo "OPERATOR_VERSION isn't set. Skipping process."
    exit 1
fi

BASE_BUILD_IMAGE=${BASE_BUILD_IMAGE:-"jaegertracing/jaeger-operator"}
TAG=${TAG:-"v${OPERATOR_VERSION}"}
BUILD_IMAGE=${BUILD_IMAGE:-"${BASE_BUILD_IMAGE}:${OPERATOR_VERSION}"}
PREVIOUS_VERSION=$(grep operator= versions.txt | awk -F= '{print $2}')

# changes to deploy/operator.yaml
sed "s~image: jaegertracing/jaeger-operator.*~image: ${BUILD_IMAGE}~gi" -i deploy/operator.yaml

# change the versions.txt, bump only operator version.
sed "s~operator=${PREVIOUS_VERSION}~operator=${OPERATOR_VERSION}~gi" -i versions.txt

mkdir -p deploy/olm-catalog/jaeger-operator/${OPERATOR_VERSION}
cp deploy/olm-catalog/jaeger-operator/manifests/jaeger-operator.clusterserviceversion.yaml \
   deploy/olm-catalog/jaeger-operator/${OPERATOR_VERSION}/jaeger-operator.v${OPERATOR_VERSION}.clusterserviceversion.yaml

operator-sdk generate csv \
    --csv-channel=stable \
    --make-manifests=false \
    --csv-version=${OPERATOR_VERSION}

# changes to deploy/olm-catalog/jaeger-operator/manifests
sed "s~containerImage: docker.io/jaegertracing/jaeger-operator:${PREVIOUS_VERSION}~containerImage: docker.io/jaegertracing/jaeger-operator:${OPERATOR_VERSION}~i" -i deploy/olm-catalog/jaeger-operator/${OPERATOR_VERSION}/jaeger-operator.v${OPERATOR_VERSION}.clusterserviceversion.yaml
sed "s~image: jaegertracing/jaeger-operator:${PREVIOUS_VERSION}~image: jaegertracing/jaeger-operator:${OPERATOR_VERSION}~i" -i deploy/olm-catalog/jaeger-operator/${OPERATOR_VERSION}/jaeger-operator.v${OPERATOR_VERSION}.clusterserviceversion.yaml
sed "s~replaces: jaeger-operator.v.*~replaces: jaeger-operator.v${PREVIOUS_VERSION}~i" -i deploy/olm-catalog/jaeger-operator/${OPERATOR_VERSION}/jaeger-operator.v${OPERATOR_VERSION}.clusterserviceversion.yaml
sed "s~version: ${PREVIOUS_VERSION}~version: ${OPERATOR_VERSION}~i" -i deploy/olm-catalog/jaeger-operator/${OPERATOR_VERSION}/jaeger-operator.v${OPERATOR_VERSION}.clusterserviceversion.yaml
sed "s~name: jaeger-operator.v${PREVIOUS_VERSION}~name: jaeger-operator.v${OPERATOR_VERSION}~i" -i deploy/olm-catalog/jaeger-operator/${OPERATOR_VERSION}/jaeger-operator.v${OPERATOR_VERSION}.clusterserviceversion.yaml

# changes to deploy/olm-catalog/jaeger-operator/jaeger-operator.package.yaml
sed "s~currentCSV: jaeger-operator.v${PREVIOUS_VERSION}~currentCSV: jaeger-operator.v${OPERATOR_VERSION}~i" -i deploy/olm-catalog/jaeger-operator/jaeger-operator.package.yaml

cp deploy/olm-catalog/jaeger-operator/${OPERATOR_VERSION}/jaeger-operator.v${OPERATOR_VERSION}.clusterserviceversion.yaml \
   deploy/olm-catalog/jaeger-operator/manifests/jaeger-operator.clusterserviceversion.yaml