#!/bin/bash
exit_code=0

while [ "$exit_code" == 0 ]
do
    kubectl get hpa -n $NAMESPACE | grep "<unknown>/90%  <unknown>/90%" -q
    exit_code=$?
    echo "Some HPA metrics are not known yet"
    sleep 1
done