FROM golang:1.12-alpine as builder
RUN apk add --no-cache make gcc musl-dev linux-headers git
ADD . /swarm
WORKDIR /swarm
RUN make alltools

FROM ethereum/client-go:v1.8.27 as geth

FROM alpine:3.9
RUN apk --no-cache add ca-certificates
COPY --from=builder /swarm/build/bin/* /usr/local/bin/
COPY --from=geth /usr/local/bin/geth /usr/local/bin/
COPY docker/run.sh /run.sh
COPY docker/run-smoke.sh /run-smoke.sh
ENTRYPOINT ["/run.sh"]
