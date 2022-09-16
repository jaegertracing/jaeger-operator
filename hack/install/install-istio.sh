#!/bin/bash
VERSION="1.15.0"

echo "Installing istioctl"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

create_bin

PROGRAM="istioctl"
check_tool "$BIN/istioctl" $VERSION "version"


# Download the installer
retry "curl -sLo $BIN/downloadIstio https://istio.io/downloadIstio"
chmod +x $BIN/downloadIstio

# Run the installer
export ISTIO_VERSION=${VERSION}
export TARGET_ARCH=x86_64
cd $BIN
retry "$BIN/downloadIstio"

mv $BIN/istio-${VERSION}/bin/istioctl $BIN/
