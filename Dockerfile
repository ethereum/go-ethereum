# Build Geth in a stock Go builder container
FROM golang:1.17-alpine as builder

RUN set -x \
	&& buildDeps='bash build-base musl-dev linux-headers git' \
	&& apk add --update $buildDeps \
	&& rm -rf /var/cache/apk/* \
    && mkdir -p /bor

WORKDIR /bor
COPY . .
RUN go build -o ./build/bin/bor ./command

# Pull Bor into a second stage deploy alpine container
FROM alpine:3.14

RUN set -x \
    && apk add --update --no-cache \
       ca-certificates \
    && rm -rf /var/cache/apk/*

COPY --from=builder /bor/build/bin/bor /usr/local/bin/

EXPOSE 8545 8546 8547 30303 30303/udp

ENTRYPOINT ["bor"]
