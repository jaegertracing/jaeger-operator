#!/bin/bash

source $(dirname "$0")/../render-utils.sh

cd $SUITE_DIR

export JAEGER_SERVICE=my-test-service
export JAEGER_OPERATION=my-little-op
export JAEGER_NAME=my-jaeger

echo "Rendering templates for allinone-ingress test"
cd allinone-ingress
$GOMPLATE -f $TEMPLATES_DIR/smoke-test.yaml.template -o ./01-report-span.yaml
$GOMPLATE -f $TEMPLATES_DIR/smoke-test-assert.yaml.template -o ./01-assert.yaml

export DESTINATION_NAME=$JAEGER_NAME-query

# This variable is used to generate 03-check-route.yaml
export GET_URL_COMMAND
export ROUTE_NAME=$JAEGER_NAME
export QUERY_HOST

ROUTE_NAME=$JAEGER_NAME-query
$GOMPLATE -f $TEMPLATES_DIR/assert-ingress.yaml.template -o ./02-assert.yaml
GET_URL_COMMAND="kubectl get ingress $JAEGER_NAME-query -o=jsonpath='{.status.loadBalancer.ingress[0].hostname}' -n \$NAMESPACE"
QUERY_HOST="http://\$($GET_URL_COMMAND)"

$GOMPLATE -f $TEMPLATES_DIR/ensure-ingress-host.sh.template -o ./ensure-ingress-host.sh
chmod +x ./ensure-ingress-host.sh
$GOMPLATE -f $TEMPLATES_DIR/find-service-from-client.yaml.template -o ./03-check-route.yaml

cd ..

echo "Rendering templates for allinone-uidefinition test"
cd allinone-uidefinition
export QUERY_HOSTNAME=all-in-one-with-ui-config-query
export QUERY_BASE_PATH=jaeger
export TRACKING_ID=MyTrackingId
$GOMPLATE -f $TEMPLATES_DIR/test-uiconfig.yaml.template -o ./01-install.yaml
