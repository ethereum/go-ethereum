#!/bin/bash

# Exit on error.
set -e

if [ -n "$USE_CLANG" ]; then
  echo "Error: Coverage test is supported only for gcc."
  exit 1
fi

ROOT=".."
# Fetch right version of gcov
if [ -d /mnt/gvfs/third-party -a -z "$CXX" ]; then
  source $ROOT/build_tools/fbcode_config.sh
  GCOV=$GCC_BASE/bin/gcov
else
  GCOV=$(which gcov)
fi

COVERAGE_DIR="$PWD/COVERAGE_REPORT"
mkdir -p $COVERAGE_DIR

# Find all gcno files to generate the coverage report

GCNO_FILES=`find $ROOT -name "*.gcno"`
$GCOV --preserve-paths --relative-only --no-output $GCNO_FILES 2>/dev/null |
  # Parse the raw gcov report to more human readable form.
  python $ROOT/coverage/parse_gcov_output.py |
  # Write the output to both stdout and report file.
  tee $COVERAGE_DIR/coverage_report_all.txt &&
echo -e "Generated coverage report for all files: $COVERAGE_DIR/coverage_report_all.txt\n"

# TODO: we also need to get the files of the latest commits.
# Get the most recently committed files.
LATEST_FILES=`
  git show --pretty="format:" --name-only HEAD |
  grep -v "^$" |
  paste -s -d,`
RECENT_REPORT=$COVERAGE_DIR/coverage_report_recent.txt

echo -e "Recently updated files: $LATEST_FILES\n" > $RECENT_REPORT
$GCOV --preserve-paths --relative-only --no-output $GCNO_FILES 2>/dev/null |
  python $ROOT/coverage/parse_gcov_output.py -interested-files $LATEST_FILES |
  tee -a $RECENT_REPORT &&
echo -e "Generated coverage report for recently updated files: $RECENT_REPORT\n"

# Unless otherwise specified, we'll not generate html report by default
if [ -z "$HTML" ]; then
  exit 0
fi

# Generate the html report. If we cannot find lcov in this machine, we'll simply
# skip this step.
echo "Generating the html coverage report..."

LCOV=$(which lcov || true 2>/dev/null)
if [ -z $LCOV ]
then
  echo "Skip: Cannot find lcov to generate the html report."
  exit 0
fi

LCOV_VERSION=$(lcov -v | grep 1.1 || true)
if [ $LCOV_VERSION ]
then
  echo "Not supported lcov version. Expect lcov 1.1."
  exit 0
fi

(cd $ROOT; lcov --no-external \
     --capture  \
     --directory $PWD \
     --gcov-tool $GCOV \
     --output-file $COVERAGE_DIR/coverage.info)

genhtml $COVERAGE_DIR/coverage.info -o $COVERAGE_DIR

echo "HTML Coverage report is generated in $COVERAGE_DIR"
