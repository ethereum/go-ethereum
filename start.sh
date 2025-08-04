#!/bin/sh
set -e

# Wait for Postgres to be ready (assumes external Postgres service)
until pg_isready -h "$DB_HOST" -p "$DB_PORT"; do
  echo "Waiting for Postgres..."
  sleep 2
done

# Start Geth
geth --dev \
  --http --http.addr 0.0.0.0 \
  --http.vhosts "*" --http.api eth,net,web3,personal \
  --ws --ws.addr 0.0.0.0 \
  --allow-insecure-unlock --mine --nodiscover &

# Set environment variables for Blockscout
export DATABASE_URL="postgresql://postgres:postgres@${DB_HOST:-localhost}:${DB_PORT:-5432}/blockscout"
export ETHEREUM_JSONRPC_HTTP_URL="http://localhost:8545"
export PORT=4000
export MIX_ENV=prod

cd /blockscout

# Run DB setup
#mix ecto.create
#mix ecto.migrate

# Start Blockscout
/blockscout/_build/prod/rel/blockscout/bin/blockscout start

wait