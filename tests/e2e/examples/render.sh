#!/bin/bash

source $(dirname "$0")/../render-utils.sh

###############################################################################
# TEST NAME: examples-all-in-one-with-options
###############################################################################
start_test "examples-all-in-one-with-options"
example_name="all-in-one-with-options"
render_install_example "$example_name" "00"
$YQ e -i '.metadata.name="my-jaeger"' ./00-install.yaml
$YQ e -i 'del(.spec.allInOne.image)' ./00-install.yaml
render_smoke_test_example "$example_name" "01"
if [ $IS_OPENSHIFT = true ]; then
    sed -i "s~my-jaeger-query:443~my-jaeger-query:443/jaeger~gi" ./01-smoke-test.yaml
else
    sed -i "s~my-jaeger-query:16686~my-jaeger-query:16686/jaeger~gi" ./01-smoke-test.yaml
fi

###############################################################################
# TEST NAME: examples-collector-with-priority-class
###############################################################################
start_test "examples-collector-with-priority-class"
example_name="collector-with-priority-class"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


###############################################################################
# TEST NAME: examples-service-types
###############################################################################
start_test "examples-service-types"
example_name="service-types"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


###############################################################################
# TEST NAME: examples-simple-prod
###############################################################################
start_test "examples-simple-prod"
example_name="simple-prod"
render_install_example "$example_name" "01"
if [[ $IS_OPENSHIFT = true && $SKIP_ES_EXTERNAL = true ]]; then
    $YQ e -i '.spec.storage.options={}' ./01-install.yaml
    $YQ e -i '.spec.storage.elasticsearch={"nodeCount":1,"resources":{"limits":{"memory":"2Gi"}}}' ./01-install.yaml
else
    render_install_elasticsearch "upstream" "00"
fi
render_smoke_test_example "$example_name" "02"


###############################################################################
# TEST NAME: examples-simple-prod-with-volumes
###############################################################################
start_test "examples-simple-prod-with-volumes"
example_name="simple-prod-with-volumes"
render_install_example "$example_name" "01"
if [[ $IS_OPENSHIFT = true && $SKIP_ES_EXTERNAL = true ]]; then
    $YQ e -i '.spec.storage.options={}' ./01-install.yaml
    $YQ e -i '.spec.storage.elasticsearch={"nodeCount":1,"resources":{"limits":{"memory":"2Gi"}}}' ./01-install.yaml
else
    render_install_elasticsearch "upstream" "00"
fi
render_smoke_test_example "$example_name" "02"
$GOMPLATE -f ./03-check-volume.yaml.template -o 03-check-volume.yaml


###############################################################################
# TEST NAME: examples-simplest
###############################################################################
start_test "examples-simplest"
example_name="simplest"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


###############################################################################
# TEST NAME: examples-with-badger
###############################################################################
start_test "examples-with-badger"
example_name="with-badger"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


###############################################################################
# TEST NAME: examples-with-badger-and-volume
###############################################################################
start_test "examples-with-badger-and-volume"
example_name="with-badger-and-volume"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


###############################################################################
# TEST NAME: examples-with-cassandra
###############################################################################
start_test "examples-with-cassandra"
example_name="with-cassandra"
render_install_cassandra "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


###############################################################################
# TEST NAME: examples-with-sampling
###############################################################################
start_test "examples-with-sampling"
export example_name="with-sampling"
render_install_cassandra "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


###############################################################################
# OpenShift examples ##########################################################
###############################################################################
if [ $IS_OPENSHIFT = true ]; then
    start_test "examples-openshift-with-htpasswd"
    export JAEGER_NAME="with-htpasswd"
    export JAEGER_USERNAME="awesomeuser"
    export JAEGER_PASSWORD="awesomepassword"
    # This variable stores the output from `htpasswd -nbs $JAEGER_USERNAME $JAEGER_PASSWORD`
    # but, to avoid the installation of the `htpasswd` command, we store the generated
    # output here
    export JAEGER_USER_PASSWORD_HASH="awesomeuser:{SHA}uUdqPVUyqNBmERU0Qxj3KFaZnjw="
    # Create the secret
    SECRET=$(echo $JAEGER_USER_PASSWORD_HASH | base64) $GOMPLATE -f ./00-install.yaml.template -o ./00-install.yaml
    # Install the Jaeger instance
    $GOMPLATE -f $EXAMPLES_DIR/openshift/with-htpasswd.yaml -o ./01-install.yaml
    $GOMPLATE -f $TEMPLATES_DIR/allinone-jaeger-assert.yaml.template -o ./01-assert.yaml

    export GET_URL_COMMAND="kubectl get routes -o=jsonpath='{.items[0].status.ingress[0].host}' -n \$NAMESPACE"
    export URL="https://\$($GET_URL_COMMAND)/search"

    # Sometimes, the Ingress/OpenShift route is there but not 100% ready so, when
    # kubectl tries to get the hostname, it returns an empty string
    $GOMPLATE -f $TEMPLATES_DIR/ensure-ingress-host.sh.template -o ./ensure-ingress-host.sh
    chmod +x ./ensure-ingress-host.sh

    INSECURE=true JAEGER_USERNAME= JAEGER_PASSWORD= EXPECTED_CODE="403" $GOMPLATE -f $TEMPLATES_DIR/assert-http-code.yaml.template -o ./02-check-unsecured.yaml
    JAEGER_USERNAME="wronguser" JAEGER_PASSWORD="wrongpassword" EXPECTED_CODE="403" $GOMPLATE -f $TEMPLATES_DIR/assert-http-code.yaml.template -o ./03-check-unauthorized.yaml
    EXPECTED_CODE="200" $GOMPLATE -f $TEMPLATES_DIR/assert-http-code.yaml.template -o ./04-check-authorized.yaml
else
    skip_test "examples-openshift-with-htpasswd" "This test is only supported in OpenShift"
fi
