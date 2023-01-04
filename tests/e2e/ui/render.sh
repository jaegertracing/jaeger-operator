#!/bin/bash

source $(dirname "$0")/../render-utils.sh

###############################################################################
# TEST NAME: allinone
###############################################################################
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


###############################################################################
# TEST NAME: production
###############################################################################
start_test "production"
export JAEGER_NAME="production-ui"

if [[ $IS_OPENSHIFT = true && $SKIP_ES_EXTERNAL = true ]]; then
    render_install_jaeger $JAEGER_NAME "production_autoprovisioned" "01"
else
    render_install_elasticsearch "upstream" "00"
    render_install_jaeger $JAEGER_NAME "production" "01"
fi


# Sometimes, the Ingress/OpenShift route is there but not 100% ready so, when
# kubectl tries to get the hostname, it returns an empty string
$GOMPLATE -f $TEMPLATES_DIR/ensure-ingress-host.sh.template -o ./ensure-ingress-host.sh
chmod +x ./ensure-ingress-host.sh

if [ $IS_OPENSHIFT = true ]; then
    # Check the OAuth proxy is enabled
    INSECURE="true" EXPECTED_CODE="403" $GOMPLATE -f $TEMPLATES_DIR/assert-http-code.yaml.template -o ./02-check-forbbiden-access.yaml
fi

# Check we can access the deployment. In OpenShift, a token will be generated
# to access the query endpoint properly
EXPECTED_CODE="200" $GOMPLATE -f $TEMPLATES_DIR/assert-http-code.yaml.template -o ./03-curl.yaml

# After 04-install.yaml, check if the security is disabled properly
INSECURE="true" EXPECTED_CODE="200" $GOMPLATE -f $TEMPLATES_DIR/assert-http-code.yaml.template -o ./05-check-disabled-security.yaml


### Test the tracking.gaID parameter ###
# Check the tracking.gaID was not there
ASSERT_PRESENT="false" TRACKING_ID="MyTrackingId" $GOMPLATE -f $TEMPLATES_DIR/test-ui-config.yaml.template -o ./06-check-NO-gaID.yaml

# Check the tracking.gaID is set properly after 07-install.yaml
ASSERT_PRESENT="true" TRACKING_ID="MyTrackingId" $GOMPLATE -f $TEMPLATES_DIR/test-ui-config.yaml.template -o ./08-check-gaID.yaml
