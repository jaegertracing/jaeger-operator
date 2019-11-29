#!/bin/bash

GOLINT=golint
EMPTY="[[:space:]]"

command -v ${GOLINT} > /dev/null
if [ $? != 0 ]; then
    if [ -z ${GOPATH} ]; then
        GOLINT="${GOPATH}/bin/golint"
    fi
fi

out=$(${GOLINT} ./... | grep -v pkg/storage/elasticsearch/v1 | grep -v zz_generated)
if [[ $out ]]; then
    echo "$out"
    exit 1
fi