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

# Ensure KUTTL is installed
$current_dir/install/install-kuttl.sh
export KUTTL=$root_dir/bin/kubectl-kuttl

mkdir -p $reports_dir

cd $root_dir
make render-e2e-tests-$test_suite_name

if [ "$use_kind_cluster" == true ]; then
	kubectl wait --timeout=5m --for=condition=available deployment ingress-nginx-controller -n ingress-nginx
	kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=5m
	make cert-manager
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

# Don't stop if something fails because we want to process the
# report anyway
set +e

$KUTTL test $KUTTL_OPTIONS --report xml
exit_code=$?

set -e

# The output XML needs some work because it adds "artifacts" as a test case.
# Also, the suites doesn't have a name so, we need to add one.
go install github.com/iblancasa/junitcli/cmd/junitcli@latest
junitcli --suite-name $test_suite_name --report --output $reports_dir/$test_suite_name.xml ./artifacts/kuttl-test.xml

if [ "$KIND_KEEP_CLUSTER" != true ] && [ "$use_kind_cluster" == true ]; then
	cd $root_dir
	make stop-kind
fi

exit $exit_code
