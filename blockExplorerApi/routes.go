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
		"GetBalance",
		"GET",
		"/api/get_balance/",
		GetBalance,
	},
	Route{
		"GetBalances",
		"GET",
		"/api/get_balances/{addresses}",
		GetBalances,
	},
	Route{
		"GetBlocksMined",
		"GET",
		"/api/get_blocks_mined",
		GetBlocksMined,
	},
	Route{
		"GetTransactions",
		"GET",
		"/api/get_transactions/{address}",
		GetTransactions,
	},
	Route{
		"GetAllTransactions",
		"GET",
		"/api/get_all_transactions/",
		GetAllTransactions,
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
