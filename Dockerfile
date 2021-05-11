# Build Geth in a stock Go builder container
FROM golang:1.16.4-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /go-ethereum
RUN cd /go-ethereum && make geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
RUN adduser --disabled-password --home /home/ethereum ethereum
RUN chown ethereum:ethereum /home/ethereum

USER ethereum
WORKDIR /home/ethereum

COPY --from=builder --chown=ethereum:ethereum /go-ethereum/build/bin/geth /home/ethereum/bin/

ENV PATH="/home/ethereum/bin:${PATH}"


EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["geth"]
