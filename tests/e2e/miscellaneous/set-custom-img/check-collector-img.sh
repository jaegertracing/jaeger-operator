#!/bin/bash


function check_image(){
    export IMG=$(kubectl get deployment my-jaeger-collector -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}')
}


check_image
while [ "$IMG" != "test" ]
do
    echo "Collector image missmatch. Expected: test. Has: $IMG"
    sleep 5
    check_image
done

echo "Collector image asserted properly!"