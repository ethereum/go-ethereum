FROM alpine:3.5

ADD . /go-expanse
RUN \
  apk add --update git go make gcc musl-dev linux-headers && \
  (cd go-expanse && make gexp)                           && \
  cp go-expanse/build/bin/gexp /usr/local/bin/           && \
  apk del git go make gcc musl-dev linux-headers          && \
  rm -rf /go-expanse && rm -rf /var/cache/apk/*

EXPOSE 9656
EXPOSE 42786
EXPOSE 42786/udp

ENTRYPOINT ["gexp"]
