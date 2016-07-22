FROM alpine:3.4

RUN \
  apk add --update go git make gcc musl-dev binutils && \
  git clone https://github.com/ethereum/go-ethereum && \
  (cd go-ethereum && make geth)                     && \
  cp go-ethereum/build/bin/geth /geth               && \
  strip /geth                                       && \
  apk del go git make gcc musl-dev binutils         && \
  rm -rf /go-ethereum && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/geth"]
