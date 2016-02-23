#!/usr/bin/env sh

set -eux

yes | sudo add-apt-repository ppa:ubuntu-toolchain-r/test
yes | sudo add-apt-repository 'deb http://llvm.org/apt/precise/ llvm-toolchain-precise-3.6 main'
wget -O - http://llvm.org/apt/llvm-snapshot.gpg.key | sudo apt-key add -
sudo apt-get update -qq
sudo apt-get install -qq clang-3.6
