FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 https://github.com/etherinc/go-etherinc && \
  (cd go-etherinc && make geth) && \
  cp go-etherinc/build/bin/geth /geth && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-etherinc && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30103

ENTRYPOINT ["/geth"]
