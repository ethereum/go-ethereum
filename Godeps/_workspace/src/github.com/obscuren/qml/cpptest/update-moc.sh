#!/bin/sh

set -e
cd `dirname $0`

export QT_SELECT=5

for file in `grep -l Q_''OBJECT *`; do
	mocfile=`echo $file | awk -F. '{print("moc_"$1".cpp")}'`
	mochack=`sed -n 's,^ *// MOC HACK: \(.*\),\1,p' $file`
	moc $file | sed "$mochack" > $mocfile
done
