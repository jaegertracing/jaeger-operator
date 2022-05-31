#!/bin/bash

source $(dirname "$0")/../render-utils.sh


if [ $IS_OPENSHIFT = true ]; then
    skip_test "upgrade" "Test not supported in OpenShift"
else
    start_test "upgrade"

    render_install_jaeger "my-jaeger" "allInOne" "00"

    # Check the deployment using the correct image
    $GOMPLATE -f ./deployment-assert.yaml.template -o ./01-assert.yaml
    $GOMPLATE -f ./02-check-jaeger-version.yaml.template -o ./02-check-jaeger-version.yaml

    $GOMPLATE -f ./replace.yaml.template -o ./03-upgrade-operator.yaml
    JAEGER_VERSION=$($ROOT_DIR/.ci/get_test_upgrade_version.sh $JAEGER_VERSION) $GOMPLATE -f ./deployment-assert.yaml.template -o ./03-assert.yaml

fi
