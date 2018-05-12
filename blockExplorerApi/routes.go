package main

//@NOTE Shyft setting up endpoints
import "net/http"

//Route stuct
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

//Routes routes
type Routes []Route

var routes = Routes{
	Route{
		"GetAccount",
		"GET",
		"/api/get_account/{address}",
		GetAccount,
	},
	Route{
		"GetAllAccounts",
		"GET",
		"/api/get_all_accounts",
		GetAllAccounts,
	},
	Route{
		"GetAllBlocks",
		"GET",
		"/api/get_all_blocks",
		GetAllBlocks,
	},
	Route{
		"GetBlock",
		"GET",
		"/api/get_block/{blockNumber}",
		GetBlock,
	},
	Route{
		"GetAllTransactions",
		"GET",
		"/api/get_all_transactions",
		GetAllTransactions,
	},
	Route{
		"GetTransaction",
		"GET",
		"/api/get_transaction/{txHash}",
		GetTransaction,
	},
	Route{
		"GetInternalTransactions",
		"GET",
		"/api/get_internal_transactions/{address}",
		GetInternalTransactions,
	},
	Route{
		"GetInternalTransactionsHash",
		"GET",
		"/api/get_internal_transactions_hash/{transactions_hash}",
		GetInternalTransactionsHash,
	},
}
