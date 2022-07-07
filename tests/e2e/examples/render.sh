#!/bin/bash

source $(dirname "$0")/../render-utils.sh

start_test "examples-agent-as-daemonset"
example_name="agent-as-daemonset"

prepare_daemonset "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


start_test "examples-agent-with-priority-class"
example_name="agent-with-priority-class"
prepare_daemonset "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


start_test "examples-business-application-injected-sidecar"
example_name="simplest"
cp $EXAMPLES_DIR/business-application-injected-sidecar.yaml ./00-install.yaml
$YQ e -i '.spec.template.spec.containers[0].livenessProbe.httpGet.path="/"' ./00-install.yaml
$YQ e -i '.spec.template.spec.containers[0].livenessProbe.httpGet.port=8080' ./00-install.yaml
$YQ e -i '.spec.template.spec.containers[0].livenessProbe.initialDelaySeconds=1' ./00-install.yaml
$YQ e -i '.spec.template.spec.containers[0].livenessProbe.failureThreshold=3' ./00-install.yaml
$YQ e -i '.spec.template.spec.containers[0].livenessProbe.periodSeconds=10' ./00-install.yaml
$YQ e -i '.spec.template.spec.containers[0].livenessProbe.successThreshold=1' ./00-install.yaml
$YQ e -i '.spec.template.spec.containers[0].livenessProbe.timeoutSeconds=1' ./00-install.yaml
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


start_test "examples-service-types"
example_name="service-types"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


start_test "examples-simple-prod"
example_name="simple-prod"
render_install_elasticsearch "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


start_test "examples-simple-prod-with-volumes"
example_name="simple-prod-with-volumes"
render_install_elasticsearch "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


start_test "examples-simplest"
example_name="simplest"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


start_test "examples-with-badger"
example_name="with-badger"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


start_test "examples-with-badger-and-volume"
example_name="with-badger-and-volume"
render_install_example "$example_name" "00"
render_smoke_test_example "$example_name" "01"


start_test "examples-with-cassandra"
example_name="with-cassandra"
render_install_cassandra "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"


start_test "examples-with-sampling"
export example_name="with-sampling"
render_install_cassandra "00"
render_install_example "$example_name" "01"
render_smoke_test_example "$example_name" "02"

###############################################################################
# OpenShift examples ##########################################################
###############################################################################
if [ $IS_OPENSHIFT = true ]; then
    start_test "examples-openshift-agent-as-daemonset"
    $GOMPLATE -f $EXAMPLES_DIR/openshift/hostport-scc-daemonset.yaml -o 00-install.yaml
    $GOMPLATE -f $EXAMPLES_DIR/openshift/service_account_jaeger-agent-daemonset.yaml -o 01-install.yaml
    $GOMPLATE -f $EXAMPLES_DIR/openshift/agent-as-daemonset.yaml -o 03-install.yaml
    render_install_vertx "04"
    $YQ e -i '.spec.template.spec.containers[0].env=[{"name": "JAEGER_AGENT_HOST", "valueFrom": {"fieldRef": {"apiVersion": "v1", "fieldPath": "status.hostIP"}}}]' ./04-install.yaml
    render_find_service "agent-as-daemonset" "production" "order" "00" "05"
else
    skip_test "examples-openshift-agent-as-daemonset" "This test is only supported in OpenShift"
fi


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
