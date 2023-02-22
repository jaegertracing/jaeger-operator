#!/bin/bash
VERSION="4.5.7"

echo "Installing kustomize"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="kustomize"

create_bin

check_tool "$BIN/$PROGRAM" $VERSION "version"

url="https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${VERSION}/kustomize_v${VERSION}_$(go env GOOS)_amd64.tar.gz"

tar_file="/tmp/kustomize.tar.gz"
retry "curl -sLo $tar_file $url"
tar -xzf $tar_file -C $BIN
