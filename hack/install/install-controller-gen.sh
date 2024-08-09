#!/bin/bash
VERSION="0.14.0"

echo "Installing controller-gen"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="controller-gen"

create_bin

export GOBIN=$BIN

check_tool "$BIN/$PROGRAM" "$VERSION" "--version"

retry "go install sigs.k8s.io/controller-tools/cmd/controller-gen@v${VERSION}"
