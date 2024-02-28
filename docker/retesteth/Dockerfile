FROM alpine:3.15 AS base
RUN apk add --no-cache alpine-sdk cmake linux-headers boost-dev boost-static
ENV LDFLAGS="-static"
WORKDIR /build

FROM base as tests
RUN git clone --depth 1 -b develop https://github.com/ethereum/tests.git /tests

FROM base AS retesteth
RUN git clone --depth 1 -b develop https://github.com/ethereum/retesteth.git /retesteth
RUN cmake /retesteth -DCMAKE_BUILD_TYPE=Release && make && cp retesteth/retesteth /target

# lllc compiles Lisp Like Language, an old Ethereum smart contract language that is still in use in tests.
# Ref: https://github.com/ethereum/retesteth
FROM alpine:3.13 AS lllc
RUN apk add --no-cache alpine-sdk cmake linux-headers boost-dev boost-static
ENV LDFLAGS="-static"
WORKDIR /build
RUN git clone --depth 1 -b master https://github.com/winsvega/solidity.git /lllc
RUN cmake /lllc -DCMAKE_BUILD_TYPE=Release -DLLL=1 && make lllc && cp lllc/lllc /target

FROM base AS solc
RUN git clone --depth 1 -b v0.8.24 https://github.com/ethereum/solidity.git /solidity
RUN touch /solidity/prerelease.txt && cmake /solidity -DCMAKE_BUILD_TYPE=Release && make solc && cp solc/solc /target

# Ref: https://github.com/ethereum/go-ethereum/blob/master/Dockerfile
FROM golang:1.21-alpine as geth
RUN apk add --no-cache gcc musl-dev linux-headers git
WORKDIR /go-ethereum
COPY go.mod .
COPY go.sum .
RUN go mod download
ADD . .
RUN go run build/ci.go install ./cmd/evm
RUN cp build/bin/evm /target

FROM alpine:3.15
ENV ETHEREUM_TEST_PATH="/tests"
COPY --from=retesteth /target /bin/retesteth
COPY --from=lllc /target /bin/lllc
COPY --from=solc /target /bin/solc
COPY --from=tests /tests /tests
COPY --from=geth /target /bin/evm
COPY docker/retesteth/data /root/.retesteth
