# # Support setting various labels on the final image
# ARG COMMIT=""
# ARG VERSION=""
# ARG BUILDNUM=""
ARG USER=$USER


# Build Geth in a stock Go builder container
FROM golang:1.18-alpine as builder

RUN apk add --no-cache gcc=11.2.1_git20220219-r2 musl-dev=1.2.3-r0 linux-headers=5.16.7-r1 git=2.36.1-r0

COPY . /go-ethereum
WORKDIR /go-ethereum
RUN go run build/ci.go install ./cmd/geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:3.15.0

RUN apk add --no-cache ca-certificates=20211220-r0

COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp

ENTRYPOINT ["geth", "--mainnet", "--port", "0", "--log.debug", "--syncmode", "full", "--datadir", "/root/.ethereum/covalent", "--replication.targets", "redis://localhost:6379/?topic=replication", "--replica.result", "--replica.specimen"]

# Add some metadata labels to help programatic image consumption
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
