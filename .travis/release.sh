#!/bin/bash

git diff -s --exit-code
if [[ $? != 0 ]]; then
    echo "The repository isn't clean. We won't proceed, as we don't know if we should commit those changes or not."
    exit 1
fi

BASE_BUILD_IMAGE=${BASE_BUILD_IMAGE:-"jaegertracing/jaeger-operator"}
OPERATOR_VERSION=${OPERATOR_VERSION:-$(git describe --tags)}
OPERATOR_VERSION=$(echo ${OPERATOR_VERSION} | grep -Po "([\d\.]+)")
TAG=${TAG:-"v${OPERATOR_VERSION}"}
BUILD_IMAGE=${BUILD_IMAGE:-"${BASE_BUILD_IMAGE}:${OPERATOR_VERSION}"}

# changes to deploy/operator.yaml
sed "s~image: jaegertracing/jaeger-operator.*~image: ${BUILD_IMAGE}~gi" -i deploy/operator.yaml

# changes to deploy/olm-catalog/jaeger.package.yaml
sed "s/currentCSV: jaeger-operator.*/currentCSV: jaeger-operator.v${OPERATOR_VERSION}/gi" -i deploy/olm-catalog/jaeger.package.yaml

# changes to deploy/olm-catalog/jaeger-operator.csv.yaml
sed "s~containerImage: docker.io/jaegertracing/jaeger-operator.*~containerImage: docker.io/${BUILD_IMAGE}~gi" -i deploy/olm-catalog/jaeger-operator.csv.yaml
sed "s/name: jaeger-operator\.v.*/name: jaeger-operator.v${OPERATOR_VERSION}/gi" -i deploy/olm-catalog/jaeger-operator.csv.yaml
sed "s~image: jaegertracing/jaeger-operator.*~image: ${BUILD_IMAGE}~gi" -i deploy/olm-catalog/jaeger-operator.csv.yaml
## there's a "version: v1" there somewhere that we want to avoid
sed -E "s/version: ([0-9\.]+).*/version: ${OPERATOR_VERSION}/gi" deploy/olm-catalog/jaeger-operator.csv.yaml

# changes to test/operator.yaml
sed "s~image: jaegertracing/jaeger-operator.*~image: ${BUILD_IMAGE}~gi" -i test/operator.yaml

git diff -s --exit-code
if [[ $? == 0 ]]; then
    echo "No changes detected. Skipping."
else
    git add \
      deploy/operator.yaml \
      deploy/olm-catalog/jaeger.package.yaml \
      deploy/olm-catalog/jaeger-operator.csv.yaml \
      test/operator.yaml

    git commit -qm "Release ${TAG}" --author="Jaeger Release <jaeger-release@jaegertracing.io>"
    git tag ${TAG}
    git push --repo=https://${GH_WRITE_TOKEN}@github.com/jaegertracing/jaeger-operator.git --tags
    git push https://${GH_WRITE_TOKEN}@github.com/jaegertracing/jaeger-operator.git refs/tags/${TAG}:master
fi
