FROM alpine:3.3

ADD . /go-ethereum
RUN \
  apk add --update git go make gcc musl-dev         && \
  (cd go-ethereum && make gubiq)                     && \
  cp go-ethereum/build/bin/gubiq /gubiq               && \
  apk del git go make gcc musl-dev                  && \
  rm -rf /go-ethereum && rm -rf /var/cache/apk/*

EXPOSE 8588
EXPOSE 30303

ENTRYPOINT ["/gubiq"]
