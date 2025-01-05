FROM golang:1.23-alpine

# Install dependencies
RUN apk add --no-cache ca-certificates

# Set the working directory
WORKDIR /go-ethereum

# Copy the source code
COPY . .

# Build Geth
RUN make geth

# Expose necessary ports
EXPOSE 8545 8546 30303

# Run Geth
CMD ["build/bin/geth"]

