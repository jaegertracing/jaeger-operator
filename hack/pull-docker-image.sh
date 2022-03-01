#!/bin/bash

if [ "$#" -ne 1 ]; then
    echo "$0 <image>"
    exit 1
fi

image=$1

n=0
until [ "$n" -ge 5 ]
do
    echo "Pulling image $image. Try $n..."
    docker pull $image && break
    n=$((n+1))
    sleep 5
done
