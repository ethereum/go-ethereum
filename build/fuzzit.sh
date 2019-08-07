#!/bin/bash
set -xe
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

if [ -z ${1+x} ]; then
    echo "must call with job type as first argument e.g. 'fuzzing' or 'sanity'"
    echo "see https://github.com/fuzzitdev/example-go/blob/master/.travis.yml"
    exit 1
fi

$DIR/libfuzzer_targets.sh

# submit fuzz targets to fuzzit.dev for continuous fuzzing

wget -q -O fuzzit https://github.com/fuzzitdev/fuzzit/releases/download/v2.4.12/fuzzit_Linux_x86_64
chmod a+x fuzzit

for TARGET in "${TARGETS[@]}"
do
    # create fuzzing target on the server if it doesn't already exist
    ./fuzzit create target ${TARGET} || true

    # submit a new fuzzing job for that target
    if [ $1 == "fuzzing" ]; then
        ./fuzzit auth ${FUZZIT_API_KEY}
        ./fuzzit create job --branch $TRAVIS_BRANCH --revision $TRAVIS_COMMIT ${TARGET} "./${TARGET}"
    else
        ./fuzzit create job --local ethereum/${TARGET} "./${TARGET}"
    fi
done
