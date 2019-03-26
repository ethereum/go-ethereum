# Build Gubiq in a stock Go builder container
FROM golang:1.11-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /go-ubiq
RUN cd /go-ubiq && make gubiq

# Pull Gubiq into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-ubiq/build/bin/gubiq /usr/local/bin/

EXPOSE 8588 8589 30388 30388/udp
ENTRYPOINT ["gubiq"]
