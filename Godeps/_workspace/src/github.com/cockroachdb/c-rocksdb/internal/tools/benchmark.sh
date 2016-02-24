#!/bin/bash
# REQUIRE: db_bench binary exists in the current directory

if [ $# -ne 1 ]; then
  echo -n "./benchmark.sh [bulkload/fillseq/overwrite/filluniquerandom/"
  echo    "readrandom/readwhilewriting/readwhilemerging/updaterandom/"
  echo    "mergerandom/randomtransaction]"
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

if [ -z $WAL_DIR ]; then
  echo "WAL_DIR is not defined"
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
# Only for tests that do range scans
num_nexts_per_seek=${NUM_NEXTS_PER_SEEK:-10}
cache_size=${CACHE_SIZE:-$((1 * G))}
duration=${DURATION:-0}

num_keys=${NUM_KEYS:-$((1 * G))}
key_size=20
value_size=${VALUE_SIZE:-400}
block_size=${BLOCK_SIZE:-8192}

const_params="
  --db=$DB_DIR \
  --wal_dir=$WAL_DIR \
  --disable_data_sync=0 \
  \
  --num=$num_keys \
  --num_levels=6 \
  --key_size=$key_size \
  --value_size=$value_size \
  --block_size=$block_size \
  --cache_size=$cache_size \
  --cache_numshardbits=6 \
  --compression_type=snappy \
  --min_level_to_compress=3 \
  --compression_ratio=0.5 \
  --level_compaction_dynamic_level_bytes=true \
  --bytes_per_sync=$((8 * M)) \
  --cache_index_and_filter_blocks=0 \
  \
  --hard_rate_limit=3 \
  --rate_limit_delay_max_milliseconds=1000000 \
  --write_buffer_size=$((128 * M)) \
  --max_write_buffer_number=8 \
  --target_file_size_base=$((128 * M)) \
  --max_bytes_for_level_base=$((1 * G)) \
  \
  --verify_checksum=1 \
  --delete_obsolete_files_period_micros=$((60 * M)) \
  --max_grandparent_overlap_factor=8 \
  --max_bytes_for_level_multiplier=8 \
  \
  --statistics=0 \
  --stats_per_interval=1 \
  --stats_interval_seconds=60 \
  --histogram=1 \
  \
  --memtablerep=skip_list \
  --bloom_bits=10 \
  --open_files=-1"

l0_config="
  --level0_file_num_compaction_trigger=4 \
  --level0_slowdown_writes_trigger=12 \
  --level0_stop_writes_trigger=20"

if [ $duration -gt 0 ]; then
  const_params="$const_params --duration=$duration"
fi

params_w="$const_params $l0_config --max_background_compactions=16 --max_background_flushes=7"
params_bulkload="$const_params --max_background_compactions=16 --max_background_flushes=7 \
                 --level0_file_num_compaction_trigger=$((10 * M)) \
                 --level0_slowdown_writes_trigger=$((10 * M)) \
                 --level0_stop_writes_trigger=$((10 * M))"

function summarize_result {
  test_out=$1
  test_name=$2
  bench_name=$3

  uptime=$( grep ^Uptime\(secs $test_out | tail -1 | awk '{ printf "%.0f", $2 }' )
  stall_time=$( grep "^Cumulative stall" $test_out | tail -1  | awk '{  print $3 }' )
  stall_pct=$( grep "^Cumulative stall" $test_out| tail -1  | awk '{  print $5 }' )
  ops_sec=$( grep ^${bench_name} $test_out | awk '{ print $5 }' )
  mb_sec=$( grep ^${bench_name} $test_out | awk '{ print $7 }' )
  lo_wgb=$( grep "^  L0" $test_out | tail -1 | awk '{ print $8 }' )
  sum_wgb=$( grep "^ Sum" $test_out | tail -1 | awk '{ print $8 }' )
  sum_size=$( grep "^ Sum" $test_out | tail -1 | awk '{ printf "%.1f", $3 / 1024.0 }' )
  wamp=$( echo "scale=1; $sum_wgb / $lo_wgb" | bc )
  wmb_ps=$( echo "scale=1; ( $sum_wgb * 1024.0 ) / $uptime" | bc )
  usecs_op=$( grep ^${bench_name} $test_out | awk '{ printf "%.1f", $3 }' )
  p50=$( grep "^Percentiles:" $test_out | tail -1 | awk '{ printf "%.1f", $3 }' )
  p75=$( grep "^Percentiles:" $test_out | tail -1 | awk '{ printf "%.1f", $5 }' )
  p99=$( grep "^Percentiles:" $test_out | tail -1 | awk '{ printf "%.0f", $7 }' )
  p999=$( grep "^Percentiles:" $test_out | tail -1 | awk '{ printf "%.0f", $9 }' )
  p9999=$( grep "^Percentiles:" $test_out | tail -1 | awk '{ printf "%.0f", $11 }' )
  echo -e "$ops_sec\t$mb_sec\t$sum_size\t$lo_wgb\t$sum_wgb\t$wamp\t$wmb_ps\t$usecs_op\t$p50\t$p75\t$p99\t$p999\t$p9999\t$uptime\t$stall_time\t$stall_pct\t$test_name" \
    >> $output_dir/report.txt
}

function run_bulkload {
  # This runs with a vector memtable and the WAL disabled to load faster. It is still crash safe and the
  # client can discover where to restart a load after a crash. I think this is a good way to load.
  echo "Bulk loading $num_keys random keys"
  cmd="./db_bench --benchmarks=fillrandom \
       --use_existing_db=0 \
       --disable_auto_compactions=1 \
       --sync=0 \
       $params_bulkload \
       --threads=1 \
       --memtablerep=vector \
       --disable_wal=1 \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/benchmark_bulkload_fillrandom.log"
  echo $cmd | tee $output_dir/benchmark_bulkload_fillrandom.log
  eval $cmd
  summarize_result $output_dir/benchmark_bulkload_fillrandom.log bulkload fillrandom
  echo "Compacting..."
  cmd="./db_bench --benchmarks=compact \
       --use_existing_db=1 \
       --disable_auto_compactions=1 \
       --sync=0 \
       $params_w \
       --threads=1 \
       2>&1 | tee -a $output_dir/benchmark_bulkload_compact.log"
  echo $cmd | tee $output_dir/benchmark_bulkload_compact.log
  eval $cmd
}

function run_fillseq {
  # This runs with a vector memtable and the WAL disabled to load faster. It is still crash safe and the
  # client can discover where to restart a load after a crash. I think this is a good way to load.
  echo "Loading $num_keys keys sequentially"
  cmd="./db_bench --benchmarks=fillseq \
       --use_existing_db=0 \
       --sync=0 \
       $params_w \
       --min_level_to_compress=0 \
       --threads=1 \
       --memtablerep=vector \
       --disable_wal=1 \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/benchmark_fillseq.v${value_size}.log"
  echo $cmd | tee $output_dir/benchmark_fillseq.v${value_size}.log
  eval $cmd
  summarize_result $output_dir/benchmark_fillseq.v${value_size}.log fillseq.v${value_size} fillseq
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
       --merge_operator=\"put\" \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/${out_name}"
  echo $cmd | tee $output_dir/${out_name}
  eval $cmd
  summarize_result $output_dir/${out_name} ${operation}.t${num_threads}.s${syncval} $operation
}

function run_filluniquerandom {
  echo "Loading $num_keys unique keys randomly"
  cmd="./db_bench --benchmarks=filluniquerandom \
       --use_existing_db=0 \
       --sync=0 \
       $params_w \
       --threads=1 \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/benchmark_filluniquerandom.log"
  echo $cmd | tee $output_dir/benchmark_filluniquerandom.log
  eval $cmd
  summarize_result $output_dir/benchmark_filluniquerandom.log filluniquerandom filluniquerandom
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
  summarize_result $output_dir/${out_name} readrandom.t${num_threads} readrandom
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
       --merge_operator=\"put\" \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/${out_name}"
  echo $cmd | tee $output_dir/${out_name}
  eval $cmd
  summarize_result $output_dir/${out_name} readwhile${operation}.t${num_threads} readwhile${operation}
}

function run_rangewhile {
  operation=$1
  full_name=$2
  reverse_arg=$3
  out_name="benchmark_${full_name}.t${num_threads}.log"
  echo "Range scan $num_keys random keys while ${operation} for reverse_iter=${reverse_arg}"
  cmd="./db_bench --benchmarks=seekrandomwhile${operation} \
       --use_existing_db=1 \
       --sync=$syncval \
       $params_w \
       --threads=$num_threads \
       --writes_per_second=$writes_per_second \
       --merge_operator=\"put\" \
       --seek_nexts=$num_nexts_per_seek \
       --reverse_iterator=$reverse_arg \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/${out_name}"
  echo $cmd | tee $output_dir/${out_name}
  eval $cmd
  summarize_result $output_dir/${out_name} ${full_name}.t${num_threads} seekrandomwhile${operation}
}

function run_range {
  full_name=$1
  reverse_arg=$2
  out_name="benchmark_${full_name}.t${num_threads}.log"
  echo "Range scan $num_keys random keys for reverse_iter=${reverse_arg}"
  cmd="./db_bench --benchmarks=seekrandom \
       --use_existing_db=1 \
       $params_w \
       --threads=$num_threads \
       --seek_nexts=$num_nexts_per_seek \
       --reverse_iterator=$reverse_arg \
       --seed=$( date +%s ) \
       2>&1 | tee -a $output_dir/${out_name}"
  echo $cmd | tee $output_dir/${out_name}
  eval $cmd
  summarize_result $output_dir/${out_name} ${full_name}.t${num_threads} seekrandom
}

function run_randomtransaction {
  echo "..."
  cmd="./db_bench $params_r --benchmarks=randomtransaction \
       --num=$num_keys \
       --transaction_db \
       --threads=5 \
       --transaction_sets=5 \
       2>&1 | tee $output_dir/benchmark_randomtransaction.log"
  echo $cmd | tee $output_dir/benchmark_rangescanwhilewriting.log
  eval $cmd
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
  if [ $job = bulkload ]; then
    run_bulkload
  elif [ $job = fillseq ]; then
    run_fillseq
  elif [ $job = overwrite ]; then
    run_change overwrite
  elif [ $job = updaterandom ]; then
    run_change updaterandom
  elif [ $job = mergerandom ]; then
    run_change mergerandom
  elif [ $job = filluniquerandom ]; then
    run_filluniquerandom
  elif [ $job = readrandom ]; then
    run_readrandom
  elif [ $job = fwdrange ]; then
    run_range $job false
  elif [ $job = revrange ]; then
    run_range $job true
  elif [ $job = readwhilewriting ]; then
    run_readwhile writing
  elif [ $job = readwhilemerging ]; then
    run_readwhile merging
  elif [ $job = fwdrangewhilewriting ]; then
    run_rangewhile writing $job false
  elif [ $job = revrangewhilewriting ]; then
    run_rangewhile writing $job true
  elif [ $job = fwdrangewhilemerging ]; then
    run_rangewhile merging $job false
  elif [ $job = revrangewhilemerging ]; then
    run_rangewhile merging $job true
  elif [ $job = randomtransaction ]; then
    run_randomtransaction
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

  echo -e "ops/sec\tmb/sec\tSize-GB\tL0_MB\tSum_GB\tW-Amp\tW-MB/s\tusec/op\tp50\tp75\tp99\tp99.9\tp99.99\tUptime\tStall-time\tStall%\tTest"
  tail -1 $output_dir/report.txt

done
