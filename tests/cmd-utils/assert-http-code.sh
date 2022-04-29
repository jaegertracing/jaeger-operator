#!/bin/bash

URL=$1
EXPECTED_CODE=$2

echo "Checking an expected HTTP response"
n=0

until [ "$n" -ge 30 ]
do
   echo "Try number $n"

   HTTP_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $URL)
   echo "HTTP response is $HTTP_RESPONSE"

   if [[ $HTTP_RESPONSE = $EXPECTED_CODE ]]
   then
      echo "curl response asserted properly"
      exit 0
   fi

   n=$((n+1))
   sleep 10
done

exit 1
