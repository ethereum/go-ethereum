FROM ubuntu:xenial

# install build + test dependencies
RUN apt-get update && \
    apt-get install --yes --no-install-recommends \
      ca-certificates \
      curl \
      fuse \
      g++ \
      gcc \
      git \
      iproute2 \
      iputils-ping \
      less \
      libc6-dev \
      make \
      pkg-config \
      && \
    apt-get clean

# install Go
ENV GO_VERSION 1.8.1
RUN curl -fSLo golang.tar.gz "https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz" && \
    tar -xzf golang.tar.gz -C /usr/local && \
    rm golang.tar.gz
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

# install docker CLI
RUN curl -fSLo docker.tar.gz https://get.docker.com/builds/Linux/x86_64/docker-17.04.0-ce.tgz && \
  tar -xzf docker.tar.gz -C /usr/local/bin --strip-components=1 docker/docker && \
  rm docker.tar.gz

# install jq
RUN curl -fSLo /usr/local/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 && \
    chmod +x /usr/local/bin/jq

# install govendor
RUN go get -u github.com/kardianos/govendor

# add custom bashrc
ADD bashrc /root/.bashrc
