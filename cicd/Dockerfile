FROM golang:1.23-alpine AS builder

RUN apk add make build-base linux-headers

COPY . /builder
RUN cd /builder && make && mv /builder/build/bin/XDC /builder/build/bin/XDC-mainnet

RUN mv /builder/common/constants/constants.go.testnet /builder/common/constants.go
RUN cd /builder && make && mv /builder/build/bin/XDC /builder/build/bin/XDC-testnet

RUN mv /builder/common/constants/constants.go.devnet /builder/common/constants.go
RUN cd /builder && make && mv /builder/build/bin/XDC /builder/build/bin/XDC-devnet

RUN mv /builder/common/constants/constants.go.local /builder/common/constants.go
RUN cd /builder && make && mv /builder/build/bin/XDC /builder/build/bin/XDC-local

# The runtime image
FROM alpine:3

WORKDIR /work

RUN apk add --no-cache bash curl

COPY --from=builder /builder/build/bin/XDC-local /usr/bin
COPY --from=builder /builder/build/bin/XDC-devnet /usr/bin
COPY --from=builder /builder/build/bin/XDC-testnet /usr/bin
COPY --from=builder /builder/build/bin/XDC-mainnet /usr/bin

# # Copy over files
ADD cicd/local /work/local
ADD cicd/devnet /work/devnet
ADD cicd/testnet /work/testnet
ADD cicd/mainnet /work/mainnet
ADD cicd/entry.sh /work/

# Create an empty pwd file
RUN touch /work/.pwd

# rpc
EXPOSE 8545
# ws
EXPOSE 8555
# port
EXPOSE 30303

ENTRYPOINT ["bash","/work/entry.sh"]