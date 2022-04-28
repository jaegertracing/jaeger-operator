#!/bin/bash
VERSION="3.10.0"

echo "Installing Gomplate"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

PROGRAM="gomplate"

url="https://github.com/hairyhenderson/gomplate/releases/download/v${VERSION}/gomplate_$(go env GOOS)-amd64-slim"

download $PROGRAM $VERSION $url
