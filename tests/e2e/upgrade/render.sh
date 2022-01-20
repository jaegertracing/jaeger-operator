#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "upgrade"
$GOMPLATE -f ./deployment-assert.yaml.template -o ./00-assert.yaml
JAEGER_VERSION=$($ROOT_DIR/.ci/get_test_upgrade_version.sh $JAEGER_VERSION) $GOMPLATE -f ./deployment-assert.yaml.template -o ./01-assert.yaml
sed "s~$IMG~$OPERATOR_IMAGE_NEXT~gi" $ROOT_DIR/tests/_build/manifests/01-jaeger-operator.yaml > ./operator-upgrade.yaml
