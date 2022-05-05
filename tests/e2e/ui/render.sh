#!/bin/bash

source $(dirname "$0")/../render-utils.sh


start_test "allinone"
export GET_URL_COMMAND
export URL
export JAEGER_NAME="all-in-one-ui"

# The URL is decided when the tests starts. So, the YAML file for the job is
# rendered after the test started
if [ $IS_OPENSHIFT = true ]; then
    GET_URL_COMMAND="kubectl get routes -o=jsonpath='{.items[0].status.ingress[0].host}' -n \$NAMESPACE"
    URL="https://\$($GET_URL_COMMAND)/search"
else
    GET_URL_COMMAND="echo http://localhost"
    URL="http://localhost/search"
fi

# Sometimes, the Ingress/OpenShift route is there but not 100% ready so, when
# kubectl tries to get the hostname, it returns an empty string
$GOMPLATE -f $TEMPLATES_DIR/ensure-ingress-host.sh.template -o ./ensure-ingress-host.sh
chmod +x ./ensure-ingress-host.sh

# Check we can access the deployment
EXPECTED_CODE="200" $GOMPLATE -f $TEMPLATES_DIR/assert-http-code.yaml.template -o ./01-curl.yaml

### Test the tracking.gaID parameter ###
# Check the tracking.gaID is set properly
ASSERT_PRESENT="true" TRACKING_ID="MyTrackingId" $GOMPLATE -f $TEMPLATES_DIR/test-ui-config.yaml.template -o ./04-test-ui-config.yaml


start_test "production"
export JAEGER_NAME="production-ui"

if [ $SKIP_ES_EXTERNAL = false ]; then
    render_install_elasticsearch "00"
fi

render_install_jaeger $JAEGER_NAME "production" "01"

# Sometimes, the Ingress/OpenShift route is there but not 100% ready so, when
# kubectl tries to get the hostname, it returns an empty string
$GOMPLATE -f $TEMPLATES_DIR/ensure-ingress-host.sh.template -o ./ensure-ingress-host.sh
chmod +x ./ensure-ingress-host.sh

# Check we can access the deployment
EXPECTED_CODE="200" $GOMPLATE -f $TEMPLATES_DIR/assert-http-code.yaml.template -o ./03-curl.yaml

### Test the tracking.gaID parameter ###
# Check the tracking.gaID was not there
ASSERT_PRESENT="false" TRACKING_ID="MyTrackingId" $GOMPLATE -f $TEMPLATES_DIR/test-ui-config.yaml.template -o ./04-check-NO-gaID.yaml

# Check the tracking.gaID is set properly after 05-install.yaml
ASSERT_PRESENT="true" TRACKING_ID="MyTrackingId" $GOMPLATE -f $TEMPLATES_DIR/test-ui-config.yaml.template -o ./06-check-gaID.yaml

# When the tracking.gaID is modified in a Kubernetes cluster, the value is not
# mofidied in the HTML code. In OpenShift, the change is performed properly
if [ $IS_OPENSHIFT = true ]; then
    # Check the tracking.gaID was changed properly after 07-install.yaml
    ASSERT_PRESENT="false" TRACKING_ID="MyTrackingId" $GOMPLATE -f $TEMPLATES_DIR/test-ui-config.yaml.template -o ./08-check-changed-gaID.yaml
    ASSERT_PRESENT="true" TRACKING_ID="aNewTrackingID" $GOMPLATE -f $TEMPLATES_DIR/test-ui-config.yaml.template -o ./09-check-new-gaIDla.yaml
fi
