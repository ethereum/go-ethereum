FROM ubuntu:14.04.2

## Environment setup
ENV GOPATH /root/go
ENV PATH /root/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games

ENV DEBIAN_FRONTEND noninteractive

## Install base dependencies
RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y git mercurial build-essential software-properties-common wget \
        pkg-config libgmp3-dev libreadline6-dev libpcre3-dev libpcre++-dev --no-install-recommends

## Install Qt5.4.1 (not required for CLI)
# RUN add-apt-repository ppa:beineri/opt-qt541-trusty -y && \
#     apt-get install -y qt54quickcontrols qt54webengine mesa-common-dev libglu1-mesa-dev
# ENV PKG_CONFIG_PATH /opt/qt54/lib/pkgconfig

# Install Go, dump the race detector
RUN wget https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go*.tar.gz                                     && \
    rm -rf /usr/local/go/pkg/linux_amd64_race                             && \
    go version

# Workaround, to make sure that docker's cache is invalidated whenever the git repo changes
ADD https://api.github.com/repos/ethereum/go-ethereum/git/refs/heads/develop file_does_not_exist

## Fetch and install go-ethereum
RUN mkdir -p $GOPATH/src/github.com/ethereum/                                                             && \
    git clone https://github.com/ethereum/go-ethereum $GOPATH/src/github.com/ethereum/go-ethereum         && \
    cd $GOPATH/src/github.com/ethereum/go-ethereum                                                        && \
    git checkout develop                                                                                  && \
    GOPATH=$GOPATH:$GOPATH/src/github.com/ethereum/go-ethereum/Godeps/_workspace go install -v ./cmd/geth

## Run & expose JSON RPC
ENTRYPOINT ["geth", "-rpc=true", "-rpcport=8545"]
EXPOSE 8545
