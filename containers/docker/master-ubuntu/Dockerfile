FROM ubuntu:wily
MAINTAINER caktux

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && \
    apt-get upgrade -q -y && \
    apt-get dist-upgrade -q -y && \
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 923F6CA9 && \
    echo "deb http://ppa.launchpad.net/ethereum/ethereum/ubuntu wily main" | tee -a /etc/apt/sources.list.d/ethereum.list && \
    apt-get update && \
    apt-get install -q -y geth

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/usr/bin/geth"]
