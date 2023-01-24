#!/bin/bash
VERSION="0.17.0"

echo "Installing kind"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="kind"

url="https://kind.sigs.k8s.io/dl/v$VERSION/kind-$(go env GOOS)-amd64"

download $PROGRAM $VERSION $url
