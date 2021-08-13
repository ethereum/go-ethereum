# syntax=docker/dockerfile:1
FROM golang:1.15.15-alpine3.14 AS builder

RUN apk add --no-cache gcc musl-dev linux-headers git

ADD . /go-ethereum
WORKDIR /go-ethereum

RUN go run build/ci.go install ./cmd/geth

# Pull all binaries into a second stage deploy alpine container
FROM alpine:3.14.2
RUN apk add --no-cache curl ca-certificates

COPY --from=builder /go-ethereum/build/bin/* /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp


COPY ./entrypoint.sh /root/entrypoint.sh
# we could ---chmod= when we copy, but we do this intentionally 
RUN chmod 755 /root/entrypoint.sh

ENTRYPOINT ["geth"]

ARG BUILD_DATE
ARG VCS_REF
ARG BUILD_VERSION
ARG VERSION
LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.name="MEV Go-Ethereum" \
      org.label-schema.description="MEV Go Ethereum Alpine" \
      org.label-schema.url="http://vcs.openmev.org/" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url="https://github.com/openmev/vcs.git" \
      org.label-schema.vendor="OpenMEV" \
      org.label-schema.version=$VERSION \
      org.label-schema.schema-version="1.0" \
      org.label-schema.build-date=$BUILD_DATE