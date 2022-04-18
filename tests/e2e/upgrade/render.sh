#!/bin/bash

source $(dirname "$0")/../render-utils.sh


if [ $IS_OPENSHIFT = true ]; then
    skip_test "upgrade" "Test not supported in OpenShift"
else
    start_test "upgrade"
    $GOMPLATE -f ./deployment-assert.yaml.template -o ./00-assert.yaml
    JAEGER_VERSION=$($ROOT_DIR/.ci/get_test_upgrade_version.sh $JAEGER_VERSION) $GOMPLATE -f ./deployment-assert.yaml.template -o ./01-assert.yaml
    $GOMPLATE -f ./01-replace.yaml.template -o ./01-replace.yaml
fi
