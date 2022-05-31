#!/bin/bash

source $(dirname "$0")/../render-utils.sh


if [ $IS_OPENSHIFT = true ]; then
    skip_test "upgrade" "Test not supported in OpenShift"
else
    start_test "upgrade"

    render_install_jaeger "my-jaeger" "allInOne" "00"

    $GOMPLATE -f ./deployment-assert.yaml.template -o ./01-assert.yaml
    $GOMPLATE -f $TEMPLATES_DIR/check-jaeger-version.yaml.template -o ./02-check-jaeger-version.yaml

    JAEGER_VERSION=$($ROOT_DIR/.ci/get_test_upgrade_version.sh $JAEGER_VERSION) $GOMPLATE -f ./deployment-assert.yaml.template -o ./03-assert.yaml
    sed "s~$IMG~$OPERATOR_IMAGE_NEXT~gi" $ROOT_DIR/tests/_build/manifests/01-jaeger-operator.yaml > ./operator-upgrade.yaml
fi

if [ $IS_OPENSHIFT = true ]; then
    skip_test "upgrade-from-latest" "Test not supported in OpenShift"
else
    start_test "upgrade-from-latest"
    $GOMPLATE -f ./remove-jaeger-operator.yaml.template -o ./00-remove-operator.yaml

    # Download the latest Jaeger Operator released manifest and install it
    LATEST_VERSION=$(git show main:versions.txt | grep operator | awk -F= '{print $2}')
    wget -q https://github.com/jaegertracing/jaeger-operator/releases/download/v$LATEST_VERSION/jaeger-operator.yaml

    # Deploy Jaeger in production mode
    jaeger_name="jaeger-test"
    render_install_elasticsearch "02"
    render_install_jaeger "$jaeger_name" "production" "03"



    # Run smoke test
    render_smoke_test "$jaeger_name" "production" "04"


    $GOMPLATE -f ./05-install-current-operator.yaml.template -o ./05-install-current-operator.yaml

    # Run smoke test
    render_smoke_test "$jaeger_name" "production" "06"
fi
