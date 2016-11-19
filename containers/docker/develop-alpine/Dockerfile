FROM alpine:3.4

RUN \
  apk add --update go git make gcc musl-dev && \
  git clone --depth 1 https://github.com/ubiq/go-ubiq && \
  (cd go-ubiq && make gubiq) && \
  cp go-ubiq/build/bin/gubiq /gubiq && \
  apk del go git make gcc musl-dev && \
  rm -rf /go-ubiq && rm -rf /var/cache/apk/*

EXPOSE 8588
EXPOSE 30388

ENTRYPOINT ["/gubiq"]
