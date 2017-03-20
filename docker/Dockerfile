FROM ubuntu:wily
MAINTAINER chrisfranko


ENV DEBIAN_FRONTEND noninteractive

# Usual update / upgrade
RUN apt-get update
RUN apt-get upgrade -q -y
RUN apt-get dist-upgrade -q -y

# Let our containers upgrade themselves
RUN apt-get install -q -y unattended-upgrades

# Install Expanse
RUN apt-get install -q -y curl git mercurial binutils bison gcc make libgmp3-dev build-essential

# Install Go
RUN \
  mkdir -p /goroot && \
  curl https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz | tar xvzf - -C /goroot --strip-components=1

# Set environment variables.
ENV GOROOT /goroot
ENV GOPATH /gopath
ENV PATH $GOROOT/bin:$GOPATH/bin:$PATH

RUN git clone http://www.github.com/expanse-org/go-expanse.git
RUN cd go-expanse && make gexp

RUN cp build/bin/gexp /usr/bin/gexp

EXPOSE 9656
EXPOSE 42786

ENTRYPOINT ["/usr/bin/gexp"]
