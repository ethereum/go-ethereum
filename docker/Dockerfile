FROM ubuntu:utopic
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

RUN git clone http://www.github.com/expanse-project/go-expanse.git && cd go-expanse
RUN git checkout develop
RUN make gexp

# Install supervisor
RUN apt-get install -q -y supervisor

# Add supervisor configs
ADD supervisord.conf supervisord.conf

EXPOSE 9656
EXPOSE 60606

CMD ["-n", "-c", "/supervisord.conf"]
ENTRYPOINT ["/usr/bin/supervisord"]
