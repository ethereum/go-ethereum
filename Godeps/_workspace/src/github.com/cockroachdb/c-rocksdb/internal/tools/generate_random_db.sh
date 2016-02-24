#!/bin/bash
#
# A shell script to load some pre generated data file to a DB using ldb tool
# ./ldb needs to be avaible to be executed.
#
# Usage: <SCRIPT> <input_data_path> <DB Path>

if [ "$#" -lt 2 ]; then
  echo "usage: $BASH_SOURCE <input_data_path> <DB Path>"
  exit 1
fi

input_data_dir=$1
db_dir=$2
rm -rf $db_dir

echo == Loading data from $input_data_dir to $db_dir

declare -a compression_opts=("no" "snappy" "zlib" "bzip2")

set -e

n=0

for f in `ls -1 $input_data_dir`
do
  echo == Loading $f with compression ${compression_opts[n % 4]}
  ./ldb load --db=$db_dir --compression_type=${compression_opts[n % 4]} --bloom_bits=10 --auto_compaction=false --create_if_missing < $input_data_dir/$f
  let "n = n + 1"
done
