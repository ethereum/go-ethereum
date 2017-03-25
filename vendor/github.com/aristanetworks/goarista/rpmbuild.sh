#!/bin/sh

# Copyright (C) 2016  Arista Networks, Inc.
# Use of this source code is governed by the Apache License 2.0
# that can be found in the COPYING file.

if [ "$#" -lt 1 ]
then
   echo "usage: $0 <binary>"
   exit 1
fi
binary=$1

if [ -z "$GOPATH" ] || [ -z "$GOOS" ] || [ -z "$GOARCH" ]
then
    echo "Please set \$GOPATH, \$GOOS and \$GOARCH"
    exit 1
fi

set -e

version=$(git rev-parse --short=7 HEAD)
pwd=$(pwd)
cd $GOPATH/bin
if [ -d $GOOS_$GOARCH ]
then
   cd $GOOS_GOARCH
fi
os=$GOOS
arch=$GOARCH
if [ "$arch" == "386" ]
then
   arch="i686"
fi
cmd="fpm -n $binary -v $version -s dir -t rpm --rpm-os $os -a $arch --epoch 0 --prefix /usr/bin $binary"
echo $cmd
$cmd
mv $binary-$version-1.$arch.rpm $pwd
