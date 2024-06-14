#!/usr/bin/env bash

set -o errexit

if [ "$TEST" = "integration" ]; then

  npm test

elif [ "$TEST" = "geth" ]; then

  npx geth-dev-assistant \
    --launch \
    --tag 'latest' \
    --accounts 4 \
    --balance 100 \
    --gasLimit 8000000

  npm test
  docker stop geth-client

elif [ "$TEST" = "colony" ]; then

  npm install -g yarn
  git clone https://github.com/JoinColony/colonyNetwork.git
  cd colonyNetwork || exit
  yarn
  yarn remove -W eth-gas-reporter --dev

  env

  SLUG="$TRAVIS_REPO_SLUG"
  BRANCH="$TRAVIS_BRANCH"

  if [ -n "$TRAVIS_PULL_REQUEST_SLUG" ]; then
    SLUG="$TRAVIS_PULL_REQUEST_SLUG"
  fi

  if [ -n "$TRAVIS_PULL_REQUEST_BRANCH" ]; then
    BRANCH="$TRAVIS_PULL_REQUEST_BRANCH"
  fi

  echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
  echo "TESTING BRANCH: https://github.com/$SLUG.git#$BRANCH"
  echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"

  yarn add -W https://github.com/"$SLUG".git#"$BRANCH"
  git submodule update --init
  yarn run provision:token:contracts
  DEBUG_CODECHECKS_TABLE=true yarn run test:contracts:gasCosts

fi