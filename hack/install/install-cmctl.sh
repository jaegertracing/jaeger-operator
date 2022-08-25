#!/bin/bash
VERSION=$1

echo "Installing cmctl"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="cmctl"

create_bin

url="https://github.com/jetstack/cert-manager/releases/download/v${VERSION}/cmctl-$(go env GOOS)-$(go env GOARCH).tar.gz"


tar_file="/tmp/cmctl.tar.gz"
retry "curl -sLo $tar_file $url"
tar -xzf $tar_file -C /tmp/

cp /tmp/$PROGRAM $BIN/
chmod +x $BIN/$PROGRAM
