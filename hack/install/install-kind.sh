#!/bin/bash
echo "Installing kind"

VERSION="0.20.0"
# Kubernetes 1.19 and 1.20 are supported by Kind until 0.17.0
if [ "$KUBE_VERSION" == "1.19" ]  || [ "$KUBE_VERSION" == "1.20"  ]; then
    VERSION="0.17.0"
fi

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="kind"

url="https://kind.sigs.k8s.io/dl/v$VERSION/kind-$(go env GOOS)-amd64"

download $PROGRAM $VERSION $url
