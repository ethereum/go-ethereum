FROM golang:latest

ARG BOR_DIR=/bor
ENV BOR_DIR=$BOR_DIR

RUN apt-get update -y && apt-get upgrade -y \
    && apt install build-essential git -y \
    && mkdir -p /bor

WORKDIR ${BOR_DIR}
COPY . .
RUN make bor-all

ENV SHELL /bin/bash
EXPOSE 8545 8546 8547 30303 30303/udp

ENTRYPOINT ["bor"]
