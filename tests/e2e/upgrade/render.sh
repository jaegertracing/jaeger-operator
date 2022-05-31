#!/bin/bash

source $(dirname "$0")/../render-utils.sh

export JAEGER_NAME

if [ $IS_OPENSHIFT = true ]; then
    skip_test "upgrade" "Test not supported in OpenShift"
else
    start_test "upgrade"
    export JAEGER_NAME="my-jaeger"

    render_install_jaeger "$JAEGER_NAME" "allInOne" "00"

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

    if version_gt $LATEST_VERSION "1.34.1"; then
        # Container Jaeger Operator images up to 1.34.1 had a bug where
        # `jaeger-operator version` don't print the version number
        EXPECTED_VERSION=$LATEST_VERSION $GOMPLATE -f $TEMPLATES_DIR/check-jaeger-operator-version.yaml -o ./02-check-operator-version.yaml
    fi

    # Deploy Jaeger in production mode
    jaeger_name="jaeger-test"
    render_install_elasticsearch "03"
    render_install_jaeger "$jaeger_name" "production" "04"

    # Run smoke test
    render_smoke_test "$jaeger_name" "production" "05"

    # Install the current Jaeger Operator
    $GOMPLATE -f ./install-current-operator.yaml.template -o ./06-install-current-operator.yaml

    # Check the Jaeger Operator version is upgraded
    EXPECTED_VERSION=$JAEGER_OPERATOR_VERSION $GOMPLATE -f $TEMPLATES_DIR/check-jaeger-operator-version.yaml -o ./07-check-updated-operator-version.yaml
    # Check the Jaeger version is the expected
    JAEGER_NAME=$jaeger_name $GOMPLATE -f $TEMPLATES_DIR/check-jaeger-version.yaml.template -o ./08-check-jaeger-version.yaml

    # Run smoke test
    render_smoke_test "$jaeger_name" "production" "08"
fi
