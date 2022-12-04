#!/bin/bash
set -x

PROJECT_PATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"

git remote add upstream https://github.com/ethereum/go-ethereum.git

set -ex
git fetch upstream master:
git rebase FETCH_HEAD
