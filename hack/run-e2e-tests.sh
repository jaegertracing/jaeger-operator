#!/bin/bash

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
set -e

# Enable verbosity
if [ "$VERBOSE" = true ]; then
    set -o xtrace
fi

if [ "$#" -ne 3 ]; then
    echo "$0 <test_suite_name> <use_kind_cluster> <olm>"
    exit 1
fi

test_suite_name=$1
use_kind_cluster=$2
olm=$3

root_dir=$current_dir/../
reports_dir=$root_dir/reports

mkdir -p $reports_dir

cd $root_dir
make render-e2e-tests-$test_suite_name

if [ "$use_kind_cluster" = true ]; then
	kubectl wait --timeout=5m --for=condition=available deployment ingress-nginx-controller -n ingress-nginx
	kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=5m
fi

if [ "$olm" = true ]; then
    echo "Skipping Jaeger Operator installation because OLM=true"
else
	echo Installing Jaeger Operator...
	kubectl create namespace observability 2>&1 | grep -v "already exists" || true
	kubectl apply -f ./tests/_build/manifests/01-jaeger-operator.yaml -n observability
	kubectl wait --timeout=5m --for=condition=available deployment jaeger-operator -n observability
fi

echo Running $test_suite_name E2E tests
cd tests/e2e/$test_suite_name/_build

kubectl kuttl test $(KUTTL_OPTIONS) --report xml || exit_code=$?

yq -p=xml e 'del(.testsuites.testsuite.testcase[] | select(.+name == "artifacts"))' ./artifacts/kuttl-test.xml -o xml > $reports_dir/$test_suite_name.xml

if [ "$exit_code" != 0 ]; then
	exit $exit_code
fi

if [ "$use_kind_cluster" = true]; then
	cd $root_dir
	make stop-kind
fi
