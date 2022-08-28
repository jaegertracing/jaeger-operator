#!/bin/bash
VERSION="3.6.0"

echo "Installing kubebuilder"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="kubebuilder"

create_bin

check_tool "$BIN/$PROGRAM" $VERSION "version"

url="https://github.com/kubernetes-sigs/kubebuilder/releases/download/v$VERSION/kubebuilder_$(go env GOOS)_$(go env GOARCH)"


retry "curl -sLo $BIN/kubebuilder $url"

chmod +x $BIN/kubebuilder

$current_dir/install-etcd.sh
