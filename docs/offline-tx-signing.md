# Offline transaction signing with Geth

Signing transactions offline improves security by keeping your private keys off of internet‑connected machines.

## Prerequisites

- A machine with geth installed (air‑gapped or offline).
- A machine with network access to broadcast the signed transaction.
- The account’s private key or keystore file.

## Steps

1. **Prepare the unsigned transaction** on an online machine using:
   ```bash
   geth --exec "eth.signTransaction({to: '0x...', value: web3.utils.toWei('1', 'ether'), gas: 21000, gasPrice: web3.utils.toWei('1', 'gwei'), nonce: web3.eth.getTransactionCount('0xYourAddress')})" attach
