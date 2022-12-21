#!/bin/bash
#
# Build the image for the E2E assert job
#
set -xe

image_name=$1
timestamp_file=$2

if [ "$MULTI_ARCH_ASSERT_IMG" = false ]; then
    docker build -t "$image_name" -f Dockerfile.asserts . $DOCKER_BUILD_OPTIONS
else
    # check if the image is in the remote container registry
    manifest=$(docker manifest inspect "$image_name" 2>/dev/null || true)
    if [ -z "$manifest" ]; then
        echo "the e2e test asserts container image is not available in the remote registry. locally building and pushing into remote registry"
        docker buildx build --push \
            --progress=plain \
            --platform ${PLATFORMS} \
            --file Dockerfile.asserts \
            --tag ${image_name} .
    fi
fi   

echo "$image_name" > "$timestamp_file"
