#!/bin/bash

BASE_BUILD_IMAGE=${BASE_BUILD_IMAGE:-"jaegertracing/jaeger-operator"}
OPERATOR_VERSION=${OPERATOR_VERSION:-$(git describe --tags)}

## if we are on a release tag, let's extract the version number
## the other possible value, currently, is 'main' (or another branch name)
## if we are not running in the CI, it fallsback to the `git describe` above
if [[ $OPERATOR_VERSION == v* ]]; then
    OPERATOR_VERSION=$(echo ${OPERATOR_VERSION} | grep -Po "([\d\.]+)")
    MAJOR_MINOR=$(echo ${OPERATOR_VERSION} | awk -F. '{print $1"."$2}')
fi

BUILD_IMAGE=${BUILD_IMAGE:-"${BASE_BUILD_IMAGE}:${OPERATOR_VERSION}"}
DOCKER_USERNAME=${DOCKER_USERNAME:-"jaegertracingbot"}

if [ "${DOCKER_PASSWORD}x" != "x" -a "${DOCKER_USERNAME}x" != "x" ]; then
    echo "Performing a 'docker login'"
    echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
fi

IMAGE_TAGS="--tag ${BUILD_IMAGE}"

if [ "${MAJOR_MINOR}x" != "x" ]; then
    MAJOR_MINOR_IMAGE="${BASE_BUILD_IMAGE}:${MAJOR_MINOR}"
    IMAGE_TAGS="${IMAGE_TAGS} --tag ${MAJOR_MINOR_IMAGE}"
fi

## now, push to quay.io
if [ "${QUAY_PASSWORD}x" != "x" -a "${QUAY_USERNAME}x" != "x" ]; then
    echo "Performing a 'docker login' for Quay"
    echo "${QUAY_PASSWORD}" | docker login -u "${QUAY_USERNAME}" quay.io --password-stdin

    echo "Tagging ${BUILD_IMAGE} as quay.io/${BUILD_IMAGE}"
    IMAGE_TAGS="${IMAGE_TAGS} --tag quay.io/${BUILD_IMAGE}"

    if [ "${MAJOR_MINOR_IMAGE}x" != "x" ]; then
        IMAGE_TAGS="${IMAGE_TAGS} --tag quay.io/${MAJOR_MINOR_IMAGE}"
    fi
fi

echo "Building with tags ${IMAGE_TAGS}"
IMAGE_TAGS=${IMAGE_TAGS} make dockerx
