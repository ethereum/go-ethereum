FROM alpine:3.8

RUN apk add --no-cache --update                                   \
            ca-certificates                                       \
            gcc                                                   \
            git                                                   \
            go                                                    \
            linux-headers                                         \
            make                                                  \
            musl-dev                                            &&\  
    #Clone
    git clone --depth 1 https://github.com/ethereum/go-ethereum &&\
    cd /go-ethereum                                             &&\
    make geth                                                   &&\
    cp /go-ethereum/build/bin/geth /geth                        &&\
    #CleanUp
    apk del                                                       \
        gcc                                                       \
        git                                                       \
        go                                                        \
        linux-headers                                             \
        make                                                      \
        musl-dev                                                &&\
        rm -rf ~/go-ethereum                                    &&\
        rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/geth"]
