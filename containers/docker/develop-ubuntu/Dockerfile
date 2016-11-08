FROM ubuntu:xenial

RUN \
  apt-get update && apt-get install -y golang git make gcc build-essential ca-certificates --no-install-recommends && \
  git clone --depth 1 --branch develop https://github.com/ethereum/go-ethereum && \
  (cd go-ethereum && make geth) && cp go-ethereum/build/bin/geth /geth && \
  apt-get remove -y golang git make gcc build-essential ca-certificates && apt-get clean && \
  apt-get autoremove -y && rm -rf /go-ethereum && rm -rf /var/cache/apt/*

EXPOSE 8545
EXPOSE 8546
EXPOSE 30303

ENTRYPOINT ["/geth"]
