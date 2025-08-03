#!/bin/sh
# Start PostgreSQL for Blockscout
docker-entrypoint.sh postgres &

# Wait for DB to be ready
until pg_isready -h localhost -p 5432; do
  echo "Waiting for Postgres..."
  sleep 2
done

# Start Geth
geth --dev \
  --http --http.addr 0.0.0.0 \
  --http.vhosts "*" --http.api eth,net,web3,personal \
  --ws --ws.addr 0.0.0.0 \
  --allow-insecure-unlock --mine --nodiscover &

# Start Blockscout
cd /blockscout && \
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/blockscout \
ETHEREUM_JSONRPC_HTTP_URL=http://localhost:8545 \
PORT=4000 \
MIX_ENV=prod \
/blockscout/_build/prod/rel/blockscout/bin/blockscout start

wait