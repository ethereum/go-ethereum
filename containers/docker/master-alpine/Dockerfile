FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 --branch release/1.6 https://github.com/ubiq/go-ubiq && \
  (cd go-ubiq && make gubiq) && \
  cp go-ubiq/build/bin/gubiq /gubiq && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-ubiq && rm -rf /var/cache/apk/*

EXPOSE 8588
EXPOSE 30388

ENTRYPOINT ["/gubiq"]
