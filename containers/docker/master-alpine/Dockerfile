FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 --branch release/1.7 https://github.com/expanse-org/go-expanse && \
  (cd go-expanse && make gexp) && \
  cp go-expanse/build/bin/gexp /gexp && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-expanse && rm -rf /var/cache/apk/*

EXPOSE 9656
EXPOSE 42786

ENTRYPOINT ["/gexp"]
