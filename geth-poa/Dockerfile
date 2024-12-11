FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev linux-headers git make

COPY ../go.mod /go-ethereum/
COPY ../go.sum /go-ethereum/
RUN cd /go-ethereum && go mod download

ADD .. /go-ethereum
RUN cd /go-ethereum && go run build/ci.go install -static ./cmd/geth

FROM alpine:latest

RUN apk add --no-cache jq

COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

COPY geth-poa/genesis.json /genesis.json

COPY geth-poa/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

EXPOSE 8545 6060 60601

ENTRYPOINT ["/entrypoint.sh"] 
