# Upgrade Notes

## Prague / EIP-2935

- The history storage contract is predeployed only in the developer genesis.
- For other networks, the contract is deployed at Prague activation by the system call during block processing and state access.
- Nodes upgrading after Prague will perform a one-time backfill of recent parent hashes into the ring buffer. If historical headers are pruned or unavailable, missing slots are skipped and only available hashes are filled.
- If you maintain a custom genesis and want predeployment, add an account entry for `HistoryStorageAddress` with `Nonce: 1` and `Code: HistoryStorageCode`.
