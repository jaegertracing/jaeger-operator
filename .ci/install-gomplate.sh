#!/bin/bash
VERSION="3.10.0"

if [[ -z "${GOPATH}" ]]; then
    DEST="/usr/local/bin/gomplate"
    export PATH=$PATH:/usr/local/bin
    SUDO="sudo"
else
    DEST="${GOPATH}/bin/gomplate"
    SUDO=
fi


if [ ! -f ${DEST} ]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        $SUDO curl https://github.com/hairyhenderson/gomplate/releases/download/v${VERSION}/gomplate_darwin-amd64-slim -sLo ${DEST}
    else
        $SUDO curl https://github.com/hairyhenderson/gomplate/releases/download/v${VERSION}/gomplate_linux-amd64-slim -sLo ${DEST}
    fi
    $SUDO chmod +x ${DEST}
fi
