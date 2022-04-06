#!/bin/bash
VERSION="0.11.1"

echo "Installing kuttl"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="kubectl-kuttl"

url="https://github.com/kudobuilder/kuttl/releases/download/v$VERSION/kubectl-kuttl_${VERSION}_$(go env GOOS)_x86_64"

download $PROGRAM $VERSION $url
