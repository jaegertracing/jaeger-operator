#!/bin/bash
VERSION="4.20.2"

echo "Installing yq"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="yq"

url="https://github.com/mikefarah/yq/releases/download/v$VERSION/yq_$(go env GOOS)_amd64"

download $PROGRAM $VERSION $url
