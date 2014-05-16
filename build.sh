#!/bin/sh
# Alternative build script

export GOPATH=`pwd`

for exe in chunkymonkey datatests inspectlevel intercept noise replay style; do
  echo "Building $exe..."
  cd "src/cmd/$exe"
  go build
  cd "$GOPATH"
  mv "src/cmd/$exe/$exe" bin
done
