#! /bin/bash

go get github.com/sirupsen/logrus

for d in plugins/*/ ; do
	CWD=$(pwd)
    echo "fetch dependencies $d"
	cd "$d"
	./dependencies.sh
	cd "$CWD"
done
