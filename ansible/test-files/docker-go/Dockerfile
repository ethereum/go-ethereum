# Adjusted from https://github.com/ethereum/go-ethereum/blob/develop/Dockerfile
FROM ubuntu:14.04

## Environment setup
ENV HOME /root
ENV GOPATH /root/go
ENV PATH /go/bin:/root/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games

RUN mkdir -p /root/go
ENV DEBIAN_FRONTEND noninteractive

## Install base dependencies
RUN apt-get update && apt-get upgrade -y
RUN apt-get install -y git mercurial build-essential software-properties-common pkg-config libgmp3-dev libreadline6-dev libpcre3-dev libpcre++-dev

## Build and install Go
RUN hg clone -u release https://code.google.com/p/go
RUN cd go && hg update go1.4
RUN cd go/src && ./make.bash && go version

## Install GUI dependencies
RUN add-apt-repository ppa:ubuntu-sdk-team/ppa -y
RUN apt-get update -y
RUN apt-get install -y qtbase5-private-dev qtdeclarative5-private-dev libqt5opengl5-dev

## Fetch and install serpent-go
RUN go get -v -d github.com/ethereum/serpent-go
WORKDIR $GOPATH/src/github.com/ethereum/serpent-go
# RUN git checkout master
RUN git submodule update --init
RUN go install -v

# Fetch and install go-ethereum
RUN go get -v -d github.com/ethereum/go-ethereum/...
WORKDIR $GOPATH/src/github.com/ethereum/go-ethereum

RUN git checkout develop

RUN git pull
RUN ETH_DEPS=$(go list -f '{{.Imports}} {{.TestImports}} {{.XTestImports}}' github.com/ethereum/go-ethereum/... | sed -e 's/\[//g' | sed -e 's/\]//g' | sed -e 's/C //g'); if [ "$ETH_DEPS" ]; then go get $ETH_DEPS; fi
RUN go install -v ./cmd/ethtest

# Run JSON RPC
ENTRYPOINT ["ethtest"]
EXPOSE 8080

