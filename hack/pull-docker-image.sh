#!/bin/bash

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install/install-utils.sh

if [ "$#" -ne 1 ]; then
    echo "$0 <image>"
    exit 1
fi

image=$1

retry "docker pull $image"
