#!/bin/bash

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# Enable verbosity
if [ "$VERBOSE" = true ]; then
    set -o xtrace
fi

test_suites=$@

# Don't stop if something fails
set -e

rm -rf logs reports

mkdir -p logs
mkdir -p reports

failed=false

for test_suite in $test_suites; do
    echo "============================================================"
    echo "Running test suite $test_suite"
    echo "============================================================"
    make run-e2e-tests-$test_suite 2>&1 | tee -a ./logs/$test_suite.txt
    exit_code=${PIPESTATUS[0]}

    if [ ! -e ./reports/$test_suite.xml ]; then
        echo "Test $test_suite failed with code $exit_code and the report was not generated" >> ./logs/failures.txt
        failed=true
    fi

    if [ "$exit_code" -ne 0 ]; then
        echo "Test $test_suite failed with code $exit_code" >> ./logs/failures.txt
        failed=true
    fi
done

go install github.com/iblancasa/junitcli/cmd/junitcli@v1.0.1
junitcli --report reports

if [ failed = true ]; then
    echo "Something failed while running the E2E tests!!!"
    exit 1
fi
