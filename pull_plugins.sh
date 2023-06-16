#! /bin/bash

arrPLUGINS=(${PLUGIN_REPOSITORIES//;/ })

mkdir -p /tmp/plugins
CWD=$(pwd)
cd ./plugins

for i in "${arrPLUGINS[@]}"
do
	echo "cloning $i"
	git clone "$i"
done

cd $CWD