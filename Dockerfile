FROM stagex/pallet-go:2025.02.0@sha256:fb2a63e3ed8756e845ceb44ff8fe57493ba15fc84f75f09487359932e089eea3 AS build
WORKDIR /go-ethereum
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN --network=none <<-EOF
  go run build/ci.go install -static ./cmd/geth
  mkdir -p /rootfs/usr/share/ca-cetificates
  cp -r /usr/share/ca-certificates/* /rootfs/
EOF

FROM scratch
COPY --from=build /rootfs/* /
COPY --from=build /go-ethereum/build/bin/geth /usr/bin/geth
EXPOSE 8545 8546 30303 30303/udp

ENTRYPOINT ["geth"]

ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""
LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"