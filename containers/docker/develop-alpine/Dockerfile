FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 https://github.com/ethereum/go-etherfact && \
  (cd go-etherfact && make getf) && \
  cp go-etherfact/build/bin/getf /getf && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-etherfact && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/getf"]
