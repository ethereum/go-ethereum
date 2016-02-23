TESTDIR=`mktemp -d /tmp/rocksdb-dump-test.XXXXX`
DUMPFILE="tools/sample-dump.dmp"

# Verify that the sample dump file is undumpable and then redumpable.
./rocksdb_undump $DUMPFILE $TESTDIR/db
./rocksdb_dump --anonymous $TESTDIR/db $TESTDIR/dump
cmp $DUMPFILE $TESTDIR/dump
