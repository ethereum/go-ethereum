#! /bin/bash

for d in plugins/*/ ; do
	CWD=$(pwd)
    echo "fetch dependencies $d"
	cd "$d"
	./dependencies.sh
	cd "$CWD"
done

for d in plugins/*/ ; do
	CWD=$(pwd)
    echo "building $d"
	cd "$d"
	/usr/local/go/bin/go build -buildmode=plugin -ldflags "-extldflags '-Wl,-z,stack-size=0x800000'" -tags "urfave_cli_no_docs,ckzg,purego" -trimpath -v -o plugin.so
	cd "$CWD"
done
