#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

vendor/k8s.io/code-generator/generate-groups.sh \
deepcopy \
github.com/jaegertracing/jaeger-operator/pkg/generated \
github.com/jaegertracing/jaeger-operator/pkg/apis \
io:v1alpha1 \
--go-header-file "./tmp/codegen/boilerplate.go.txt"
