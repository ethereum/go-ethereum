# Build Geth in a stock Go builder container
FROM golang:1.10-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /go-ethereum
RUN cd /go-ethereum && make geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

RUN addgroup -g 1000 geth && \
    adduser -h /root -D -u 1000 -G geth geth && \
    chown geth:geth /root

USER geth

EXPOSE 8545 8546 30303 30303/udp 30304/udp
ENTRYPOINT ["geth"]
