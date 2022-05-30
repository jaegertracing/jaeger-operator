#!/bin/bash

source $(dirname "$0")/../render-utils.sh


if [ $IS_OPENSHIFT = true ]; then
    skip_test "upgrade" "Test not supported in OpenShift"
else
    start_test "upgrade"

    render_install_jaeger "my-jaeger" "allInOne" "00"

    $GOMPLATE -f ./deployment-assert.yaml.template -o ./01-assert.yaml
    $GOMPLATE -f ./02-check-jaeger-version.yaml.template -o ./02-check-jaeger-version.yaml

    JAEGER_VERSION=$($ROOT_DIR/.ci/get_test_upgrade_version.sh $JAEGER_VERSION) $GOMPLATE -f ./deployment-assert.yaml.template -o ./03-assert.yaml
    sed "s~$IMG~$OPERATOR_IMAGE_NEXT~gi" $ROOT_DIR/tests/_build/manifests/01-jaeger-operator.yaml > ./operator-upgrade.yaml
fi
