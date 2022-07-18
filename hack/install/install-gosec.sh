#!/bin/bash
VERSION="2.12.0 "

echo "Installing gosec"

current_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $current_dir/install-utils.sh

create_bin

PROGRAM="gosec"
check_tool "$BIN/gosec" $VERSION "version"


# Download the installer
retry "curl -sLo $BIN/install-gosec.sh https://raw.githubusercontent.com/securego/gosec/master/install.sh"
chmod +x $BIN/install-gosec.sh

# Run the installer
retry "$BIN/install-gosec.sh v${VERSION}"


export PATH=$PATH:$BIN
