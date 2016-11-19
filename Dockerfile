FROM alpine:3.3

ADD . /go-ubiq
RUN \
  apk add --update git go make gcc musl-dev         && \
  (cd go-ubiq && make gubiq)                     && \
  cp go-ubiq/build/bin/gubiq /gubiq               && \
  apk del git go make gcc musl-dev                  && \
  rm -rf /go-ubiq && rm -rf /var/cache/apk/*

EXPOSE 8588
EXPOSE 30388

ENTRYPOINT ["/gubiq"]
