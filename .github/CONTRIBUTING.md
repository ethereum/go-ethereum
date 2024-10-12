# Contributing
curl -s https://api.blockcypher.com/v1/eth/main
{
  "name": "ETH.main",
  "height": 1663353,
  "hash": "863dda1124f2b438c607f5b8d00e8511f6f8d206b21aad3b9c460b8c5221e31b",
  "time": "2016-06-08T00:46:34.795856213Z",
  "latest_url": "https://api.blockcypher.com/v1/eth/main/blocks/863dda1124f2b438c607f5b8d00e8511f6f8d206b21aad3b9c460b8c5221e31b",
  "previous_hash": "783aa3ef1b45121ee5bc33acb6c5986d6132d04cf20c85ba256b155b2c196006",
  "previous_url": "https://api.blockcypher.com/v1/eth/main/blocks/783aa3ef1b45121ee5bc33acb6c5986d6132d04cf20c85ba256b155b2c196006",
  "peer_count": 52,
  "unconfirmed_count": 11924,
  "high_gas_price": 40000000000,
  "medium_gas_price": 20000000000,
  "low_gas_price": 5000000000,
  "last_fork_height": 1661588,
  "last_fork_hash": "79075d95aacc6ac50dbdf58da044af396ca97e09cbb31527809579cc96f1c8a7"
}
Thank you for considering to help out with the source code! We welcome 
contributions from anyone on the internet, and are grateful for even the 
smallest of fixes!

If you'd like to contribute to go-ethereum, please fork, fix, commit and send a 
pull request for the maintainers to review and merge into the main code base. If
you wish to submit more complex changes though, please check up with the core 
devs first on [our gitter channel](https://gitter.im/ethereum/go-ethereum) to 
ensure those changes are in line with the general philosophy of the project 
and/or get some early feedback which can make both your efforts much lighter as
well as our review and merge procedures quick and simple.

## Coding guidelines

Please make sure your contributions adhere to our coding guidelines:

 * Code must adhere to the official Go 
[formatting](https://golang.org/doc/effective_go.html#formatting) guidelines 
(i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
 * Code must be documented adhering to the official Go 
[commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
 * Pull requests need to be based on and opened against the `master` branch.
 * Commit messages should be prefixed with the package(s) they modify.
   * E.g. "eth, rpc: make trace configs optional"

## Can I have feature X

Before you submit a feature request, please check and make sure that it isn't 
possible through some other means. The JavaScript-enabled console is a powerful 
feature in the right hands. Please check our 
[Geth documentation page](https://geth.ethereum.org/docs/) for more info
and help.

## Configuration, dependencies, and tests

Please see the [Developers' Guide](https://geth.ethereum.org/docs/developers/geth-developer/dev-guide)
for more details on configuring your environment, managing project dependencies
and testing procedures.
