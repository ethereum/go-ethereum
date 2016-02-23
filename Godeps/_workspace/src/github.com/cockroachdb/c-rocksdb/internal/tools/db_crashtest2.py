#! /usr/bin/env python
import os
import re
import sys
import time
import random
import getopt
import logging
import tempfile
import subprocess
import shutil

# This python script runs db_stress multiple times. Some runs with
# kill_random_test that causes rocksdb to crash at various points in code.

def main(argv):
    try:
        opts, args = getopt.getopt(argv, "hsd:t:k:o:b:")
    except getopt.GetoptError:
        print str(getopt.GetoptError)
        print "db_crashtest2.py -d <duration_test> -t <#threads> " \
              "-k <kills with prob 1/k> -o <ops_per_thread> "\
              "-b <write_buffer_size> [-s (simple mode)]\n"
        sys.exit(2)

    # default values, will be overridden by cmdline args
    kill_random_test = 97  # kill with probability 1/97 by default
    duration = 10000  # total time for this script to test db_stress
    threads = 32
    ops_per_thread = 200000
    write_buf_size = 4 * 1024 * 1024
    simple_mode = False
    write_buf_size_set = False

    for opt, arg in opts:
        if opt == '-h':
            print "db_crashtest2.py -d <duration_test> -t <#threads> " \
                  "-k <kills with prob 1/k> -o <ops_per_thread> " \
                  "-b <write_buffer_size> [-s (simple mode)]\n"
            sys.exit()
        elif opt == '-s':
            simple_mode = True
            if not write_buf_size_set:
                write_buf_size = 32 * 1024 * 1024
        elif opt == "-d":
            duration = int(arg)
        elif opt == "-t":
            threads = int(arg)
        elif opt == "-k":
            kill_random_test = int(arg)
        elif opt == "-o":
            ops_per_thread = int(arg)
        elif opt == "-b":
            write_buf_size = int(arg)
            write_buf_size_set = True
        else:
            print "unrecognized option " + str(opt) + "\n"
            print "db_crashtest2.py -d <duration_test> -t <#threads> " \
                  "-k <kills with prob 1/k> -o <ops_per_thread> " \
                  "-b <write_buffer_size>\n"
            sys.exit(2)

    cur_time = time.time()
    exit_time = cur_time + duration
    half_time = cur_time + duration / 2

    print "Running whitebox-crash-test with \ntotal-duration=" + str(duration) \
          + "\nthreads=" + str(threads) + "\nops_per_thread=" \
          + str(ops_per_thread) + "\nwrite_buffer_size=" \
          + str(write_buf_size) + "\n"

    total_check_mode = 4
    check_mode = 0

    test_tmpdir = os.environ.get("TEST_TMPDIR")
    if test_tmpdir is None or test_tmpdir == "":
        dbname = tempfile.mkdtemp(prefix='rocksdb_crashtest2_')
    else:
        dbname = test_tmpdir + "/rocksdb_crashtest2"
        shutil.rmtree(dbname, True)

    while time.time() < exit_time:
        killoption = ""
        if check_mode == 0:
            # run with kill_random_test
            killoption = " --kill_random_test=" + str(kill_random_test)
            # use large ops per thread since we will kill it anyway
            additional_opts = "--ops_per_thread=" + \
                              str(100 * ops_per_thread) + killoption
        elif check_mode == 1:
            # normal run with universal compaction mode
            additional_opts = "--ops_per_thread=" + str(ops_per_thread) + \
                              " --compaction_style=1"
        elif check_mode == 2:
            # normal run with FIFO compaction mode
            # ops_per_thread is divided by 5 because FIFO compaction
            # style is quite a bit slower on reads with lot of files
            additional_opts = "--ops_per_thread=" + str(ops_per_thread / 5) + \
                              " --compaction_style=2"
        else:
            # normal run
            additional_opts = "--ops_per_thread=" + str(ops_per_thread)

        if simple_mode:
            cmd = re.sub('\s+', ' ', """
                ./db_stress
                --column_families=1
                --threads=%s
                --write_buffer_size=%s
                --destroy_db_initially=0
                --reopen=20
                --prefixpercent=0
                --readpercent=50
                --writepercent=35
                --delpercent=5
                --iterpercent=10
                --db=%s
                --max_key=100000000
                --mmap_read=%s
                --block_size=16384
                --cache_size=1048576
                --open_files=500000
                --verify_checksum=1
                --sync=0
                --progress_reports=0
                --disable_wal=0
                --disable_data_sync=1
                --target_file_size_base=16777216
                --target_file_size_multiplier=1
                --max_write_buffer_number=3
                --max_background_compactions=1
                --max_bytes_for_level_base=67108864
                --filter_deletes=%s
                --memtablerep=skip_list
                --prefix_size=0
                 %s
                """ % (threads,
                       write_buf_size,
                       dbname,
                       random.randint(0, 1),
                       random.randint(0, 1),
                       additional_opts))
        else:
            cmd = re.sub('\s+', ' ', """
                ./db_stress
                --test_batches_snapshots=%s
                --threads=%s
                --write_buffer_size=%s
                --destroy_db_initially=0
                --reopen=20
                --readpercent=45
                --prefixpercent=5
                --writepercent=35
                --delpercent=5
                --iterpercent=10
                --db=%s
                --max_key=100000000
                --mmap_read=%s
                --block_size=16384
                --cache_size=1048576
                --open_files=500000
                --verify_checksum=1
                --sync=0
                --progress_reports=0
                --disable_wal=0
                --disable_data_sync=1
                --target_file_size_base=2097152
                --target_file_size_multiplier=2
                --max_write_buffer_number=3
                --max_background_compactions=20
                --max_bytes_for_level_base=10485760
                --filter_deletes=%s
                --memtablerep=prefix_hash
                --prefix_size=7
                %s
                """ % (random.randint(0, 1),
                       threads,
                       write_buf_size,
                       dbname,
                       random.randint(0, 1),
                       random.randint(0, 1),
                       additional_opts))

        print "Running:" + cmd + "\n"

        popen = subprocess.Popen([cmd], stdout=subprocess.PIPE,
                                 stderr=subprocess.STDOUT,
                                 shell=True)
        stdoutdata, stderrdata = popen.communicate()
        retncode = popen.returncode
        msg = ("check_mode={0}, kill option={1}, exitcode={2}\n".format(
               check_mode, killoption, retncode))
        print msg
        print stdoutdata

        expected = False
        if (killoption == '') and (retncode == 0):
            # we expect zero retncode if no kill option
            expected = True
        elif killoption != '' and retncode < 0:
            # we expect negative retncode if kill option was given
            expected = True

        if not expected:
            print "TEST FAILED. See kill option and exit code above!!!\n"
            sys.exit(1)

        stdoutdata = stdoutdata.lower()
        errorcount = (stdoutdata.count('error') -
                      stdoutdata.count('got errors 0 times'))
        print "#times error occurred in output is " + str(errorcount) + "\n"

        if (errorcount > 0):
            print "TEST FAILED. Output has 'error'!!!\n"
            sys.exit(2)
        if (stdoutdata.find('fail') >= 0):
            print "TEST FAILED. Output has 'fail'!!!\n"
            sys.exit(2)

        # First half of the duration, keep doing kill test. For the next half,
        # try different modes.
        if time.time() > half_time:
            # we need to clean up after ourselves -- only do this on test
            # success
            shutil.rmtree(dbname, True)
            check_mode = (check_mode + 1) % total_check_mode

        time.sleep(1)  # time to stabilize after a kill

if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
