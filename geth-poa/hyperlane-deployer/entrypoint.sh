#!/bin/sh
set -exu

# Sleep to ensure chain is up
sleep 10

if test -f /hyperlane-monorepo/artifacts/done; then
  echo "Deploy artifacts already exist. Skipping deployment."
else

  # Define the expect script inline
  /usr/bin/expect <<EOF
  set timeout -1
  spawn hyperlane deploy core \
    --yes \
    --targets sepolia,mevcommitsettlement \
    --chains /chain-config.yml \
    --ism /multisig-ism.yml \
    --out "/hyperlane-monorepo/artifacts" \
    --key $CONTRACT_DEPLOYER_PRIVATE_KEY
  expect {
    "? Do you want use some existing contract addresses? (Y/n)" {
      send -- "n\r"
      exp_continue
    }
    "*low balance on*" { 
      send -- "y\r"
      exp_continue
    }
    eof
  }
EOF

  # Standardize artifact names
  for file in /hyperlane-monorepo/artifacts/agent-config-*.json; do
    mv "$file" "/hyperlane-monorepo/artifacts/agent-config.json"
  done
  for file in /hyperlane-monorepo/artifacts/core-deployment-*.json; do
    mv "$file" "/hyperlane-monorepo/artifacts/core-deployment.json"
  done

  # Signal done
  touch /hyperlane-monorepo/artifacts/done
fi

if test -f artifacts/done-warp-route; then
  echo "Warp route already deployed. Skipping."
else
  echo "Deploying warp route."
  hyperlane deploy warp \
    --yes \
    --key $CONTRACT_DEPLOYER_PRIVATE_KEY \
    --chains /chain-config.yml \
    --config /warp-tokens.yml \
    --core /hyperlane-monorepo/artifacts/core-deployment.json

  # Standardize artifact names
  for file in /hyperlane-monorepo/artifacts/warp-deployment-*.json; do
    mv "$file" "/hyperlane-monorepo/artifacts/warp-deployment.json"
  done
  for file in /hyperlane-monorepo/artifacts/warp-ui-token-config-*.json; do
    mv "$file" "/hyperlane-monorepo/artifacts/warp-ui-token-config.json"
  done
  
  touch artifacts/done-warp-route
fi

# Sleep to allow deployer health check (polled every 5sec) to pass
sleep 10
