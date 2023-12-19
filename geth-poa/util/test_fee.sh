#!/bin/sh

set -x

# For testing where fees are accumulated

treasuryAcct=0x0FD1bDBB92AF752a201A900e0E2bc68253C14b4c

# Before 
tresuryBalance=$(cast balance $treasuryAcct)
payerBalance=$(cast balance 0xBe3dEF3973584FdcC1326634aF188f0d9772D57D)
signer1Balance=$(cast balance 0x788EBABe5c3dD422Ef92Ca6714A69e2eabcE1Ee4)
signer2Balance=$(cast balance 0xd9cd8E5DE6d55f796D980B818D350C0746C25b97)
receiverBalance=$(cast balance 0x110F2d06f045299Ed38fE8D1BdF172461Cd7B918)

# Send tx
cast send --private-key 0xc065f4c9a6dda0785e2224f5af8e473614de1c029acf094f03d5830e2dd5b0ea 0x110F2d06f045299Ed38fE8D1BdF172461Cd7B918 --value 0.1ether

# After
tresuryBalanceAfter=$(cast balance $treasuryAcct)
payerBalanceAfter=$(cast balance 0xBe3dEF3973584FdcC1326634aF188f0d9772D57D)
signer1BalanceAfter=$(cast balance 0x788EBABe5c3dD422Ef92Ca6714A69e2eabcE1Ee4)
signer2BalanceAfter=$(cast balance 0xd9cd8E5DE6d55f796D980B818D350C0746C25b97)
receiverBalanceAfter=$(cast balance 0x110F2d06f045299Ed38fE8D1BdF172461Cd7B918)

# Print diff
echo "Tresury balance diff: $(($tresuryBalanceAfter - $tresuryBalance))"
echo "Payer balance diff: $(($payerBalanceAfter - $payerBalance))"
echo "Signer1 balance diff: $(($signer1BalanceAfter - $signer1Balance))"
echo "Signer2 balance diff: $(($signer2BalanceAfter - $signer2Balance))"
echo "Receiver balance diff: $(($receiverBalanceAfter - $receiverBalance))"