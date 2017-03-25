# Copyright (C) 2016  Arista Networks, Inc.
# Use of this source code is governed by the Apache License 2.0
# that can be found in the COPYING file.

# TODO: move this to cmd/ockafka (https://github.com/docker/hub-feedback/issues/292)
FROM golang:1.7.3

RUN mkdir -p /go/src/github.com/aristanetworks/goarista/cmd
WORKDIR /go/src/github.com/aristanetworks/goarista
COPY ./ .
RUN go get -d ./cmd/ockafka/... \
  && go install ./cmd/ockafka

ENTRYPOINT ["/go/bin/ockafka"]
