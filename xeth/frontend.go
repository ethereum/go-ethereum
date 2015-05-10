package xeth

// Frontend should be implemented by users of XEth. Its methods are
// called whenever XEth makes a decision that requires user input.
type Frontend interface {
	// UnlockAccount is called when a transaction needs to be signed
	// but the key corresponding to the transaction's sender is
	// locked.
	//
	// It should unlock the account with the given address and return
	// true if unlocking succeeded.
	UnlockAccount(address []byte) bool

	// This is called for all transactions inititated through
	// Transact. It should prompt the user to confirm the transaction
	// and return true if the transaction was acknowledged.
	//
	// ConfirmTransaction is not used for Call transactions
	// because they cannot change any state.
	ConfirmTransaction(tx string) bool
}

// dummyFrontend is a non-interactive frontend that allows all
// transactions but cannot not unlock any keys.
type dummyFrontend struct{}

func (dummyFrontend) UnlockAccount([]byte) bool      { return false }
func (dummyFrontend) ConfirmTransaction(string) bool { return true }
