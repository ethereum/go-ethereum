#!/bin/sh
set -exu

# Sleep to ensure chain is up
sleep 10

if test -f /hyperlane-monorepo/artifacts/done; then
  echo "Deploy artifacts already exist. Skipping deployment."
else
  echo "Deploying core contracts"
  # Define the expect script inline
  /usr/bin/expect <<EOF
  set timeout -1
  spawn hyperlane deploy core \
    --yes \
    --targets sepolia,mevcommitsettlement \
    --chains /chain-config.yml \
    --ism /multisig-ism.yml \
    --out "/hyperlane-monorepo/artifacts" \
    --key $HYPERLANE_DEPLOYER_PRIVATE_KEY
  expect {
    "*low balance on*" { 
      send -- "Y\r"
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
  echo "Deploying warp route"
  # Define the expect script inline
  /usr/bin/expect <<EOF
  set timeout -1
  spawn hyperlane deploy warp \
    --yes \
    --key $HYPERLANE_DEPLOYER_PRIVATE_KEY \
    --chains /chain-config.yml \
    --config /warp-tokens.yml \
    --core /hyperlane-monorepo/artifacts/core-deployment.json
  expect {
    "*low balance on*" { 
      send -- "Y\r"
      exp_continue
    }
    eof
  }
EOF

  # Standardize artifact names
  for file in /hyperlane-monorepo/artifacts/warp-deployment-*.json; do
    mv "$file" "/hyperlane-monorepo/artifacts/warp-deployment.json"
  done
  for file in /hyperlane-monorepo/artifacts/warp-ui-token-config-*.json; do
    mv "$file" "/hyperlane-monorepo/artifacts/warp-ui-token-config.json"
  done

  # Signal done 
  touch artifacts/done-warp-route
fi

# Sleep to allow deployer health check (polled every 5sec) to pass
sleep 10
