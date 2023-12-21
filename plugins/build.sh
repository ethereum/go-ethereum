#! /bin/bash

for d in plugins/*/ ; do
	CWD=$(pwd)
    echo "fetch dependencies $d"
	cd "$d"
	./dependencies.sh
	cd "$CWD"
done
