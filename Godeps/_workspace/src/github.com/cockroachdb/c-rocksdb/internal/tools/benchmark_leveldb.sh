#!/bin/bash
# REQUIRE: db_bench binary exists in the current directory
#
# This should be used with the LevelDB fork listed here to use additional test options.
# For more details on the changes see the blog post listed below.
#   https://github.com/mdcallag/leveldb-1
#   http://smalldatum.blogspot.com/2015/04/comparing-leveldb-and-rocksdb-take-2.html

if [ $# -ne 1 ]; then
  echo -n "./benchmark.sh [fillseq/overwrite/readrandom/readwhilewriting]"
  exit 0
fi

# size constants
K=1024
M=$((1024 * K))
G=$((1024 * M))

if [ -z $DB_DIR ]; then
  echo "DB_DIR is not defined"
  exit 0
fi

output_dir=${OUTPUT_DIR:-/tmp/}
if [ ! -d $output_dir ]; then
  mkdir -p $output_dir
fi

# all multithreaded tests run with sync=1 unless
# $DB_BENCH_NO_SYNC is defined
syncval="1"
if [ ! -z $DB_BENCH_NO_SYNC ]; then
  echo "Turning sync off for all multithreaded tests"
  syncval="0";
fi

num_threads=${NUM_THREADS:-16}
# Only for *whilewriting, *whilemerging
writes_per_second=${WRITES_PER_SECOND:-$((10 * K))}
cache_size=${CACHE_SIZE:-$((1 * G))}

num_keys=${NUM_KEYS:-$((1 * G))}
key_size=20
value_size=${VALUE_SIZE:-400}
block_size=${BLOCK_SIZE:-4096}

const_params="
  --db=$DB_DIR \
  \
  --num=$num_keys \
  --value_size=$value_size \
  --cache_size=$cache_size \
  --compression_ratio=0.5 \
  \
  --write_buffer_size=$((2 * M)) \
  \
  --histogram=1 \
  \
  --bloom_bits=10 \
  --open_files=$((20 * K))"

params_w="$const_params "

function summarize_result {
  test_out=$1
  test_name=$2
  bench_name=$3
  nthr=$4

  usecs_op=$( grep ^${bench_name} $test_out | awk '{ printf "%.1f", $3 }' )
  mb_sec=$( grep ^${bench_name} $test_out | awk '{ printf "%.1f", $5 }' )
  ops=$( grep "^Count:" $test_out | awk '{ print $2 }' )
  ops_sec=$( echo "scale=0; (1000000.0 * $nthr) / $usecs_op" | bc )
  avg=$( grep "^Count:" $test_out | awk '{ printf "%.1f", $4 }' )
  p50=$( grep "^Min:" $test_out | awk '{ printf "%.1f", $4 }' )
  echo -e "$ops_sec\t$mb_sec\t$usecs_op\t$avg\t$p50\t$test_name" \
    >> $output_dir/report.txt
}

function run_fillseq {
  # This runs with a vector memtable and the WAL disabled to load faster. It is still crash safe and the
  # client can discover where to restart a load after a crash. I think this is a good way to load.
  echo "Loading $num_keys keys sequentially"
  cmd="./db_bench --benchmarks=fillseq \
       --use_existing_db=0 \
       --sync=0 \
       $params_w \
       --threads=1 \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/benchmark_fillseq.v${value_size}.log"
  echo $cmd | tee $output_dir/benchmark_fillseq.v${value_size}.log
  eval $cmd
  summarize_result $output_dir/benchmark_fillseq.v${value_size}.log fillseq.v${value_size} fillseq 1
}

function run_change {
  operation=$1
  echo "Do $num_keys random $operation"
  out_name="benchmark_${operation}.t${num_threads}.s${syncval}.log"
  cmd="./db_bench --benchmarks=$operation \
       --use_existing_db=1 \
       --sync=$syncval \
       $params_w \
       --threads=$num_threads \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/${out_name}"
  echo $cmd | tee $output_dir/${out_name}
  eval $cmd
  summarize_result $output_dir/${out_name} ${operation}.t${num_threads}.s${syncval} $operation $num_threads
}

function run_readrandom {
  echo "Reading $num_keys random keys"
  out_name="benchmark_readrandom.t${num_threads}.log"
  cmd="./db_bench --benchmarks=readrandom \
       --use_existing_db=1 \
       $params_w \
       --threads=$num_threads \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/${out_name}"
  echo $cmd | tee $output_dir/${out_name}
  eval $cmd
  summarize_result $output_dir/${out_name} readrandom.t${num_threads} readrandom $num_threads
}

function run_readwhile {
  operation=$1
  echo "Reading $num_keys random keys while $operation"
  out_name="benchmark_readwhile${operation}.t${num_threads}.log"
  cmd="./db_bench --benchmarks=readwhile${operation} \
       --use_existing_db=1 \
       --sync=$syncval \
       $params_w \
       --threads=$num_threads \
       --writes_per_second=$writes_per_second \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/${out_name}"
  echo $cmd | tee $output_dir/${out_name}
  eval $cmd
  summarize_result $output_dir/${out_name} readwhile${operation}.t${num_threads} readwhile${operation} $num_threads
}

function now() {
  echo `date +"%s"`
}

report="$output_dir/report.txt"
schedule="$output_dir/schedule.txt"

echo "===== Benchmark ====="

# Run!!!
IFS=',' read -a jobs <<< $1
for job in ${jobs[@]}; do

  if [ $job != debug ]; then
    echo "Start $job at `date`" | tee -a $schedule
  fi

  start=$(now)
  if [ $job = fillseq ]; then
    run_fillseq
  elif [ $job = overwrite ]; then
    run_change overwrite
  elif [ $job = readrandom ]; then
    run_readrandom
  elif [ $job = readwhilewriting ]; then
    run_readwhile writing
  elif [ $job = debug ]; then
    num_keys=1000; # debug
    echo "Setting num_keys to $num_keys"
  else
    echo "unknown job $job"
    exit
  fi
  end=$(now)

  if [ $job != debug ]; then
    echo "Complete $job in $((end-start)) seconds" | tee -a $schedule
  fi

  echo -e "ops/sec\tmb/sec\tusec/op\tavg\tp50\tTest"
  tail -1 $output_dir/report.txt

done
