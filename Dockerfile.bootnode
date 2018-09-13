FROM golang:1.10-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /tomochain
RUN cd /tomochain && make bootnode

RUN chmod +x /tomochain/build/bin/bootnode

FROM alpine:latest

LABEL maintainer="etienne@tomochain.com"

WORKDIR /tomochain

COPY --from=builder /tomochain/build/bin/bootnode /usr/local/bin/bootnode

COPY docker/bootnode ./

EXPOSE 30301

ENTRYPOINT ["./entrypoint.sh"]

CMD ["-verbosity", "6", "-nodekey", "bootnode.key", "--addr", ":30301"]
