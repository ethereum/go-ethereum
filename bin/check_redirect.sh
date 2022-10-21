#!/bin/sh

urls=(
  "http://geth.ethereum.org"
  "https://geth.ethereum.org"
  "http://geth.ethereum.org/"
  "https://geth.ethereum.org/"
  "http://geth.ethereum.org/install"
  "https://geth.ethereum.org/install"
  "http://geth.ethereum.org/install/"
  "https://geth.ethereum.org/install/"
  "http://ethereum.github.io/go-ethereum" 
  "https://ethereum.github.io/go-ethereum" 
  "http://ethereum.github.io/go-ethereum/" 
  "https://ethereum.github.io/go-ethereum/" 
  "http://ethereum.github.io/go-ethereum/install" 
  "https://ethereum.github.io/go-ethereum/install" 
  "http://ethereum.github.io/go-ethereum/install/" 
  "https://ethereum.github.io/go-ethereum/install/" 
)
for u in "${urls[@]}"
do
	echo "$u -> $(curl $u -w --silent -I 2>&1 | grep Location)"
done
