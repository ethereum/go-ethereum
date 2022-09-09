---
title: Security
description: A primer on Geth security best practice.
---

## Downloading Geth

Download Geth using the links on our [Downloads](/downloads) page. The SHA256 hashes of the downloaded files can be compared to ours to ensure precise consistency with our releases. This protects against malicious code being inadvertently downloaded from an adversarial source. The same measures should also be taken to download trusted consensus client software.

## Networking security

The local machine's firewall settings should:

- Block all traffic to `8545`, or whatever custom port has been defined for JSON-RPC requests to the node, except for traffic from explicitly defined trusted machines.
- Allow traffic on `TCP 30303` or whichever custom port has been defined for peer-to-peer communications. This allows the node to connect to peers.
- Allow traffic on `UDP 30303` or whichever custom port has been defined for peer-to-peer communications. This allows node discovery.

## Account security

Account security comes down to keeping private keys and account passwords backed up and inaccessible to adversaries. This is something that users take responsibility for. Geth provides an encrypted store for keys that are unlocked using an account password. If the key files or the passwors are lost, the account is impossible to access and the funds are effectively lost forever. If access to the unencrypted keys is obtained by an adversary they gain control of any funds associated with the account.

Geth has built-in account management tools that are sufficiently secure for most purposes. However, Clef is recommended as an external account management and signing tool. It can be run decoupled from Geth and can even be run on dedicated secure external hardware such as a VM or a secure USB drive. This is considered best practise because the user is required to manually review all actions that touch sensitive data, except where specific predefined rules are implemented. Signing is done locally to Clef rather than giving key access to a node.

Geth allows account unlocking by passing account passwords at startup. This unlocks the account all the while that Geth is running. This is not allowed when `http` traffic is enabled, even with appropriate firewall settings. The combination of `http` and `-unlock` poses too much of a security risk because an attacker able to access the node over the exposed HTTP port would be able to make JSON-RPC requests to the node from the unlocked account, including sending funds to other addresses.

**back up your keystore and passwords safely and securely!**

## Other security considerations

Even with a perfectly secure node, users can still be manipulated by attackers into exposing security weaknesses or inadvertently interact with insecure smart contracts. For an overview, please see the Ethereum [security best practise webpage](https://ethereum.org/en/security) and this introduction to [smart contract security](https://ethereum.org/en/developers/docs/smart-contracts/security).
