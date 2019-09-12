#!/bin/bash

DEST="${GOPATH}/bin/operator-sdk"

function install_sdk() {
    echo "Downloading the operator-sdk ${SDK_VERSION} into ${DEST}"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        curl https://github.com/operator-framework/operator-sdk/releases/download/${SDK_VERSION}/operator-sdk-${SDK_VERSION}-x86_64-apple-darwin -sLo ${DEST}
    else
        curl https://github.com/operator-framework/operator-sdk/releases/download/${SDK_VERSION}/operator-sdk-${SDK_VERSION}-x86_64-linux-gnu -sLo ${DEST}
    fi
    chmod +x ${DEST}
}

mkdir -p ${GOPATH}/bin

if [ ! -f ${DEST} ]; then
    install_sdk
fi

${DEST} version | grep -q ${SDK_VERSION}
if [ $? != 0 ]; then
    install_sdk
fi
