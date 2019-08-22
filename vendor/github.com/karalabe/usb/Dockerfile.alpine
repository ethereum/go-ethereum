FROM golang:alpine

RUN apk add --no-cache git gcc musl-dev linux-headers

ADD . $GOPATH/src/github.com/karalabe/usb
RUN cd $GOPATH/src/github.com/karalabe/usb && go install
