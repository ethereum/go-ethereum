#!/bin/bash

sudo apt-get install software-properties-common
sudo add-apt-repository -y ppa:ethereum/ethereum
sudo add-apt-repository -y ppa:ethereum/ethereum-dev
sudo apt-get update

sudo apt-get install -y build-essential golang git-all

GOPATH=/home/vagrant/go go get github.com/tools/godep

sudo chown -R vagrant:vagrant ~vagrant/go

echo "export GOPATH=/home/vagrant/go" >> ~vagrant/.bashrc
echo "export PATH=\\\$PATH:\\\$GOPATH/bin:/usr/local/go/bin" >> ~vagrant/.bashrc
