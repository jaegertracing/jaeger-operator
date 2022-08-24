#!/bin/bash
VERSION="0.1.12"

echo "Installing goimports"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

create_bin

export GOBIN=$BIN
retry "go install golang.org/x/tools/cmd/goimports@v${VERSION}"
