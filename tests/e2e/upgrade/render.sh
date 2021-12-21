#!/bin/bash

source $(dirname "$0")/../render-utils.sh

cd $SUITE_DIR

echo "Rendering templates for upgrade test"
cd upgrade
$GOMPLATE -f ./deployment-assert.yaml.template -o ./00-assert.yaml
JAEGER_VERSION=$($ROOT_DIR/.ci/get_test_upgrade_version.sh $JAEGER_VERSION) $GOMPLATE -f ./deployment-assert.yaml.template -o ./01-assert.yaml
sed "s~local/jaeger-operator:e2e~local/jaeger-operator:next~gi" $ROOT_DIR/tests/_build/manifests/01-jaeger-operator.yaml > ./operator-upgrade.yaml
