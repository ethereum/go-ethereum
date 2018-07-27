FROM golang:1.10.3-alpine

# To copy complete directory
COPY ./ /go/src/github.com/ShyftEthereum/go-empyrean
WORKDIR /go/src/github.com/ShyftEthereum/go-empyrean

RUN \
  apk add --update git make gcc musl-dev linux-headers ca-certificates && \
  (cd /go/src/github.com/ShyftEthereum/go-empyrean && make geth) && \
  cp -v /go/src/github.com/ShyftEthereum/go-empyrean/build/bin/geth /bin 

COPY config.toml /
COPY unlockPasswords.txt /
COPY ShyftNetwork.json /

COPY UTC--2018-03-18T22-35-03.111589431Z--43ec6d0942f7faef069f7f63d0384a27f529b062 /
COPY UTC--2018-03-18T22-36-53.786508893Z--9e602164c5826ebb5a6b68e4afd9cd466043dc4a /
COPY UTC--2018-03-18T22-37-04.212101982Z--5bd738164c61fb50eb12e227846cbaef2de965aa /
COPY UTC--2018-03-18T22-37-12.082606217Z--c04ee4131895f1d0c294d508af65d94060aa42bb /
COPY UTC--2018-03-18T22-37-19.650676366Z--07d899c4ac0c1725c35c5f816e60273b33a964f7 /

WORKDIR /go/src/github.com/ShyftEthereum/go-empyrean

CMD ["./shyft-cli/initShyftGeth.sh", "./shyft-cli/startShyftGeth.sh"]
