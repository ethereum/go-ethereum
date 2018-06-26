FROM golang:1.10-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /tomochain
RUN cd /tomochain && make tomo

FROM alpine:latest

LABEL maintainer="etienne@tomochain.com"

WORKDIR /tomochain

COPY --from=builder /tomochain/build/bin/tomo /usr/local/bin/tomo

RUN chmod +x /usr/local/bin/tomo

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/usr/local/bin/tomo", "--help"]
