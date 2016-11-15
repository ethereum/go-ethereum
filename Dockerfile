FROM alpine:3.3

ADD . /go-ethereum
RUN \
  apk add --update go make gcc musl-dev             && \
  (cd go-ethereum && make geth)                     && \
  cp go-ethereum/build/bin/geth /geth               && \
  apk del go make gcc musl-dev                      && \
  rm -rf /go-ethereum && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/geth"]
