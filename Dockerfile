FROM golang:1.23-alpine AS builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /XDPoSChain
RUN cd /XDPoSChain && make XDC

FROM alpine:latest

WORKDIR /XDPoSChain

COPY --from=builder /XDPoSChain/build/bin/XDC /usr/local/bin/XDC

RUN chmod +x /usr/local/bin/XDC

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/usr/local/bin/XDC"]

CMD ["--help"]
