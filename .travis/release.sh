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

sed "s~image: jaegertracing\/jaeger-operator\:.*~image: ${BUILD_IMAGE}~gi" -i deploy/operator.yaml

git diff -s --exit-code
if [[ $? == 0 ]]; then
    echo "No changes detected. Skipping."
else
    git add deploy/operator.yaml
    git commit -qm "Release ${TAG}" --author="Jaeger Release <jaeger-release@jaegertracing.io>"
    git tag ${TAG}
    git push --repo=https://${GH_WRITE_TOKEN}@github.com/jaegertracing/jaeger-operator.git --tags
    git push --repo=https://${GH_WRITE_TOKEN}@github.com/jaegertracing/jaeger-operator.git HEAD:master
fi
