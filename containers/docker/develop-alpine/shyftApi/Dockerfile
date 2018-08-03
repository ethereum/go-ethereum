FROM golang:1.10.3-alpine

RUN apk add --update git make gcc musl-dev linux-headers ca-certificates

COPY ./shyftBlockExplorerApi /go/src/github.com/ShyftNetwork/go-empyrean/shyftBlockExplorerApi
COPY ./wait-for.sh /
WORKDIR /go/src/github.com/ShyftNetwork/go-empyrean/shyftBlockExplorerApi

RUN go get -u github.com/kardianos/govendor
RUN govendor sync
RUN cat /go/src/github.com/ShyftNetwork/go-empyrean/shyftBlockExplorerApi/vendor/vendor.json
RUN echo `ls -l /go/src/github.com/ShyftNetwork/go-empyrean/shyftBlockExplorerApi/vendor`
CMD go run -v *.go

EXPOSE 8080
