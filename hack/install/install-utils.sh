#!/bin/bash
#
# Some bash functions to install third party software.
#

if [[ "$(basename -- "$0")" = "retry.sh" ]]; then
    echo "Don't run $0, source it" >&2
    exit 1
fi

# Retry the given command until 5 times.
#   retry <command>
#
function retry() {
    if [ "$#" -ne 1 ]; then
        error "Wrong number of parameters used for retry. Usage: retry <command to retry>"
        exit 1
    fi

    command=$1

    n=0
    until [ "$n" -ge 5 ]
    do
        echo "Try $n... $command"
        $command && break
        n=$((n+1))
        sleep 5
    done
}


# Create the bin directory if needd and export its path into the BIN env var.
#   create_bin
#
function create_bin() {
    current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
    bin="$current_dir/../../bin"
    mkdir -p "$bin"
    export BIN=$bin
}


# Check if the tool is there and uses the correct version. The <parameter>
# argument refers to the parameter needed to provide to the tool in order to
# print the version.
#   check_version <program> <version> <parameter>
#
function check_tool() {
    if [ "$#" -ne 3 ]; then
        echo "Wrong number of parameters used for check_tool. Usage: check_version <program> <version> <parameter>"
        exit 1
    fi

    tool=$1
    version=$2
    parameter=$3

    # If the program is there and uses the correct version, do nothing
    if [[ -f "$tool" ]]; then
        if [[ "$($tool $parameter)" =~ .*"$version".* ]]; then
            echo "$(basename -- $tool) $version is installed already"
            exit 0
        fi
    fi
}


# Donwload the given tool if needed. Do nothing if the tool is the correct one.
#   download <tool name> <version> <URL>
#
function download() {
    if [ "$#" -ne 3 ]; then
        error "Wrong number of parameters used for download. Usage: download <tool name> <version> <URL>"
        exit 1
    fi

    program=$1
    version=$2
    url=$3

    create_bin

    tool_path=$BIN/$program

    check_tool "$tool_path" "$version" "--version"

    retry "curl -sLo $tool_path $url"
    chmod +x $tool_path
}
