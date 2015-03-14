#!/bin/bash

# Strict mode
set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
TEST_DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

echo -e "\n################# Testing JS ##################"
# TODO: Use mocha and real testing tools instead of rolling our own
cd $TEST_DIR/../js 
if [ -x "$(which nodejs)" ] ; then 
	nodejs test.js
fi
if [ -x "$(which node)" ] ; then 
	node test.js
fi

echo -e "\n################# Testing C ##################"
$TEST_DIR/c/test.sh

echo -e "\n################# Testing Python ##################"
$TEST_DIR/python/test.sh

#echo "################# Testing Go ##################"
#$TEST_DIR/go/test.sh
