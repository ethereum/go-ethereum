#!/usr/bin/env bash

## This is only a note to "self" so i do not forget the steps.
## Ugly code, live with it.

go_path="./go";

if [[ -d $go_path ]]; then
    cd $go_path && git clone https://github.com/karalabe/xgo && cd xgo/ && go build
    docker pull karalabe/xgo-latest
    ./xgo --targets="linux/amd64,windows/amd64,darwin/amd64" --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 -pkg=cmd/shift github.com/chattynet/chatty
else
    echo "Create the directory $go_path";
    exit 1;
fi

exit 0;
