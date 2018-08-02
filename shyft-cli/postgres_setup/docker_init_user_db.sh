#!/bin/bash
set -e
echo $POSTGRES_USER
echo $POSTGRES_DB
# createTables="$(cat /create_tables.psql)"
# echo $createTables
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE shyftdb;
EOSQL
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d shyftdb < /create_tables.psql
