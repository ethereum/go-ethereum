TMP_DIR="/tmp/rocksdb-sanity-test"

if [ "$#" -lt 2 ]; then
  echo "usage: ./auto_sanity_test.sh [new_commit] [old_commit]"
  echo "Missing either [new_commit] or [old_commit], perform sanity check with the latest and 10th latest commits."
  recent_commits=`git log | grep -e "^commit [a-z0-9]\+$"| head -n10 | sed -e 's/commit //g'`
  commit_new=`echo "$recent_commits" | head -n1`
  commit_old=`echo "$recent_commits" | tail -n1`
  echo "the most recent commits are:"
  echo "$recent_commits"
else
  commit_new=$1
  commit_old=$2
fi

if [ ! -d $TMP_DIR ]; then
  mkdir $TMP_DIR
fi
dir_new="${TMP_DIR}/${commit_new}"
dir_old="${TMP_DIR}/${commit_old}"

function makestuff() {
  echo "make clean"
  make clean > /dev/null
  echo "make db_sanity_test -j32"
  make db_sanity_test -j32 > /dev/null
  if [ $? -ne 0 ]; then
    echo "[ERROR] Failed to perform 'make db_sanity_test'"
    exit 1
  fi
}

rm -r -f $dir_new
rm -r -f $dir_old

echo "Running db sanity check with commits $commit_new and $commit_old."

echo "============================================================="
echo "Making build $commit_new"
git checkout $commit_new
if [ $? -ne 0 ]; then
  echo "[ERROR] Can't checkout $commit_new"
  exit 1
fi
makestuff
mv db_sanity_test new_db_sanity_test
echo "Creating db based on the new commit --- $commit_new"
./new_db_sanity_test $dir_new create
cp ./tools/db_sanity_test.cc $dir_new
cp ./tools/auto_sanity_test.sh $dir_new

echo "============================================================="
echo "Making build $commit_old"
git checkout $commit_old
if [ $? -ne 0 ]; then
  echo "[ERROR] Can't checkout $commit_old"
  exit 1
fi
cp -f $dir_new/db_sanity_test.cc ./tools/.
cp -f $dir_new/auto_sanity_test.sh ./tools/.
makestuff
mv db_sanity_test old_db_sanity_test
echo "Creating db based on the old commit --- $commit_old"
./old_db_sanity_test $dir_old create

echo "============================================================="
echo "[Backward Compability Check]"
echo "Verifying old db $dir_old using the new commit --- $commit_new"
./new_db_sanity_test $dir_old verify
if [ $? -ne 0 ]; then
  echo "[ERROR] Backward Compability Check fails:"
  echo "    Verification of $dir_old using commit $commit_new failed."
  exit 2
fi

echo "============================================================="
echo "[Forward Compatibility Check]"
echo "Verifying new db $dir_new using the old commit --- $commit_old"
./old_db_sanity_test $dir_new verify
if [ $? -ne 0 ]; then
  echo "[ERROR] Forward Compability Check fails:"
  echo "    $dir_new using commit $commit_old failed."
  exit 2
fi

rm old_db_sanity_test
rm new_db_sanity_test
rm -rf $dir_new
rm -rf $dir_old

echo "Auto sanity test passed!"
