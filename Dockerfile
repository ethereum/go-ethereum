# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""
ARG SCROLL_LIB_PATH=/scroll/lib

# Build libzkp dependency
FROM scrolltech/go-rust-builder:go-1.21-rust-nightly-2023-12-03 as chef
WORKDIR app

FROM chef as planner
COPY ./rollup/ccc/libzkp/ .
RUN cargo chef prepare --recipe-path recipe.json

FROM chef as zkp-builder
COPY ./rollup/ccc/libzkp/rust-toolchain ./
COPY --from=planner /app/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json

COPY ./rollup/ccc/libzkp .
RUN cargo clean
RUN cargo build --release

# Build Geth in a stock Go builder container
FROM scrolltech/go-rust-builder:go-1.21-rust-nightly-2023-12-03 as builder

ADD . /go-ethereum

ARG SCROLL_LIB_PATH

RUN apt update && apt install vim netcat-openbsd net-tools curl -y
RUN mkdir -p $SCROLL_LIB_PATH

COPY --from=zkp-builder /app/target/release/libzkp.so $SCROLL_LIB_PATH

ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$SCROLL_LIB_PATH
ENV CGO_LDFLAGS="-L$SCROLL_LIB_PATH -Wl,-rpath,$SCROLL_LIB_PATH"

RUN cd /go-ethereum && env GO111MODULE=on go run build/ci.go install -buildtags circuit_capacity_checker ./cmd/geth

# Pull Geth into a second stage deploy alpine container
FROM ubuntu:20.04

RUN apt-get -qq update \
    && apt-get -qq install -y --no-install-recommends ca-certificates

COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

ARG SCROLL_LIB_PATH

RUN mkdir -p $SCROLL_LIB_PATH

COPY --from=zkp-builder /app/target/release/libzkp.so $SCROLL_LIB_PATH

ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$SCROLL_LIB_PATH
ENV CGO_LDFLAGS="-ldl -L$SCROLL_LIB_PATH -Wl,-rpath,$SCROLL_LIB_PATH"

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["geth"]

# Add some metadata labels to help programatic image consumption
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
