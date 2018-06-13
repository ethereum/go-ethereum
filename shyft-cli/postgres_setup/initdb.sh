#!/bin/bash

cd ./shyft-cli/postgres_setup
psql -U postgres -f create_shyftdb.psql
