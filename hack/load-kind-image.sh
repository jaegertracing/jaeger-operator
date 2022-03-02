#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "$0 <image> <location of kind>"
    exit 1
fi

image=$1
kind=$2

if [ "$(kubectl get no -o yaml | grep -F $image)" ]; then
    echo "The image $image is in the KIND cluster. It is not needed to load it again"
else
    echo "Loading the $image container image in the KIND cluster"
    $kind load docker-image "$image"
fi
