#!/bin/bash

cd ./shyft-cli/postgres_setup
psql -U postgres -d shyftdb -f drop_tables.psql