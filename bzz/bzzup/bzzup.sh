#! /bin/bash

INDEX='index.html'

bzzroot="$1"
[ "_$1" = _ ] && bzzroot=.

delimiter='{"entries":[{'

pushd "$bzzroot" > /dev/null

(for path in `find . -type f`
do
name=`echo "$path" | cut -c2-`
[ _`basename "$name"` = "_$INDEX" ] && name=`dirname "$name"`
echo -n "$delimiter"
hash=`wget -q -O- --post-file="$path" http://localhost:8500/raw`
mime=`mimetype -b "$path"`
echo -n "\"hash\":\"$hash\",\"path\":\"$name\",\"contentType\":\"$mime\""
delimiter='},{'

done
echo -n '}]}') | wget -q -O- --post-data=`cat` http://localhost:8500/raw

echo

popd > /dev/null

