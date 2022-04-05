#!/bin/sh
if [ $# = 0 ]; then
    echo usage: vgot cmdpackage[@version]... >&2
    exit 2
fi
d=`mktemp -d`
cd "$d"
go mod init temp >/dev/null 2>&1
for i; do
    pkg=`echo $i | sed 's/@.*//'`
    go get -d "$i" &&
    go install "$pkg" &&
    echo installed `go list -f '{{.ImportPath}}@{{.Module.Version}}' "$pkg"`
done
rm -r "$d"
