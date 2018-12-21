FROM golang:1.11-alpine as builder

ARG VERSION

RUN apk add --update git gcc g++ linux-headers
RUN mkdir -p $GOPATH/src/github.com/ethereum && \
    cd $GOPATH/src/github.com/ethereum && \
    git clone https://github.com/ethersphere/go-ethereum && \
    cd $GOPATH/src/github.com/ethereum/go-ethereum && \
    git checkout ${VERSION} && \
    go install -ldflags "-X main.gitCommit=${VERSION}" ./cmd/swarm && \
    go install -ldflags "-X main.gitCommit=${VERSION}" ./cmd/swarm/swarm-smoke && \
    go install -ldflags "-X main.gitCommit=${VERSION}" ./cmd/geth && \
    cp $GOPATH/bin/swarm /swarm && cp $GOPATH/bin/geth /geth && cp $GOPATH/bin/swarm-smoke /swarm-smoke


# Release image with the required binaries and scripts
FROM alpine:3.8
WORKDIR /
COPY --from=builder /swarm /geth /swarm-smoke /
ADD run.sh /run.sh
ADD run-smoke.sh /run-smoke.sh
ENTRYPOINT ["/run.sh"]
