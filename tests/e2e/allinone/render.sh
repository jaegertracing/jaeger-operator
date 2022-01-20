#!/bin/bash

source $(dirname "$0")/../render-utils.sh

jaeger_name="my-jaeger"

start_test "allinone-ingress"
render_smoke_test "$jaeger_name" "allInOne" "01"

# This variable is used to generate 03-check-route.yaml
export GET_URL_COMMAND
export ROUTE_NAME=$jaeger_name
export QUERY_HOST
export DESTINATION_NAME=$jaeger_name-query

if [ $IS_OPENSHIFT = true ]; then
    $GOMPLATE -f $TEMPLATES_DIR/openshift/assert-route.yaml.template -o ./02-assert.yaml
    GET_URL_COMMAND="kubectl get routes -o=jsonpath='{.items[0].status.ingress[0].host}' -n \$NAMESPACE"
    QUERY_HOST="https://\$($GET_URL_COMMAND)"
else
    ROUTE_NAME=$jaeger_name-query
    $GOMPLATE -f $TEMPLATES_DIR/assert-ingress.yaml.template -o ./02-assert.yaml
    GET_URL_COMMAND="kubectl get ingress $ROUTE_NAME -o=jsonpath='{.status.loadBalancer.ingress[0].hostname}' -n \$NAMESPACE"
    QUERY_HOST="http://\$($GET_URL_COMMAND)"
fi

$GOMPLATE -f $TEMPLATES_DIR/ensure-ingress-host.sh.template -o ./ensure-ingress-host.sh
chmod +x ./ensure-ingress-host.sh
export JAEGER_SERVICE_NAME="smoke-test-service"
$GOMPLATE -f $TEMPLATES_DIR/find-service-from-client.yaml.template -o ./03-check-route.yaml


start_test "allinone-uidefinition"
export QUERY_BASE_PATH=jaeger
export TRACKING_ID=MyTrackingId

# The URL is decided when the tests starts. So, the YAML file for the job is rendered after the test started
if [ $IS_OPENSHIFT = true ]; then
    GET_URL_COMMAND="kubectl get routes -o=jsonpath='{.items[0].status.ingress[0].host}' -n \$NAMESPACE"
else
    GET_URL_COMMAND="echo http://all-in-one-with-ui-config-query:16686"
fi

$GOMPLATE -f $TEMPLATES_DIR/ensure-ingress-host.sh.template -o ./ensure-ingress-host.sh
chmod +x ./ensure-ingress-host.sh

if [ $IS_OPENSHIFT = true ]; then
    GET_URL_COMMAND="https://\$($GET_URL_COMMAND)"
else
    GET_URL_COMMAND="http://all-in-one-with-ui-config-query:16686"
fi

$GOMPLATE -f $TEMPLATES_DIR/render-test-ui.yaml.template -o ./01-install.yaml
