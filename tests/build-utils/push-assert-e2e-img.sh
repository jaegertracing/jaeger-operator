#!/bin/bash
#
# Push the assert E2E image from local to the correct place
#
set -e

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
root_dir=$current_dir/../../

ASSERT_IMG="$("$current_dir"/get-assert-e2e-img.sh)"

if [ "$USE_KIND_CLUSTER" = true ]; then
    "$root_dir"/hack/load-kind-image.sh "$ASSERT_IMG"
elif [ "$MULTI_ARCH_ASSERT_IMG" = false ]; then
    echo "Pushing the E2E Test asserts Docker image to the remote registry"

    # Check if the image is in the container registry
    manifest=$(docker manifest inspect "$ASSERT_IMG" 2>/dev/null || true)
    if [ -z "$manifest" ]; then
        docker push "$ASSERT_IMG"
    else
        echo "The image $ASSERT_IMG is in the registry. Not pushing"
    fi
fi
