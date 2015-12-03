#! /bin/bash

INDEX='index.html'
port="8500"
delimiter='{"entries":[{'

if [[ ! -z "$2" ]]; then
  port="$2"
fi

if [ -f "$1" ]; then
hash=`wget -q -O- --post-file="$1" http://localhost:$port/raw`
mime=`mimetype -b "$1"`
wget -q -O- --post-data="$delimiter\"hash\":\"$hash\",\"contentType\":\"$mime\"}]}" http://localhost:8500/raw
echo

else

[ -d "$1" ] || exit -1

bzzroot="$1"
[ "_$1" = _ ] && bzzroot=.

pushd "$bzzroot" > /dev/null

(for path in `find . -type f`
do
name=`echo "$path" | cut -c3-`
[ _`basename "$name"` = "_$INDEX" ] && name=`dirname "$name"`
echo -n "$delimiter"
hash=`wget -q -O- --post-file="$path" http://localhost:$port/raw`
mime=`mimetype -b "$path"`
if [ "_$name" = '_.' ]; then
echo -n "\"hash\":\"$hash\",\"contentType\":\"$mime\""
else
echo -n "\"hash\":\"$hash\",\"path\":\"$name\",\"contentType\":\"$mime\""
fi
delimiter='},{'

done
echo -n '}]}') | wget -q -O- --post-data=`cat` http://localhost:$port/raw
echo

popd > /dev/null

fi
