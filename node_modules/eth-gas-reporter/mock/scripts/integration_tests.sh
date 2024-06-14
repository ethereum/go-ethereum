#!/usr/bin/env bash

# --------------------------------- Truffle --------------------------------------------------------

# Basic, no reporter options specified
test_truffle_v5_basic() {
  echo "-----------------------------------------------------------"
  echo "> Visual inspection integration test (Truffle - no options)"
  echo "> YOU MUST LOOK AT THIS TEST TO DETECT FAILURE"
  echo "-----------------------------------------------------------"

  npx truffle test --network development "$@"

}

# With options
test_truffle_v5_with_options() {
  echo "--------------------------------------------------------------------"
  echo "> Visual inspection integration test (Truffle - reporter options)"
  echo "> YOU MUST LOOK AT THIS TEST TO DETECT FAILURE"
  echo "--------------------------------------------------------------------"

  # Swap out no-options truffle.js for one with config
  cp ./truffle.js ./safe_truffle.js
  cp ./config-template.js ./truffle.js

  # Test
  DEBUG_CODECHECKS_TABLE=true npx truffle test --network development "$@"

  if [ "$CI" = "true" ]; then
    npx codechecks
  fi

  # Swap in truffle.js
  cp ./safe_truffle.js ./truffle.js
  rm ./safe_truffle.js
}

# --------------------------------- Buidler (V5 plugin) --------------------------------------------

# Basic, no reporter options specified
# Swaps out TestMetacoin.sol because it throws a truffle/assert.sol not found error
test_buildler_v5_plugin() {
  echo "-----------------------------------------------------------"
  echo "> Visual inspection integration test (Buidler - no options)"
  echo "> YOU MUST LOOK AT THIS TEST TO DETECT FAILURE"
  echo "-----------------------------------------------------------"

  mv ./test/metacoin.js ./metacoin.js
  mv ./buidler-metacoinjs-template.js ./test/buidler-metacoinjs-template.js
  mv ./test/TestMetacoin.sol ./TestMetacoin.sol

  npx buidler test --network development

  mv ./TestMetacoin.sol ./test/TestMetacoin.sol
  mv ./metacoin.js ./test/metacoin.js
  mv ./test/buidler-metacoinjs-template.js ./buidler-metacoinjs-template.js
}
