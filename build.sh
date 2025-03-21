#!/bin/bash

version=$(grep Version main.go  | head -1 | sed 's/^\(.*\)"\(.*\)".*/\2/')

echo "----------------------------------------------------------------------"
echo " go get -u"
echo "----------------------------------------------------------------------"
go get -u ./...

echo "----------------------------------------------------------------------"
echo " go mod tidy"
echo "----------------------------------------------------------------------"
go mod tidy

mkdir -p bin/

rm bin/pgSimload*
echo "----------------------------------------------------------------------"
echo "Building binaries"
echo "----------------------------------------------------------------------"
echo "Building windows version, stripped executable (in bin/pgSimload_win.exe)"
GOOS=windows GOARCH=amd64 GOAMD64=v2 go build -trimpath -buildmode=pie -ldflags "-s -w -X 'main.Version=${version}'" -o bin/pgSimload_win.exe .
echo "Building mac (darwin) version, stripped executable (in bin/pgSimload_mac)"
GOOS=darwin GOARCH=amd64 GOAMD64=v2 go build -trimpath -buildmode=pie -ldflags "-s -w -X 'main.Version=${version}'" -o bin/pgSimload_mac .
echo "Building linux (amd64) version, stripped executable, not static (in bin/pgSimload)"
GOOS=linux GOARCH=amd64 GOAMD64=v2 go build -trimpath -buildmode=pie -ldflags "-s -w -X 'main.Version=${version}'" -o bin/pgSimload .
echo "Building linux (amd64) version, stripped executable, static (in bin/pgSimload_static)"
GOOS=linux GOARCH=amd64 GOAMD64=v2 CGO_ENABLED=0 go build -trimpath -buildmode=pie -ldflags "-s -w -X 'main.Version=${version}'" -o bin/pgSimload_static .
echo "Building linux (amd64) version, UNSTRIPPED executable, static (in bin/pgSimload_static_non_stripped)"
GOOS=linux GOARCH=amd64 GOAMD64=v2 CGO_ENABLED=0 go build -trimpath -buildmode=pie -ldflags "-s -X 'main.Version=${version}'" -o bin/pgSimload_static_non_stripped .


echo "----------------------------------------------------------------------"
echo "This script can copy the binary to /usr/local/bin if you want"
echo "It assumes user "`whoami`" is sudoer..."
echo "Proceed (y/n)?"
read REPLY 
if [ $REPLY != "y" ]; then
  echo "Exiting.."
  exit 1
fi

echo "Do you want the static linux binary to be installed or the linked one?"
echo "Static version should work anywhere while the linked may not"
echo "Install static (s) or linked one (l) ? CTRL-C to abort"
read REPLY 
if [ $REPLY = "l" ]; then
  sudo cp -v bin/pgSimload /usr/local/bin
  echo "Linked pgSimload binary installed in /usr/local/bin/pgSimload"
else 
  sudo cp -v bin/pgSimload_static /usr/local/bin/pgSimload
  echo "Static pgSimload binary installed in /usr/local/bin/pgSimload"
fi
