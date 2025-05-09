# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

FROM stagex/core-ca-certificates@sha256:d6fca6c0080e8e5360cd85fc1c4bd3eab71ce626f40602e38488bfd61fd3e89d AS ca-certificates

# Build Geth using a hermetic, full-source bootstrapped and deterministic toolchain
FROM stagex/pallet-go@sha256:7ffeced176cf8c3035d918618ade8b824f9553ade8b0510df55df9012f35b0a8 AS build

# Get dependencies - will also be cached if we won't change go.mod/go.sum
WORKDIR /go-ethereum
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN --network=none <<-EOF
  go run build/ci.go install -static ./cmd/geth
EOF

# Pull Geth into a clean layer with only required dependencies
FROM scratch 
COPY --from=ca-certificates . /
COPY --from=build /go-ethereum/build/bin/geth /usr/bin/geth
EXPOSE 8545 8546 30303 30303/udp

ENTRYPOINT ["geth"]

# Add some metadata labels to help programmatic image consumption
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""
LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
