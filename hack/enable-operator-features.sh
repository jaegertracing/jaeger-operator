#!/bin/bash

export ROOT_DIR=$(realpath $(dirname ${BASH_SOURCE[0]})/../)
$ROOT_DIR/hack/install/install-yq.sh > /dev/null
source $ROOT_DIR/hack/common.sh

set -e

if [ ! -z "$JAEGER_OPERATOR_VERBOSITY"  ]; then
    $YQ -i e \
    '.spec.template.spec.containers[0].env += {"name": "LOG-LEVEL", "value": "DEBUG"}' \
    $ROOT_DIR/config/manager/manager.yaml
fi

if [ "$JAEGER_OPERATOR_KAFKA_MINIMAL" = true  ]; then
    $YQ -i e \
        '.spec.template.spec.containers[0].env += {"name": "KAFKA-PROVISIONING-MINIMAL", "value": "true"} ' \
        $ROOT_DIR/config/manager/manager.yaml
fi
