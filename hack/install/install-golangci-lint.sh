#!/bin/bash
VERSION="1.55.2"

echo "Installing golangci-lint"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

create_bin

export PROGRAM="golangci-lint"

check_tool "$BIN/$PROGRAM" "$VERSION" "version"

export GOBIN=$BIN
retry "go install github.com/golangci/golangci-lint/cmd/golangci-lint@v${VERSION}"
