FROM alpine:3.4

RUN \
  apk add --update go git make gcc musl-dev && \
  git clone --depth 1 --branch release/1.5 https://github.com/ubiq/go-ubiq && \
  (cd go-ethereum && make gubiq) && \
  cp go-ethereum/build/bin/gubiq /gubiq && \
  apk del go git make gcc musl-dev && \
  rm -rf /go-ethereum && rm -rf /var/cache/apk/*

EXPOSE 8588
EXPOSE 30303

ENTRYPOINT ["/gubiq"]
