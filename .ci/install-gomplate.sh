#!/bin/bash

DEST="${GOPATH}/bin/gomplate"
VERSION="3.10.0"

function install_gomplate() {
    echo "Downloading the gomplate"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        curl https://github.com/hairyhenderson/gomplate/releases/download/v${VERSION}/gomplate_darwin-amd64-slim -sLo ${DEST}
    else
        curl https://github.com/hairyhenderson/gomplate/releases/download/v${VERSION}/gomplate_linux-amd64-slim -sLo ${DEST}
    fi
    chmod +x ${DEST}
}

mkdir -p ${GOPATH}/bin

if [ ! -f ${DEST} ]; then
    install_gomplate
fi

${DEST} --version | grep -q ${VERSION}
if [ $? != 0 ]; then
    install_gomplate
fi
