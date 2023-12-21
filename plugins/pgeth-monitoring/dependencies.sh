#! /bin/bash

go get github.com/ethereum/go-ethereum/plugins/pgeth-monitoring
go get github.com/redis/go-redis/v9@v9.0.3
go get github.com/prometheus/client_golang@v1.12.0
go get github.com/attestantio/go-eth2-client/http@v0.16.0
go mod tidy
