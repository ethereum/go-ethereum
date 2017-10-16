FROM ubuntu:xenial

RUN \
  apt-get update && apt-get upgrade -q -y && \
  apt-get install -y --no-install-recommends golang git make gcc libc-dev ca-certificates && \
  git clone --depth 1 --branch release/1.7 https://github.com/expanse-org/go-expanse && \
  (cd go-expanse && make gexp) && \
  cp go-expanse/build/bin/gexp /gexp && \

  apt-get remove -y golang git make gcc libc-dev && apt autoremove -y && apt-get clean && \
  rm -rf /go-expanse

EXPOSE 9656
EXPOSE 42786

ENTRYPOINT ["/gexp"]
