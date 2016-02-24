#!/bin/bash

set -e

NUM=10000000

if [ $# -eq 1 ];then
  DATA_DIR=$1
elif [ $# -eq 2 ];then
  DATA_DIR=$1
  STAT_FILE=$2
fi

# On the production build servers, set data and stat
# files/directories not in /tmp or else the tempdir cleaning
# scripts will make you very unhappy.
DATA_DIR=${DATA_DIR:-$(mktemp -t -d rocksdb_XXXX)}
STAT_FILE=${STAT_FILE:-$(mktemp -t -u rocksdb_test_stats_XXXX)}

function cleanup {
  rm -rf $DATA_DIR
  rm -f $STAT_FILE.fillseq
  rm -f $STAT_FILE.readrandom
  rm -f $STAT_FILE.overwrite
  rm -f $STAT_FILE.memtablefillreadrandom
}

trap cleanup EXIT

if [ -z $GIT_BRANCH ]; then
  git_br=`git rev-parse --abbrev-ref HEAD`
else
  git_br=$(basename $GIT_BRANCH)
fi

if [ $git_br == "master" ]; then
  git_br=""
else
  git_br="."$git_br
fi

make release

# measure fillseq + fill up the DB for overwrite benchmark
./db_bench \
    --benchmarks=fillseq \
    --db=$DATA_DIR \
    --use_existing_db=0 \
    --bloom_bits=10 \
    --num=$NUM \
    --writes=$NUM \
    --cache_size=6442450944 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0  > ${STAT_FILE}.fillseq

# measure overwrite performance
./db_bench \
    --benchmarks=overwrite \
    --db=$DATA_DIR \
    --use_existing_db=1 \
    --bloom_bits=10 \
    --num=$NUM \
    --writes=$((NUM / 10)) \
    --cache_size=6442450944 \
    --cache_numshardbits=6  \
    --table_cache_numshardbits=4 \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=8 > ${STAT_FILE}.overwrite

# fill up the db for readrandom benchmark (1GB total size)
./db_bench \
    --benchmarks=fillseq \
    --db=$DATA_DIR \
    --use_existing_db=0 \
    --bloom_bits=10 \
    --num=$NUM \
    --writes=$NUM \
    --cache_size=6442450944 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=1 > /dev/null

# measure readrandom with 6GB block cache
./db_bench \
    --benchmarks=readrandom \
    --db=$DATA_DIR \
    --use_existing_db=1 \
    --bloom_bits=10 \
    --num=$NUM \
    --reads=$((NUM / 5)) \
    --cache_size=6442450944 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=16 > ${STAT_FILE}.readrandom

# measure readrandom with 6GB block cache and tailing iterator
./db_bench \
    --benchmarks=readrandom \
    --db=$DATA_DIR \
    --use_existing_db=1 \
    --bloom_bits=10 \
    --num=$NUM \
    --reads=$((NUM / 5)) \
    --cache_size=6442450944 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --open_files=55000 \
    --use_tailing_iterator=1 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=16 > ${STAT_FILE}.readrandomtailing

# measure readrandom with 100MB block cache
./db_bench \
    --benchmarks=readrandom \
    --db=$DATA_DIR \
    --use_existing_db=1 \
    --bloom_bits=10 \
    --num=$NUM \
    --reads=$((NUM / 5)) \
    --cache_size=104857600 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=16 > ${STAT_FILE}.readrandomsmallblockcache

# measure readrandom with 8k data in memtable
./db_bench \
    --benchmarks=overwrite,readrandom \
    --db=$DATA_DIR \
    --use_existing_db=1 \
    --bloom_bits=10 \
    --num=$NUM \
    --reads=$((NUM / 5)) \
    --writes=512 \
    --cache_size=6442450944 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --write_buffer_size=1000000000 \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=16 > ${STAT_FILE}.readrandom_mem_sst


# fill up the db for readrandom benchmark with filluniquerandom (1GB total size)
./db_bench \
    --benchmarks=filluniquerandom \
    --db=$DATA_DIR \
    --use_existing_db=0 \
    --bloom_bits=10 \
    --num=$((NUM / 4)) \
    --writes=$((NUM / 4)) \
    --cache_size=6442450944 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=1 > /dev/null

# dummy test just to compact the data
./db_bench \
    --benchmarks=readrandom \
    --db=$DATA_DIR \
    --use_existing_db=1 \
    --bloom_bits=10 \
    --num=$((NUM / 1000)) \
    --reads=$((NUM / 1000)) \
    --cache_size=6442450944 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=16 > /dev/null

# measure readrandom after load with filluniquerandom with 6GB block cache
./db_bench \
    --benchmarks=readrandom \
    --db=$DATA_DIR \
    --use_existing_db=1 \
    --bloom_bits=10 \
    --num=$((NUM / 4)) \
    --reads=$((NUM / 4)) \
    --cache_size=6442450944 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --open_files=55000 \
    --disable_auto_compactions=1 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=16 > ${STAT_FILE}.readrandom_filluniquerandom

# measure readwhilewriting after load with filluniquerandom with 6GB block cache
./db_bench \
    --benchmarks=readwhilewriting \
    --db=$DATA_DIR \
    --use_existing_db=1 \
    --bloom_bits=10 \
    --num=$((NUM / 4)) \
    --reads=$((NUM / 4)) \
    --writes_per_second=1000 \
    --write_buffer_size=100000000 \
    --cache_size=6442450944 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=16 > ${STAT_FILE}.readwhilewriting

# measure memtable performance -- none of the data gets flushed to disk
./db_bench \
    --benchmarks=fillrandom,readrandom, \
    --db=$DATA_DIR \
    --use_existing_db=0 \
    --num=$((NUM / 10)) \
    --reads=$NUM \
    --cache_size=6442450944 \
    --cache_numshardbits=6 \
    --table_cache_numshardbits=4 \
    --write_buffer_size=1000000000 \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --value_size=10 \
    --threads=16 > ${STAT_FILE}.memtablefillreadrandom

common_in_mem_args="--db=/dev/shm/rocksdb \
    --num_levels=6 \
    --key_size=20 \
    --prefix_size=12 \
    --keys_per_prefix=10 \
    --value_size=100 \
    --compression_type=none \
    --compression_ratio=1 \
    --hard_rate_limit=2 \
    --write_buffer_size=134217728 \
    --max_write_buffer_number=4 \
    --level0_file_num_compaction_trigger=8 \
    --level0_slowdown_writes_trigger=16 \
    --level0_stop_writes_trigger=24 \
    --target_file_size_base=134217728 \
    --max_bytes_for_level_base=1073741824 \
    --disable_wal=0 \
    --wal_dir=/dev/shm/rocksdb \
    --sync=0 \
    --disable_data_sync=1 \
    --verify_checksum=1 \
    --delete_obsolete_files_period_micros=314572800 \
    --max_grandparent_overlap_factor=10 \
    --use_plain_table=1 \
    --open_files=-1 \
    --mmap_read=1 \
    --mmap_write=0 \
    --memtablerep=prefix_hash \
    --bloom_bits=10 \
    --bloom_locality=1 \
    --perf_level=0"

# prepare a in-memory DB with 50M keys, total DB size is ~6G
./db_bench \
    $common_in_mem_args \
    --statistics=0 \
    --max_background_compactions=16 \
    --max_background_flushes=16 \
    --benchmarks=filluniquerandom \
    --use_existing_db=0 \
    --num=52428800 \
    --threads=1 > /dev/null

# Readwhilewriting
./db_bench \
    $common_in_mem_args \
    --statistics=1 \
    --max_background_compactions=4 \
    --max_background_flushes=0 \
    --benchmarks=readwhilewriting\
    --use_existing_db=1 \
    --duration=600 \
    --threads=32 \
    --writes_per_second=81920 > ${STAT_FILE}.readwhilewriting_in_ram

# Seekrandomwhilewriting
./db_bench \
    $common_in_mem_args \
    --statistics=1 \
    --max_background_compactions=4 \
    --max_background_flushes=0 \
    --benchmarks=seekrandomwhilewriting \
    --use_existing_db=1 \
    --use_tailing_iterator=1 \
    --duration=600 \
    --threads=32 \
    --writes_per_second=81920 > ${STAT_FILE}.seekwhilewriting_in_ram

# measure fillseq with bunch of column families
./db_bench \
    --benchmarks=fillseq \
    --num_column_families=500 \
    --write_buffer_size=1048576 \
    --db=$DATA_DIR \
    --use_existing_db=0 \
    --num=$NUM \
    --writes=$NUM \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0  > ${STAT_FILE}.fillseq_lots_column_families

# measure overwrite performance with bunch of column families
./db_bench \
    --benchmarks=overwrite \
    --num_column_families=500 \
    --write_buffer_size=1048576 \
    --db=$DATA_DIR \
    --use_existing_db=1 \
    --num=$NUM \
    --writes=$((NUM / 10)) \
    --open_files=55000 \
    --statistics=1 \
    --histogram=1 \
    --disable_data_sync=1 \
    --disable_wal=1 \
    --sync=0 \
    --threads=8 > ${STAT_FILE}.overwrite_lots_column_families

# send data to ods
function send_to_ods {
  key="$1"
  value="$2"

  if [ -z $JENKINS_HOME ]; then
    # running on devbox, just print out the values
    echo $1 $2
    return
  fi

  if [ -z "$value" ];then
    echo >&2 "ERROR: Key $key doesn't have a value."
    return
  fi
  curl -s "https://www.intern.facebook.com/intern/agent/ods_set.php?entity=rocksdb_build$git_br&key=$key&value=$value" \
    --connect-timeout 60
}

function send_benchmark_to_ods {
  bench="$1"
  bench_key="$2"
  file="$3"

  QPS=$(grep $bench $file | awk '{print $5}')
  P50_MICROS=$(grep $bench $file -A 6 | grep "Percentiles" | awk '{print $3}' )
  P75_MICROS=$(grep $bench $file -A 6 | grep "Percentiles" | awk '{print $5}' )
  P99_MICROS=$(grep $bench $file -A 6 | grep "Percentiles" | awk '{print $7}' )

  send_to_ods rocksdb.build.$bench_key.qps $QPS
  send_to_ods rocksdb.build.$bench_key.p50_micros $P50_MICROS
  send_to_ods rocksdb.build.$bench_key.p75_micros $P75_MICROS
  send_to_ods rocksdb.build.$bench_key.p99_micros $P99_MICROS
}

send_benchmark_to_ods overwrite overwrite $STAT_FILE.overwrite
send_benchmark_to_ods fillseq fillseq $STAT_FILE.fillseq
send_benchmark_to_ods readrandom readrandom $STAT_FILE.readrandom
send_benchmark_to_ods readrandom readrandom_tailing $STAT_FILE.readrandomtailing
send_benchmark_to_ods readrandom readrandom_smallblockcache $STAT_FILE.readrandomsmallblockcache
send_benchmark_to_ods readrandom readrandom_memtable_sst $STAT_FILE.readrandom_mem_sst
send_benchmark_to_ods readrandom readrandom_fillunique_random $STAT_FILE.readrandom_filluniquerandom
send_benchmark_to_ods fillrandom memtablefillrandom $STAT_FILE.memtablefillreadrandom
send_benchmark_to_ods readrandom memtablereadrandom $STAT_FILE.memtablefillreadrandom
send_benchmark_to_ods readwhilewriting readwhilewriting $STAT_FILE.readwhilewriting
send_benchmark_to_ods readwhilewriting readwhilewriting_in_ram ${STAT_FILE}.readwhilewriting_in_ram
send_benchmark_to_ods seekrandomwhilewriting seekwhilewriting_in_ram ${STAT_FILE}.seekwhilewriting_in_ram
send_benchmark_to_ods fillseq fillseq_lots_column_families ${STAT_FILE}.fillseq_lots_column_families
send_benchmark_to_ods overwrite overwrite_lots_column_families ${STAT_FILE}.overwrite_lots_column_families
