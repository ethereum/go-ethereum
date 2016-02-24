#!/usr/bin/env bash
# install all required packages for rocksdb
sudo apt-get update
sudo apt-get -y install git make gcc g++ libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev default-jdk

# set java home so we can build rocksdb jars
export JAVA_HOME=$(echo /usr/lib/jvm/java-7-openjdk*)
cd /rocksdb
make jclean clean
make -j 4 rocksdbjavastatic
cp /rocksdb/java/target/librocksdbjni-* /rocksdb-build
cp /rocksdb/java/target/rocksdbjni-* /rocksdb-build
sudo shutdown -h now

