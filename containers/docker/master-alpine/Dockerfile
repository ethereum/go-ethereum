FROM alpine:3.3

RUN \
  apk add --update go git make gcc musl-dev gmp-dev gmp && \
  git clone https://github.com/ethereum/go-ethereum     && \
  (cd go-ethereum && make geth)                         && \
  cp go-ethereum/build/bin/geth /geth                   && \
  apk del go git make gcc musl-dev gmp-dev              && \
  rm -rf /go-ethereum && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/geth"]
