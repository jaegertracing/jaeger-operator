#!/bin/bash
echo "Installing go dependencies"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

cd $current_dir/../../

retry "go mod download"
