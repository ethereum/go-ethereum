FROM ubuntu:xenial

ENV PATH=/usr/lib/go-1.10/bin:$PATH

RUN apt-get update                              &&\
    apt-get upgrade -q -y                       &&\
    apt-get install -y --no-install-recommends    \
    ca-certificates                               \
    gcc                                           \
    git                                           \
    golang-1.10                                   \
    libc-dev                                      \
    make                                        &&\
    #Clone and build    
    git clone --depth 1                           \
        https://github.com/ethereum/go-ethereum &&\
    cd /go-ethereum                             &&\
    make geth                                   &&\
    cp /go-ethereum/build/bin/geth /geth        &&\
    #Cleanup
    apt-get remove -y                             \
    golang-1.10                                   \
            git                                   \
            make                                  \
            gcc                                   \
            libc-dev                            &&\
    apt-get autoremove -y                       &&\
    apt-get clean                               &&\
    rm -rf /go-ethereum            

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/geth"]
   
