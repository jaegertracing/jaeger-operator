#!/bin/bash
VERSION="2.3.1"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="kubebuilder"

create_bin

check_tool "$BIN/$PROGRAM" $VERSION "version"

url="https://github.com/kubernetes-sigs/kubebuilder/releases/download/v$VERSION/kubebuilder_${VERSION}_$(go env GOOS)_amd64.tar.gz"

tar_file="/tmp/kubebuilder.tar.gz"
retry "curl -sLo $tar_file $url"
tar -xzf $tar_file -C /tmp/

cp /tmp/kubebuilder_${VERSION}_$(go env GOOS)_amd64/bin/* $BIN/

export PATH=$PATH:$BIN
