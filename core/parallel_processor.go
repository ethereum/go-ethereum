package core

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// storageAccessPair identifies a declared storage slot in an EIP-2930 access list.
type storageAccessPair struct {
	addr common.Address
	key  common.Hash
}

// BuildTransactionStorageParallelGroups partitions transaction indices into waves
// where each wave may be scheduled together only if every pair of txs has
// pairwise disjoint declared address sets (sender, to, and all access-list
// addresses). Assumes access lists completely declare touched accounts.
//
// Groups are built greedily in block order: each group starts at the smallest
// index not yet assigned; txs are scanned in ascending index order and a tx is
// appended only if its address set does not intersect any member of the group.
func BuildTransactionStorageParallelGroups(txs []*types.Transaction, signer types.Signer) ([][]int, error) {
	n := len(txs)
	if n == 0 {
		return nil, nil
	}
	if !ParallelTxGroupingByStorageOverlap {
		groups := make([][]int, n)
		for i := 0; i < n; i++ {
			groups[i] = []int{i}
		}
		return groups, nil
	}
	addrSets, err := collectDeclaredAddressSets(txs, signer)
	if err != nil {
		return nil, err
	}
	unassigned := make([]bool, n)
	for i := range unassigned {
		unassigned[i] = true
	}
	var groups [][]int
	for {
		seed := -1
		for i := 0; i < n; i++ {
			if unassigned[i] {
				seed = i
				break
			}
		}
		if seed == -1 {
			break
		}
		group := []int{seed}
		unassigned[seed] = false
		for j := 0; j < n; j++ {
			if !unassigned[j] {
				continue
			}
			conflict := false
			for _, gi := range group {
				if declaredAddressSetsOverlap(addrSets[gi], addrSets[j]) {
					conflict = true
					break
				}
			}
			if !conflict {
				group = append(group, j)
				unassigned[j] = false
			}
		}
		groups = append(groups, group)
	}
	return groups, nil
}

// collectTransactionStorageSlotSets returns, per transaction index, the set of
// declared (address, storage key) pairs from the EIP-2930 access list. Indices
// with no declared storage slots have a nil map entry.
func collectTransactionStorageSlotSets(txs []*types.Transaction) []map[storageAccessPair]struct{} {
	n := len(txs)
	slotSets := make([]map[storageAccessPair]struct{}, n)
	for i, tx := range txs {
		acl := tx.AccessList()
		if acl == nil {
			continue
		}
		set := make(map[storageAccessPair]struct{})
		for _, tuple := range acl {
			for _, key := range tuple.StorageKeys {
				set[storageAccessPair{addr: tuple.Address, key: key}] = struct{}{}
			}
		}
		if len(set) > 0 {
			slotSets[i] = set
		}
	}
	return slotSets
}

// collectDeclaredAddressSets returns, per tx index, the set of declared
// addresses: sender (from), recipient (to) if any, and every address listed in
// the EIP-2930 access list (including address-only tuples).
func collectDeclaredAddressSets(txs []*types.Transaction, signer types.Signer) ([]map[common.Address]struct{}, error) {
	n := len(txs)
	sets := make([]map[common.Address]struct{}, n)
	for i, tx := range txs {
		from, err := types.Sender(signer, tx)
		if err != nil {
			return nil, fmt.Errorf("tx %d: %w", i, err)
		}
		set := make(map[common.Address]struct{})
		set[from] = struct{}{}
		if to := tx.To(); to != nil {
			set[*to] = struct{}{}
		}
		for _, tuple := range tx.AccessList() {
			set[tuple.Address] = struct{}{}
		}
		sets[i] = set
	}
	return sets, nil
}

func declaredStorageSlotsOverlap(a, b map[storageAccessPair]struct{}) bool {
	if a == nil || b == nil {
		return false
	}
	for pair := range a {
		if _, ok := b[pair]; ok {
			return true
		}
	}
	return false
}

func declaredAddressSetsOverlap(a, b map[common.Address]struct{}) bool {
	for addr := range a {
		if _, ok := b[addr]; ok {
			return true
		}
	}
	return false
}

// Sets each receipt's CumulativeGasUsed from the canonical block order (sum of GasUsed for txs 0..i).
// Call this after parallel execution where each ApplyTransactionWithEVM used a per-tx local cumulative base.
func normalizeReceiptCumulativeGas(receipts []*types.Receipt) {
	var cum uint64
	for _, r := range receipts {
		cum += r.GasUsed
		r.CumulativeGasUsed = cum
	}
}

// MARK: - Logging

// Prints for each transaction, the data structures it will access according to its EIP-2930 access list (addresses and storage keys).
// Assumes access lists are fully and accurately declared.
// Legacy transactions have no access list.
func PrintTransactionAccessLists(txs []*types.Transaction) {
	for i, tx := range txs {
		acl := tx.AccessList()
		if acl == nil {
			fmt.Printf("tx %d [%s]: no access list (legacy)\n", i, tx.Hash().Hex())
			continue
		}
		if len(acl) == 0 {
			fmt.Printf("tx %d [%s]: empty access list\n", i, tx.Hash().Hex())
			continue
		}
		fmt.Printf("tx %d [%s]: access list (%d entries)\n", i, tx.Hash().Hex(), len(acl))
		for j, tuple := range acl {
			if len(tuple.StorageKeys) == 0 {
				fmt.Printf("  [%d] address %s (no storage keys)\n", j, tuple.Address.Hex())
			} else {
				fmt.Printf("  [%d] address %s storage keys:\n", j, tuple.Address.Hex())
				for k, key := range tuple.StorageKeys {
					fmt.Printf("    [%d] %s\n", k, key.Hex())
				}
			}
		}
	}
}

// Prints an account-level diagnostic adjacency matrix M
// Where M[i][j] == 1 if transactions i and j touch at least one common account.
// Every access-list address plus the tx recipient (To), when present.
// Legacy txs contribute only To. Pairs i==j are always 0.
func PrintTransactionAccessOverlapMatrix(txs []*types.Transaction) {
	n := len(txs)
	if n == 0 {
		return
	}
	// Precompute the set of accounts per transaction (access-list + To).
	addrSets := make([]map[common.Address]struct{}, n)
	for i, tx := range txs {
		acl := tx.AccessList()

		set := make(map[common.Address]struct{})

		// Access-list accounts
		for _, tuple := range acl {
			set[tuple.Address] = struct{}{}
		}

		// Recipient account (account-level balance/nonce/code)
		if to := tx.To(); to != nil {
			set[*to] = struct{}{}
		}

		// If you want to also include senders and you have a signer in scope:
		// from, err := types.Sender(signer, tx)
		// if err == nil {
		//     set[from] = struct{}{}
		// }

		if len(set) > 0 {
			addrSets[i] = set
		}
	}

	fmt.Println("transaction account-level access overlap matrix (1 = share at least one account):")
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			val := 0
			if i != j && addrSets[i] != nil && addrSets[j] != nil {
				for addr := range addrSets[i] {
					if _, ok := addrSets[j][addr]; ok {
						val = 1
						break
					}
				}
			}
			if j+1 == n {
				fmt.Printf("%d\n", val)
			} else {
				fmt.Printf("%d ", val)
			}
		}
	}
}

// Prints BuildTransactionStorageParallelGroups output as human-readable index lists.
func PrintTransactionStorageParallelGroups(txs []*types.Transaction, signer types.Signer) {
	groups, err := BuildTransactionStorageParallelGroups(txs, signer)
	if err != nil {
		fmt.Printf("transaction address-parallel groups: (error building groups: %v)\n", err)
		return
	}
	if len(groups) == 0 {
		return
	}
	fmt.Print("transaction address-parallel groups (tx indices, greedy in block order): ")
	for g, group := range groups {
		if g > 0 {
			fmt.Print(" | ")
		}
		fmt.Print("[")
		for k, idx := range group {
			if k > 0 {
				fmt.Print(" ")
			}
			fmt.Printf("%d", idx)
		}
		fmt.Print("]")
	}
	fmt.Println()
}

// Prints a storage-level diagnostic adjacency matrix M
// Where M[i][j] == 1 if transactions i and j declare at least one identical (address, storage key) pair in their access lists.
// Address-only access-list entries (no storage keys) do not contribute.
// Pairs i==j are always 0.
func PrintTransactionStorageAccessOverlapMatrix(txs []*types.Transaction) {
	n := len(txs)
	if n == 0 {
		return
	}
	slotSets := collectTransactionStorageSlotSets(txs)

	fmt.Println("transaction storage-slot overlap matrix (1 = share at least one declared address+storage key):")
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			val := 0
			if i != j && slotSets[i] != nil && slotSets[j] != nil {
				for pair := range slotSets[i] {
					if _, ok := slotSets[j][pair]; ok {
						val = 1
						break
					}
				}
			}
			if j+1 == n {
				fmt.Printf("%d\n", val)
			} else {
				fmt.Printf("%d ", val)
			}
		}
	}
}
