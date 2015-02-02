FROM ubuntu:14.04

## Environment setup
ENV HOME /root
ENV GOPATH /root/go
ENV PATH /golang/bin:/root/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games
ENV PKG_CONFIG_PATH /opt/qt54/lib/pkgconfig

RUN mkdir -p /root/go
ENV DEBIAN_FRONTEND noninteractive

## Install base dependencies
RUN apt-get update && apt-get upgrade -y
RUN apt-get install -y git mercurial build-essential software-properties-common pkg-config libgmp3-dev libreadline6-dev libpcre3-dev libpcre++-dev mesa-common-dev libglu1-mesa-dev

## Install Qt5.4 dependencies from PPA
RUN add-apt-repository ppa:beineri/opt-qt54-trusty -y
RUN apt-get update -y
RUN apt-get install -y qt54quickcontrols qt54webengine 

## Build and install latest Go
RUN git clone https://go.googlesource.com/go golang
RUN cd golang && git checkout go1.4.1
RUN cd golang/src && ./all.bash && go version

## Fetch and install QML
RUN go get -u -v -d github.com/obscuren/qml
WORKDIR $GOPATH/src/github.com/obscuren/qml
RUN git checkout v1
RUN go install -v

## Fetch and install go-ethereum
RUN go get -u -v -d github.com/ethereum/go-ethereum/...
WORKDIR $GOPATH/src/github.com/ethereum/go-ethereum
RUN ETH_DEPS=$(go list -f '{{.Imports}} {{.TestImports}} {{.XTestImports}}' github.com/ethereum/go-ethereum/... | sed -e 's/\[//g' | sed -e 's/\]//g' | sed -e 's/C //g'); if [ "$ETH_DEPS" ]; then go get $ETH_DEPS; fi
RUN go install -v ./cmd/ethereum

## Run & expose JSON RPC
ENTRYPOINT ["ethereum", "-rpc=true", "-rpcport=8080"]
EXPOSE 8080
