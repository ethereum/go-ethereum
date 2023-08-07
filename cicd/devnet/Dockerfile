FROM golang:1.14 as builder

RUN apt-get update && apt-get install -y git build-essential

COPY . /builder
RUN mv /builder/common/constants/constants.go.devnet /builder/common/constants.go

RUN cd /builder && make


# The actual image for devnet containers
FROM golang:1.14

RUN apt-get update && apt-get install -y git build-essential

WORKDIR /work

COPY --from=builder /builder/build/bin/XDC /usr/bin
RUN chmod +x /usr/bin/XDC
    
# Copy over files
ADD cicd/devnet/genesis.json /work/genesis.json
ADD cicd/devnet/bootnodes.list /work/bootnodes.list
ADD cicd/devnet/start.sh /work/start.sh

# Create an empty pwd file
RUN touch /work/.pwd

# rpc
EXPOSE 8545
# ws
EXPOSE 8555
# port
EXPOSE 30303

ENTRYPOINT ["bash","/work/start.sh"]