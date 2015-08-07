FROM ubuntu:utopic
MAINTAINER caktux

ENV DEBIAN_FRONTEND noninteractive

# Usual update / upgrade
RUN apt-get update
RUN apt-get upgrade -q -y
RUN apt-get dist-upgrade -q -y

# Let our containers upgrade themselves
RUN apt-get install -q -y unattended-upgrades

# Install Expanse
RUN apt-get install -q -y software-properties-common
RUN add-apt-repository ppa:expanse/expanse
RUN add-apt-repository ppa:expanse/expanse-dev
RUN apt-get update
RUN apt-get install -q -y gexp

# Install supervisor
RUN apt-get install -q -y supervisor

# Add supervisor configs
ADD supervisord.conf supervisord.conf

EXPOSE 9656
EXPOSE 60606

CMD ["-n", "-c", "/supervisord.conf"]
ENTRYPOINT ["/usr/bin/supervisord"]
