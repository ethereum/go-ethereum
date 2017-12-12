FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates  && \

  # clone status-go
  mkdir -p /usr/lib/go/src/github.com/status-im && \
  git clone --depth 1 --branch develop https://github.com/status-im/status-go.git /usr/lib/go/src/github.com/status-im/status-go && \

  # clone go-ethereum (and install everything)
  git clone --depth 1 --branch status/1.6.1-stable https://github.com/status-im/go-ethereum && \
  (cd go-ethereum && build/env.sh go run build/ci.go install ./cmd/wnode) && \
  cp go-ethereum/build/bin/wnode /wnode && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-ethereum && rm -rf /var/cache/apk/*

EXPOSE 30379

ENTRYPOINT ["/wnode"]
