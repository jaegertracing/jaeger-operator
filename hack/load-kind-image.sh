#!/bin/bash

if [ "$#" -ne 1 ]; then
    echo "$0 <image>"
    exit 1
fi

image=$1

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
$current_dir/install/install-kind.sh
kind=$current_dir/../bin/kind

if [ "$(kubectl get no -o yaml | grep -F $image)" ]; then
    echo "The image $image is in the KIND cluster. It is not needed to load it again"
else
    echo "Loading the $image container image in the KIND cluster"
    $kind load docker-image "$image"
fi
