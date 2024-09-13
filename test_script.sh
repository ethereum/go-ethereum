#!/bin/bash

# Run the test and capture the output
test_output=$(go test -v -timeout 30m github.com/ethereum/go-ethereum/core)

# Initialize variables to store test statuses
test_names=""
test_statuses=""

# Process the output line by line
while IFS= read -r line; do
    if [[ $line =~ ^===\ (RUN|CONT)\ (.+) ]]; then
        test_name=${BASH_REMATCH[2]}
        if [[ $test_names != *"$test_name"* ]]; then
            test_names="$test_names$test_name "
            test_statuses="${test_statuses}STARTED "
        fi
    elif [[ $line =~ ^---\ (PASS|FAIL|SKIP):\ (.+)\ \(.+\)$ ]]; then
        status=${BASH_REMATCH[1]}
        test_name=${BASH_REMATCH[2]}
        index=0
        for name in $test_names; do
            if [ "$name" = "$test_name" ]; then
                test_statuses=$(echo "$test_statuses" | awk -v i=$index -v s=$status '{split($0,a," "); a[i+1]=s; for(j=1;j<=NF;j++) printf "%s%s", a[j], (j==NF?"\n":" ")}')
                break
            fi
            ((index++))
        done
    fi
done <<< "$test_output"

# Check for tests that didn't pass
non_passing_tests=""
index=0
for status in $test_statuses; do
    if [[ $status != "PASS" && $status != "SKIP" ]]; then
        test_name=$(echo "$test_names" | cut -d' ' -f$((index+1)))
        non_passing_tests="$non_passing_tests$test_name: $status "
    fi
    ((index++))
done

if [ -z "$non_passing_tests" ]; then
    echo "All tests passed or were skipped successfully."
else
    echo "The following tests did not pass:"
    echo "$non_passing_tests"
fi

# Check for any unexpected FAIL messages
if echo "$test_output" | grep -q "FAIL"; then
    echo "WARNING: 'FAIL' message found in output. This may indicate an issue with the test runner."
fi

# Print the total execution time
execution_time=$(echo "$test_output" | grep "FAIL" | awk '{print $3}')
echo "Total execution time: $execution_time"
