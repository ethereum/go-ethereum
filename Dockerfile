# Build Geth in a stock Go builder container
FROM golang:1.18-alpine as builder

RUN apk add --no-cache gcc musl-dev linux-headers git

ADD . /go-ethereum
RUN cd /go-ethereum && go run build/ci.go install ./cmd/geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["geth"]

# Add some metadata labels to help programmatic image consumption
ARG BUILDNUM
ARG COMMIT
ARG VERSION

LABEL org.opencontainers.image.build-number="$BUILDNUM"
LABEL org.opencontainers.image.description="Geth is a full node Ethereum implementation written in Go."
LABEL org.opencontainers.image.revision="$COMMIT"
LABEL org.opencontainers.image.source="https://github.com/ethereum/go-ethereum"
LABEL org.opencontainers.image.title="Geth"
LABEL org.opencontainers.image.version="$VERSION"