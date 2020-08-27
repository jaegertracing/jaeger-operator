#!/bin/bash

git diff -s --exit-code
if [[ $? != 0 ]]; then
    echo "The repository isn't clean. We won't proceed, as we don't know if we should commit those changes or not."
    exit 1
fi

BASE_BUILD_IMAGE=${BASE_BUILD_IMAGE:-"jaegertracing/jaeger-operator"}
BASE_TAG=${BASE_TAG:-$(git describe --tags)}
OPERATOR_VERSION=${OPERATOR_VERSION:-${BASE_TAG}}
OPERATOR_VERSION=$(echo ${OPERATOR_VERSION} | grep -Po "([\d\.]+)")
JAEGER_VERSION=$(echo ${OPERATOR_VERSION} | grep -Po "([\d]+\.[\d]+\.[\d]+)" | head -n 1)
TAG=${TAG:-"v${OPERATOR_VERSION}"}
BUILD_IMAGE=${BUILD_IMAGE:-"${BASE_BUILD_IMAGE}:${OPERATOR_VERSION}"}
CREATED_AT=$(date -u -Isecond)
PREVIOUS_VERSION=$(grep operator= versions.txt | awk -F= '{print $2}')

if [[ ${BASE_TAG} =~ ^release/v.[[:digit:].]+(\-.*)?$ ]]; then
    echo "Releasing ${OPERATOR_VERSION} from ${BASE_TAG}"
else
    echo "The release tag does not match the expected format: ${BASE_TAG}"
    exit 1
fi

if [ "${GH_WRITE_TOKEN}x" == "x" ]; then
    echo "The GitHub write token isn't set. Skipping release process."
    exit 1
fi

# changes to deploy/operator.yaml
sed "s~image: jaegertracing/jaeger-operator.*~image: ${BUILD_IMAGE}~gi" -i deploy/operator.yaml
sed "s~image: jaegertracing/jaeger-agent:.*~image: jaegertracing/jaeger-agent:${JAEGER_VERSION}~gi" -i deploy/operator.yaml

# changes to test/operator.yaml
sed "s~image: jaegertracing/jaeger-operator.*~image: ${BUILD_IMAGE}~gi" -i test/operator.yaml

# change the versions.txt
sed "s~${PREVIOUS_VERSION}~${OPERATOR_VERSION}~gi" -i versions.txt

operator-sdk generate csv \
    --csv-channel=stable \
    --csv-version=${OPERATOR_VERSION} \
    --from-version=${PREVIOUS_VERSION}

# changes to deploy/olm-catalog/jaeger-operator/newversion/...
sed "s~containerImage: docker.io/jaegertracing/jaeger-operator:${PREVIOUS_VERSION}~containerImage: docker.io/jaegertracing/jaeger-operator:${OPERATOR_VERSION}~i" -i deploy/olm-catalog/jaeger-operator/manifests/jaeger-operator.clusterserviceversion.yaml

git diff -s --exit-code
if [[ $? == 0 ]]; then
    echo "No changes detected. Skipping."
else
    git add \
      deploy/operator.yaml \
      deploy/olm-catalog/jaeger-operator/jaeger-operator.package.yaml \
      deploy/olm-catalog/jaeger-operator/manifests/jaeger-operator.clusterserviceversion.yaml \
      test/operator.yaml \
      versions.txt

    git diff -s --exit-code
    if [[ $? != 0 ]]; then
        echo "There are more changes than expected. Skipping the release."
        git diff
        exit 1
    fi

    git config user.email "jaeger-release@jaegertracing.io"
    git config user.name "Jaeger Release"

    git commit -qm "Release ${TAG}"
    git tag ${TAG}
    git push --repo=https://${GH_WRITE_TOKEN}@github.com/jaegertracing/jaeger-operator.git --tags
    git push https://${GH_WRITE_TOKEN}@github.com/jaegertracing/jaeger-operator.git refs/tags/${TAG}:master
fi
