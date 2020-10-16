#!/bin/sh
if [ $# = 0 ]; then
    usage: vgot cmdpackage[@version]... >&2
    exit 2
fi
d=`mktemp -d`
cd "$d"
echo 'module temp' > go.mod
for i; do
    pkg=`echo $i | sed 's/@.*//'`
    go get "$i" &&
    go install "$pkg" &&
    echo installed `go list -f '{{.ImportPath}}@{{.Module.Version}}' "$pkg"`
done
rm -r "$d"
