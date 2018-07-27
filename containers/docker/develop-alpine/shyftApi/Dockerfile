FROM golang:1.10.3-alpine

COPY ./shyftBlockExplorerApi /go/src/github.com/ShyftNetwork/go-empyrean/shyftBlockExplorerApi
# COPY ./vendor /go/src/github.com/ShyftNetwork/go-empyrean/vendor
COPY ./core /go/src/github.com/ShyftNetwork/go-empyrean/core
# COPY ./common /go/src/github.com/ShyftNetwork/go-empyrean/common
COPY .git /go/src/github.com/ShyftNetwork/go-empyrean/.git


WORKDIR /go/src/github.com/ShyftNetwork/go-empyrean/shyftBlockExplorerApi

RUN apk add --update git make gcc musl-dev linux-headers ca-certificates
RUN go get ./

RUN go build

CMD go get github.com/pilu/fresh && fresh

EXPOSE 8080
