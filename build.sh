#!/bin/sh

# would be nice to update this automagically
versionNumber=0.0.1
commitId=$(git rev-parse HEAD)

build () {
	echo "build $1/$2"
	env GOOS=$1 GOARCH=$2 go build -ldflags "-X main.versionNumber=$versionNumber -X main.commitId=$commitId" -o dist/udmp-iptables-log-rules_$2 main.go	
}


[ -d "dist" ] && rm -r dist
mkdir "dist"

build linux amd64
build linux arm64