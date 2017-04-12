FROM ubuntu:xenial

RUN \
  apt-get update && apt-get upgrade -q -y && \
  apt-get install -y --no-install-recommends golang git make gcc libc-dev ca-certificates && \
  git clone --depth 1 --branch release/1.5 https://github.com/ubiq/go-ubiq && \
  (cd go-ubiq && make gubiq) && \
  cp go-ubiq/build/bin/gubiq /gubiq && \
  apt-get remove -y golang git make gcc libc-dev && apt autoremove -y && apt-get clean && \
  rm -rf /go-ubiq

EXPOSE 8588
EXPOSE 30388

ENTRYPOINT ["/gubiq"]
