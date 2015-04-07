#!/bin/bash

# Strict mode
set -e

if [ -x "$(which virtualenv2)" ] ; then
   VIRTUALENV_EXEC=virtualenv2
elif [ -x "$(which virtualenv)" ] ; then
   VIRTUALENV_EXEC=virtualenv
else
   echo "Could not find a suitable version of virtualenv"
   false
fi

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
TEST_DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

[ -d $TEST_DIR/python-virtual-env ] || $VIRTUALENV_EXEC --system-site-packages $TEST_DIR/python-virtual-env
source $TEST_DIR/python-virtual-env/bin/activate
pip install -r $TEST_DIR/requirements.txt > /dev/null
# force installation of nose in virtualenv even if existing in thereuser's system
pip install nose -I
pip install --upgrade --no-deps --force-reinstall -e $TEST_DIR/../..
cd $TEST_DIR
nosetests --with-doctest -v --nocapture
