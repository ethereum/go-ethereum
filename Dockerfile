FROM alpine:3.5 as builder

ADD . /go-ethereum
ARG MAKE_TARGET=geth
RUN \
  apk add --update git go make gcc musl-dev linux-headers && \
  (cd go-ethereum && make $MAKE_TARGET)                   && \
  echo "Dockerfile builder stage finished."
#  cp go-ethereum/build/bin/geth /usr/local/bin/           && \
#  apk del git go make gcc musl-dev linux-headers         && \
#  rm -rf /go-ethereum && rm -rf /var/cache/apk/*

FROM alpine:3.5

COPY --from=builder /go-ethereum/build/bin/* /usr/local/bin/

EXPOSE 8545
EXPOSE 30303
EXPOSE 30303/udp

ENTRYPOINT ["geth"]
