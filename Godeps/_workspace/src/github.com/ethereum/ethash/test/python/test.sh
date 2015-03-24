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

[ -d $TEST_DIR/python-virtual-env ] || virtualenv --system-site-packages $TEST_DIR/python-virtual-env
source $TEST_DIR/python-virtual-env/bin/activate
pip install -r $TEST_DIR/requirements.txt > /dev/null
pip install --upgrade --no-deps --force-reinstall -e $TEST_DIR/../..
cd $TEST_DIR
nosetests --with-doctest -v --nocapture
