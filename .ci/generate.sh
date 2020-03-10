#!/bin/bash

OPENAPIGEN=openapi-gen
command -v ${OPENAPIGEN} > /dev/null
if [ $? != 0 ]; then
    if [ -n ${GOPATH} ]; then
        OPENAPIGEN="${GOPATH}/bin/openapi-gen"
    fi
fi

CONTROLLERGEN=controller-gen
command -v ${CONTROLLERGEN} > /dev/null
if [ $? != 0 ]; then
    if [ -n ${GOPATH} ]; then
        CONTROLLERGEN="${GOPATH}/bin/controller-gen"
    fi
fi

CLIENTGEN=client-gen
command -v ${CLIENTGEN} > /dev/null
if [ $? != 0 ]; then
    if [ -n ${GOPATH} ]; then
        CLIENTGEN="${GOPATH}/bin/client-gen"
    fi
fi

# generate the CRD(s)
${CONTROLLERGEN} crd paths=./pkg/apis/jaegertracing/... crd:maxDescLen=0,trivialVersions=true output:dir=./deploy/crds/
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate CRDs."
    exit ${RT}
fi

# move the generated CRD to the same location the operator-sdk places
mv deploy/crds/jaegertracing.io_jaegers.yaml deploy/crds/jaegertracing.io_jaegers_crd.yaml

# generate the schema validation (openapi) stubs
${OPENAPIGEN} --logtostderr=true -o "" -i ./pkg/apis/jaegertracing/v1 -O zz_generated.openapi -p ./pkg/apis/jaegertracing/v1 -h /dev/null -r "-"
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate the openapi (schema validation) stubs."
    exit ${RT}
fi

# generate the Kubernetes stubs
operator-sdk generate k8s
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate the Kubernetes stubs."
    exit ${RT}
fi

# generate the clients
${CLIENTGEN} \
    --input "jaegertracing/v1" \
    --input-base github.com/jaegertracing/jaeger-operator/pkg/apis \
    --go-header-file /dev/null \
    --output-package github.com/jaegertracing/jaeger-operator/pkg/client \
    --clientset-name versioned \
    --output-base ../../../
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate the Jaeger Tracing clients."
    exit ${RT}
fi