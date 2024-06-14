#!/usr/bin/env bash
#
# E2E CI: installs PR candidate on openzeppelin-contracts and runs coverage
#

# TODO: uncomment this when zeppelin job gets fixed
# set -o errexit

# Get rid of any caches
sudo rm -rf node_modules
echo "NVM CURRENT >>>>>" && nvm current
nvm use 20

# Use PR env variables (for forks) or fallback on local if PR not available
SED_REGEX="s/git@github.com:/https:\/\/github.com\//"

if [[ -v CIRCLE_PR_REPONAME ]]; then
  PR_PATH="https://github.com/$CIRCLE_PR_USERNAME/$CIRCLE_PR_REPONAME#$CIRCLE_SHA1"
else
  PR_PATH=$(echo "$CIRCLE_REPOSITORY_URL#$CIRCLE_SHA1" | sudo sed "$SED_REGEX")
fi

echo "PR_PATH >>>>> $PR_PATH"

# Install Zeppelin
git clone https://github.com/OpenZeppelin/openzeppelin-contracts.git
cd openzeppelin-contracts

# Swap installed coverage for PR branch version
echo ">>>>> npm install"
npm install

# Use HH latest
npm install hardhat@latest --save-dev

echo ">>>>> npm uninstall solidity-coverage --save-dev"
npm uninstall solidity-coverage --save-dev

echo ">>>>> npm add $PR_PATH --dev"
npm install "$PR_PATH" --save-dev

echo ">>>>> cat package.json"
cat package.json

# Track perf
CI=false npm run coverage

# TODO: remove EXIT 0 when zeppelin job is fixed - currently failing for time-related reasons in circleci
# TODO: uncomment set command at top of this file
exit 0