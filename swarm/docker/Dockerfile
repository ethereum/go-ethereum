FROM golang:1.11-alpine as builder

RUN apk add --update git gcc g++ linux-headers
RUN mkdir -p $GOPATH/src/github.com/ethereum && \
    cd $GOPATH/src/github.com/ethereum && \
    git clone --depth 1 https://github.com/ethersphere/go-ethereum && \
    cd $GOPATH/src/github.com/ethereum/go-ethereum && \
    VERSION=$(git rev-parse --short HEAD) go install -ldflags "-X main.gitCommit=${VERSION}" ./cmd/swarm && \
    VERSION=$(git rev-parse --short HEAD) go install -ldflags "-X main.gitCommit=${VERSION}" ./cmd/swarm/swarm-smoke && \
    VERSION=$(git rev-parse --short HEAD) go install -ldflags "-X main.gitCommit=${VERSION}" ./cmd/geth && \
    cp $GOPATH/bin/swarm /swarm && cp $GOPATH/bin/geth /geth && cp $GOPATH/bin/swarm-smoke /swarm-smoke


# Release image with the required binaries and scripts
FROM alpine:3.8
WORKDIR /
COPY --from=builder /swarm /geth /swarm-smoke /
ADD run.sh /run.sh
ENTRYPOINT ["/run.sh"]
