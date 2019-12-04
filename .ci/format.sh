#!/bin/bash

${GOPATH}/bin/goimports -local "github.com/jaegertracing/jaeger-operator" -l -w $(git ls-files "*\.go" | grep -v vendor)
