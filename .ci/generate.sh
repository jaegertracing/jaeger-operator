#!/bin/bash

OPENAPIGEN=openapi-gen
command -v ${OPENAPIGEN} > /dev/null
if [ $? != 0 ]; then
    if [ -n ${GOPATH} ]; then
        OPENAPIGEN="${GOPATH}/bin/openapi-gen"
    fi
fi

# generate the CRD(s)
operator-sdk generate crds
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate CRDs."
    exit ${RT}
fi

# revert the APIs that are not managed by us
for revert in deploy/crds/kafka.strimzi.io_kafkas_crd.yaml deploy/crds/kafka.strimzi.io_kafkausers_crd.yaml; do
    if [ -f "${revert}" ]; then
        rm "${revert}"
    fi
done

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
