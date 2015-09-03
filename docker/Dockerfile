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
RUN curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer | bash
RUN /bin/bash -c "source /root/.gvm/scripts/gvm"
RUN gvm install go1.4
RUN gvm use go1.4 --default
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
