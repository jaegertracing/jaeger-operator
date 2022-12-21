#!/bin/bash
#
# Build the image for the E2E assert job
#
set -xe

image_name=$1
timestamp_file=$2

# check if the image is in the container registry
manifest=$(docker manifest inspect "$image_name" 2>/dev/null || true)
if [ -z "$manifest" ]; then
    echo "the e2e test asserts container image is not available in the remote registry. locally building and pushing into remote registry"
    docker buildx build --push \
        --progress=plain \
        --platform ${PLATFORMS} \
        --file Dockerfile.asserts \
        --tag ${image_name} .
else
    echo "the e2e test asserts container image is available in the remote registry. pulling into local environment"
    docker pull "$image_name"
fi
echo "$image_name" > "$timestamp_file"
