FROM ubuntu:xenial

ENV PATH=/usr/lib/go-1.9/bin:$PATH

RUN \
  apt-get update && apt-get upgrade -q -y && \
  apt-get install -y --no-install-recommends golang-1.9 git make gcc libc-dev ca-certificates && \
  git clone --depth 1 https://github.com/ubiq/go-ubiq && \
  (cd go-ubiq && make gubiq) && \
  cp go-ubiq/build/bin/gubiq /gubiq && \
  apt-get remove -y golang-1.9 git make gcc libc-dev && apt autoremove -y && apt-get clean && \
  rm -rf /go-ubiq

EXPOSE 8588
EXPOSE 30388

ENTRYPOINT ["/gubiq"]
