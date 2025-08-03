# Base Geth build stage remains the same

# Additional stage for Blockscout dependencies
FROM hexpm/elixir:1.14.5-erlang-25.3.2.6-alpine-3.18.4 as blockscout-builder

RUN apk add --no-cache build-base git curl postgresql-dev inotify-tools npm gcompat

# Clone Blockscout source
RUN git clone https://github.com/blockscout/blockscout.git /blockscout

WORKDIR /blockscout

RUN mix local.hex --force && \
    mix local.rebar --force && \
    mix deps.get && \
    mix compile

RUN mix do ecto.create, ecto.migrate

RUN mix phx.digest && MIX_ENV=prod mix release

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
