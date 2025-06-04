#!/usr/bin/env bash
#
# E2E CI: installs PR candidate on sc-forks/hardhat-e2e (a simple example,
# similar to Metacoin) and runs coverage
#

set -o errexit

function verifyCoverageExists {
  if [ ! -d "coverage" ]; then
    echo "ERROR: no coverage folder was created."
    exit 1
  fi
}

function verifyMatrixExists {
  if [ ! -f "testMatrix.json" ]; then
    echo "ERROR: no matrix file was created."
    exit 1
  fi
}

# Get rid of any caches
sudo rm -rf node_modules
echo "NVM CURRENT >>>>>" && nvm current

# Use PR env variables (for forks) or fallback on local if PR not available
SED_REGEX="s/git@github.com:/https:\/\/github.com\//"

if [[ -v CIRCLE_PR_REPONAME ]]; then
  PR_PATH="https://github.com/$CIRCLE_PR_USERNAME/$CIRCLE_PR_REPONAME#$CIRCLE_SHA1"
else
  PR_PATH=$(echo "$CIRCLE_REPOSITORY_URL#$CIRCLE_SHA1" | sudo sed "$SED_REGEX")
fi

echo "PR_PATH >>>>> $PR_PATH"

echo ""
echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
echo "Simple hardhat/hardhat-trufflev5    "
echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
echo ""

# Install hardhat-e2e (HardhatEVM)
git clone https://github.com/sc-forks/hardhat-e2e.git
cd hardhat-e2e
npm install --silent

# Install and run solidity-coverage @ PR
npm install --save-dev --silent $PR_PATH
cat package.json

npx hardhat init-foundry
cat foundry.toml

npx hardhat coverage

verifyCoverageExists

npx hardhat coverage --matrix

verifyMatrixExists

cat testMatrix.json

echo ""
echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
echo "wighawag/hardhat-deploy                "
echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
echo ""
cd ..
npm install -g yarn
git clone https://github.com/cgewecke/template-ethereum-contracts.git
cd template-ethereum-contracts
yarn
yarn add $PR_PATH --dev
cat package.json

# Here we want to make sure that HH cache triggers a
# complete recompile after coverage runs by verifying
# that gas consumption is same in both runs.
yarn run gas
yarn run coverage
yarn run gas

verifyCoverageExists
