cmd_Release/secp256k1.a := rm -f Release/secp256k1.a && ./gyp-mac-tool filter-libtool libtool  -static -o Release/secp256k1.a Release/obj.target/secp256k1/src/secp256k1/src/secp256k1.o
