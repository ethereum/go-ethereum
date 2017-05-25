FROM vertigo/go-builder as builder

ADD . /go-ethereum
RUN \
  (cd go-ethereum && make geth)                           && \
  cp go-ethereum/build/bin/geth /geth                     && \
  rm -rf /go-ethereum

FROM alpine:3.5

COPY --from=builder /geth /geth

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/geth"]
