#!/bin/bash

source $(dirname "$0")/../render-utils.sh


start_test "ui-definition"
export QUERY_BASE_PATH="jaeger"
export TRACKING_ID="MyTrackingId"
export GET_URL_COMMAND

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
