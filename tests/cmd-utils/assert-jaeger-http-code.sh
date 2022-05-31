#!/bin/bash

if [ "$#" -ne 5 ]; then
    echo "$0 <url> <expected HTTP code> <is OpenShift?> <namespace> <Jaeger name>"
    exit 1
fi

export URL=$1
export EXPECTED_CODE=$2
export IS_OPENSHIFT=$3
export NAMESPACE=$4
export JAEGER_NAME=$5

export ROOT_DIR=$(realpath $(dirname ${BASH_SOURCE[0]})/../../)
source $ROOT_DIR/hack/common.sh

echo "Checking an expected HTTP response"
n=0

if [ $IS_OPENSHIFT = true ]; then
   echo "Running in OpenShift"

   if [ "$INSECURE" = "true" ]; then
      echo "Not using any secret"
   elif [ ! -z "$JAEGER_USERNAME" ]; then
      echo "Using Jaeger basic authentication"
   else
      echo "User not provided. Getting the token..."
      SECRET=$($ROOT_DIR/tests/cmd-utils/get-token.sh $NAMESPACE $JAEGER_NAME)
   fi
fi


export SLEEP_TIME=10
export MAX_RETRIES=30
export INSECURE_FLAG


until [ "$n" -ge $MAX_RETRIES ]; do
   n=$((n+1))
   echo "Try number $n/$MAX_RETRIES the $URL"

   HTTP_RESPONSE=$(curl \
      ${SECRET:+-H "Authorization: Bearer ${SECRET}"} \
      ${JAEGER_USERNAME:+-u $JAEGER_USERNAME:$JAEGER_PASSWORD} \
      -X GET $URL \
      $INSECURE_FLAG -s \
      -o /dev/null \
      -w %{http_code})
   CMD_EXIT_CODE=$?

   if [ $CMD_EXIT_CODE != 0 ]; then
      echo "Something failed while trying to contact the server. Trying insecure mode"
      INSECURE_FLAG="-k"
      continue
   fi

   if [[ "$HTTP_RESPONSE" = "$EXPECTED_CODE" ]]; then
      echo "curl response asserted properly"
      exit 0
   fi

   echo "HTTP response is $HTTP_RESPONSE. $EXPECTED_CODE expected. Waiting $SLEEP_TIME s"
   sleep $SLEEP_TIME
done

exit 1
