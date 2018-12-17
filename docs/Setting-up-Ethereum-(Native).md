## Dream API

i.e., something I'm envisioning for *soon*(tm)

```go
eth, err := eth.New(/*config*/)
if err != nil {
    logger.Fatalln(err)
}

// State holds accounts without matching private keys
state := eth.State()
// wallet holds accounts with matching private keys
wallet := eth.Wallet()
wallet.NewAccount() // create a new account (return Account)
wallet.Accounts() // return []Account

acc := wallet.GetAcccount(0) // Get first account (return Account)
to := state.GetAccount(toAddr)
// Transact from the account
err := acc.Transact(to, big(100), big(10000), big(500), big(util.DefaultGasPrice), nil)
if err != nil {
    logger.Fatalln(err)
}
```