#!/bin/bash

if ! psql -lqt | cut -d \| -f 1 | grep -qw shyftdb; then    # Check if db is instantiated
    echo Creating postgres db...
    sh ./shyft-cli/postgres_setup/initdb.sh &&              # Init DB
    echo Successfully created postgres db! &&
    sh ./shyft-cli/postgres_setup/init_tables.sh &&         # Init tables
    sh ./shyft-cli/resetShyftGeth.sh &&                     # Reset geth data
    sh ./shyft-cli/initShyftGeth.sh                         # Init Shyft Geth
else
    echo Postgres DB found!
    sh ./shyft-cli/postgres_setup/drop_tables.sh &&         # Drop tables
    sh ./shyft-cli/postgres_setup/init_tables.sh &&         # Init tables
    sh ./shyft-cli/resetShyftGeth.sh &&                     # Reset geth data
    sh ./shyft-cli/initShyftGeth.sh                         # Init Shyft Geth
fi

