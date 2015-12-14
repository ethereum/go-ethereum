#! /bin/bash

INDEX='index.html'
proxy="http://localhost:8500"
delimiter='{"entries":[{'


if [[ ! -z "$2" ]]; then
  proxy="$2"
fi

if [ -f "$1" ]; then
hash=`wget -q -O- --post-file="$1" $proxy/bzzr:/`
mime=`mimetype -b "$1"`
# echo wget -q -O- --post-data="$delimiter\"hash\":\"$hash\",\"contentType\":\"$mime\"}]}" $proxy/bzzr:/
wget -q -O- --post-data="$delimiter\"hash\":\"$hash\",\"contentType\":\"$mime\"}]}" $proxy/bzzr:/
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
hash=`wget -q -O- --post-file="$path" $proxy/bzzr:/`
mime=`mimetype -b "$path"`
if [ "$mime" = "text/plain" ]; then
   echo -n $path|grep -q '.css' && mime="text/css"
fi
if [ "_$name" = '_.' ]; then
echo -n "\"hash\":\"$hash\",\"contentType\":\"$mime\""
else
echo -n "\"hash\":\"$hash\",\"path\":\"$name\",\"contentType\":\"$mime\""
fi
delimiter='},{'

done
echo -n '}]}') | wget -q -O- --post-data=`cat` $proxy/bzzr:/
echo

popd > /dev/null

fi
