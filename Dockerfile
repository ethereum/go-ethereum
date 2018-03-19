# Build Geth in a stock Go builder container
FROM golang:1.9-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /go-etherinc
RUN cd /go-etherinc && make geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-etherinc/build/bin/geth /usr/local/bin/

EXPOSE 8545 8546 30103 30103/udp 30104/udp
ENTRYPOINT ["geth"]
