#!/bin/bash

source $(dirname "$0")/../render-utils.sh

export JAEGER_NAME

if [ $IS_OPENSHIFT = true ]; then
    skip_test "upgrade" "Test not supported in OpenShift"
else
    start_test "upgrade"
    export JAEGER_NAME="my-jaeger"

    render_install_jaeger "$JAEGER_NAME" "allInOne" "00"

    # Check the deployment using the correct image
    $GOMPLATE -f ./deployment-assert.yaml.template -o ./01-assert.yaml
    $GOMPLATE -f $TEMPLATES_DIR/check-jaeger-version.yaml.template -o ./02-check-jaeger-version.yaml

    $GOMPLATE -f ./replace.yaml.template -o ./03-upgrade-operator.yaml
    JAEGER_VERSION=$($ROOT_DIR/.ci/get_test_upgrade_version.sh $JAEGER_VERSION) $GOMPLATE -f ./deployment-assert.yaml.template -o ./03-assert.yaml

fi

if [ $IS_OPENSHIFT = true ]; then
    skip_test "upgrade-from-latest-release" "Test not supported in OpenShift"
else
    start_test "upgrade-from-latest-release"
    $GOMPLATE -f ./remove-jaeger-operator.yaml.template -o ./00-remove-operator.yaml

    # Download the latest Jaeger Operator released manifest and install it
    LATEST_VERSION=$(curl --max-time 5 --retry 5 -H "Accept: application/vnd.github.v3+json" https://api.github.com/repos/jaegertracing/jaeger-operator/releases/latest | $YQ -P ".tag_name" | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+')
    wget -q https://github.com/jaegertracing/jaeger-operator/releases/download/v$LATEST_VERSION/jaeger-operator.yaml

    if version_gt $LATEST_VERSION "1.35.0"; then
        # Container Jaeger Operator images up to 1.35.0 had a bug where
        # `jaeger-operator version` don't print the version number
        EXPECTED_VERSION=$LATEST_VERSION $GOMPLATE -f $TEMPLATES_DIR/check-jaeger-operator-version.yaml -o ./02-check-operator-version.yaml
    fi

    # Deploy Jaeger in production mode
    jaeger_name="jaeger-test"
    render_install_elasticsearch "upstream" "03"
    render_install_jaeger "$jaeger_name" "production" "04"

    # Run smoke test
    render_smoke_test "$jaeger_name" "false" "05"

    # Install the current Jaeger Operator
    $GOMPLATE -f ./install-current-operator.yaml.template -o ./06-install-current-operator.yaml

    # Check the Jaeger Operator version is upgraded
    EXPECTED_VERSION=$JAEGER_OPERATOR_VERSION $GOMPLATE -f $TEMPLATES_DIR/check-jaeger-operator-version.yaml -o ./07-check-updated-operator-version.yaml
    # Check the Jaeger version is the expected
    JAEGER_NAME=$jaeger_name $GOMPLATE -f $TEMPLATES_DIR/check-jaeger-version.yaml.template -o ./08-check-jaeger-version.yaml

    # Run smoke test
    render_smoke_test "$jaeger_name" "false" "08"
fi
