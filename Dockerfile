# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""
# Base Geth build stage remains the same

# Additional stage for Blockscout dependencies
FROM hexpm/elixir:1.17.0-erlang-27.0-alpine-3.19.1 as blockscout-builder

RUN apk add --no-cache build-base git curl postgresql-dev inotify-tools npm gcompat bash

# Clone Blockscout source
RUN git clone https://github.com/blockscout/blockscout.git /blockscout

WORKDIR /blockscout

RUN mix local.hex --force && \
    mix local.rebar --force && \
    mix deps.get && \
    mix compile

#RUN mix do ecto.create, ecto.migrate

RUN mix phx.digest && MIX_ENV=prod mix release

# Build Geth in a stock Go builder container
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev linux-headers git

# Get dependencies - will also be cached if we won't change go.mod/go.sum
COPY go.mod /go-ethereum/
COPY go.sum /go-ethereum/
RUN cd /go-ethereum && go mod download

ADD . /go-ethereum
RUN cd /go-ethereum && go run build/ci.go install -static ./cmd/geth

# Final image
FROM alpine:latest

RUN apk add --no-cache ca-certificates libstdc++ postgresql-client su-exec bash curl

COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/
COPY --from=blockscout-builder /blockscout /blockscout


# Custom entrypoint
COPY start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 8545 8546 30303 4000
ENTRYPOINT ["/bin/sh", "/start.sh"]