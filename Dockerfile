FROM golang:latest

ARG BOR_DIR=/var/lib/bor
ENV BOR_DIR=$BOR_DIR

RUN apt-get update -y && apt-get upgrade -y \
    && apt install build-essential git -y \
    && mkdir -p ${BOR_DIR}

WORKDIR ${BOR_DIR}
COPY . .
RUN make bor

RUN cp build/bin/bor /usr/bin/

ENV SHELL /bin/bash
EXPOSE 8545 8546 8547 30303 30303/udp

ENTRYPOINT ["bor"]
