# Build Bor in a stock Go builder container
FROM golang:1.12-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /bor
RUN cd /bor && make bor

# Pull Bor into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /bor/build/bin/bor /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp