# Use Alpine Linux
FROM golang:1.9-alpine as builder 

ADD . /go-ubiq
RUN \
  apk add --update git go make gcc musl-dev linux-headers && \
  (cd go-ubiq && make gubiq)                           && \
  cp go-ubiq/build/bin/gubiq /usr/local/bin/           && \
  apk del git go make gcc musl-dev linux-headers          && \
  rm -rf /go-ubiq && rm -rf /var/cache/apk/*

EXPOSE 8588
EXPOSE 30388
EXPOSE 30388/udp

ENTRYPOINT ["gubiq"]
