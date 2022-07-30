#!/bin/bash
#
# Build the image for the E2E assert job
#
set -xe

image_name=$1
timestamp_file=$2
docker build -t "$image_name" -f Dockerfile.asserts . $DOCKER_BUILD_OPTIONS
echo "$image_name" > "$timestamp_file"
