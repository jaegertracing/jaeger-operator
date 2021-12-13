#!/bin/bash
#
# Utils for the render.sh scripts.
#
if [[ "$(basename -- "$0")" == "render-utils.sh" ]]; then
    echo "Don't run $0, source it" >&2
    exit 1
fi

# Enable verbosity
if [ "$VERBOSE" = true ]; then
    set -o xtrace
fi

# Check the dependencies are there
export GOMPLATE=$(which gomplate)
if [ -z "$GOMPLATE" ]; then
    "gomplate is not installed. Please, install it"
    exit 1
fi

export ROOT_DIR=../../../..
export TEST_DIR=../../..
export TEMPLATES_DIR=$TEST_DIR/templates
export EXAMPLES_DIR=$ROOT_DIR/examples
export SUITE_DIR=$(dirname "$0")
