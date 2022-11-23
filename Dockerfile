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
RUN groupadd -g 10137 bor \
    && useradd -u 10137 --no-log-init --create-home -r -g bor bor \
    && chown -R bor:bor ${BOR_DIR}

ENV SHELL /bin/bash
EXPOSE 8545 8546 8547 30303 30303/udp

ENTRYPOINT ["bor"]
