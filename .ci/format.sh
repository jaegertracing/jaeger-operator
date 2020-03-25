#!/bin/bash

GOIMPORTS=goimports
command -v ${GOIMPORTS} > /dev/null
if [ $? != 0 ]; then
    if [ -n ${GOPATH} ]; then
        GOIMPORTS="${GOPATH}/bin/goimports"
    fi
fi

${GOIMPORTS} -local "github.com/jaegertracing/jaeger-operator" -l -w $(git ls-files "*\.go" | grep -v vendor)
