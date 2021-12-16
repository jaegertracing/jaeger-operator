#!/bin/bash

OPERATOR_VERSION=$(git describe --tags)
echo "${GITHUB_TOKEN}" | gh auth login --with-token

gh config set prompt disabled
gh release create \
    -t "Release ${OPERATOR_VERSION}" \
    "${OPERATOR_VERSION}" \
    'dist/jaeger-operator.yaml#Installation manifest for Kubernetes'
