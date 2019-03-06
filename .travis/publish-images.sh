#!/bin/bash

BASE_BUILD_IMAGE=${BASE_BUILD_IMAGE:-"jaegertracing/jaeger-operator"}
OPERATOR_VERSION=${OPERATOR_VERSION:-$(git describe --tags)}

## if we are on a release tag, let's extract the version number
## the other possible value, currently, is 'master' (or another branch name)
## if we are not running in travis, it fallsback to the `git describe` above
if [[ $OPERATOR_VERSION == release* ]]; then
    OPERATOR_VERSION=$(echo ${OPERATOR_VERSION} | grep -Po "([\d\.]+)")
    MAJOR_MINOR=$(echo ${OPERATOR_VERSION} | awk -F. '{print $1"."$2}')
fi

BUILD_IMAGE=${BUILD_IMAGE:-"${BASE_BUILD_IMAGE}:${OPERATOR_VERSION}"}

if [ "${DOCKER_PASSWORD}x" != "x" -a "${DOCKER_USERNAME}x" != "x" ]; then
    echo "Performing a 'docker login'"
    echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
fi

echo "Building and publishing image ${BUILD_IMAGE}"
make build docker push BUILD_IMAGE=${BUILD_IMAGE}

if [ "${MAJOR_MINOR}x" != "x" ]; then
    MAJOR_MINOR_IMAGE="${BASE_BUILD_IMAGE}:${MAJOR_MINOR}"
    docker tag "${BUILD_IMAGE}" "${MAJOR_MINOR_IMAGE}"
    docker push "${MAJOR_MINOR_IMAGE}"
fi