#!/bin/bash
#  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
#  This source code is licensed under the BSD-style license found in the
#  LICENSE file in the root directory of this source tree. An additional grant
#  of patent rights can be found in the PATENTS file in the same directory.

set -e
if [ -z "$GIT" ]
then
  GIT="git"
fi

# Print out the colored progress info so that it can be brainlessly 
# distinguished by users.
function title() {
  echo -e "\033[1;32m$*\033[0m"
}

usage="Create new RocksDB version and prepare it for the release process\n"
usage+="USAGE: ./make_new_version.sh <version>"

# -- Pre-check
if [[ $# < 1 ]]; then
  echo -e $usage
  exit 1
fi

ROCKSDB_VERSION=$1

GIT_BRANCH=`git rev-parse --abbrev-ref HEAD`
echo $GIT_BRANCH

if [ $GIT_BRANCH != "master" ]; then
  echo "Error: Current branch is '$GIT_BRANCH', Please switch to master branch."
  exit 1
fi

title "Adding new tag for this release ..."
BRANCH="$ROCKSDB_VERSION.fb"
$GIT checkout -b $BRANCH

# Setting up the proxy for remote repo access
title "Pushing new branch to remote repo ..."
git push origin --set-upstream $BRANCH

title "Branch $BRANCH is pushed to github;"
