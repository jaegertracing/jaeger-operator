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
fi
