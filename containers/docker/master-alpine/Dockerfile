FROM alpine:3.3

RUN \
  apk add --update go git make gcc musl-dev         && \
  git clone https://github.com/expanse-org/go-expanse && \
  (cd go-expanse && make geth)                     && \
  cp go-expanse/build/bin/gexp /gexp               && \
  apk del go git make gcc musl-dev                  && \
  rm -rf /go-expanse && rm -rf /var/cache/apk/*

EXPOSE 9656
EXPOSE 42786

ENTRYPOINT ["/gexp"]
