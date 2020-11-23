FROM golang:1.10-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /XDCchain
RUN cd /XDCchain && make XDC

FROM alpine:latest

LABEL maintainer="anil@xinfin.org"

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["XDC"]
