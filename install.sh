#!/bin/sh

if [ "$1" == "" ]; then
	echo "Usage $0 executable branch"
	echo "executable    ethereum | mist"
	echo "branch        develop | master"
	exit
fi

exe=$1
path=$exe
branch=$2

if [ "$branch" == "develop" ]; then
	path="cmd/$exe"
fi

# Test if go is installed
command -v go >/dev/null 2>&1 || { echo >&2 "Unable to find 'go'. This script requires go."; exit 1; }

# Test if $GOPATH is set
if [ "$GOPATH" == "" ]; then
	echo "\$GOPATH not set"
	exit
fi

echo "changing branch to $branch"
cd $GOPATH/src/github.com/ethereum/go-ethereum
git checkout $branch

# installing package dependencies doesn't work for develop
# branch as go get always pulls from master head
# so build will continue to fail, but this installs locally
# for people who git clone since go install will manage deps

#echo "go get -u -d github.com/ethereum/go-ethereum/$path"
#go get -v -u -d github.com/ethereum/go-ethereum/$path
#if [ $? != 0 ]; then
#	echo "go get failed"
#	exit
#fi

cd $GOPATH/src/github.com/ethereum/go-ethereum/$path

if [ "$exe" == "mist" ]; then
	echo "Building Mist GUI. Assuming Qt is installed. If this step"
	echo "fails; please refer to: https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum(Go)"
else
	echo "Building ethereum CLI."
fi

go install
echo "done. Please run $exe :-)"
