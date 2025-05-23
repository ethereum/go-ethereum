const MetaCoin = artifacts.require('./MetaCoin.sol');
const ConvertLib = artifacts.require('./ConvertLib.sol');

contract('MetaCoin', function (accounts) {
  let meta;

  before(async function(){
    const lib = await ConvertLib.new();
    MetaCoin.link(lib);
  })

  beforeEach(async function () {
    meta = await MetaCoin.new()
    meta = await MetaCoin.new()
  })
  afterEach(async function () {
    meta = await MetaCoin.new()
    meta = await MetaCoin.new()
  })

  it('should put 10000 MetaCoin in the first account', async function () {
    const balance = await meta.getBalance.call(accounts[0])
    assert.equal(balance.valueOf(), 10000, "10000 wasn't in the first account")
  })

  it('should call a function that depends on a linked library', function () {
    var metaCoinBalance
    var metaCoinEthBalance

    return meta.getBalance.call(accounts[0]).then(function (outCoinBalance) {
      metaCoinBalance = parseInt(outCoinBalance.toString())
      return meta.getBalanceInEth.call(accounts[0])
    }).then(function (outCoinBalanceEth) {
      metaCoinEthBalance = parseInt(outCoinBalanceEth.toString())
    }).then(function () {
      assert.equal(metaCoinEthBalance, 2 * metaCoinBalance, 'Library function returned unexpected function, linkage may be broken')
    })
  })
  it('should send coin correctly', function () {
    // Get initial balances of first and second account.
    var account_one = accounts[0]
    var account_two = accounts[1]

    var account_one_starting_balance
    var account_two_starting_balance
    var account_one_ending_balance
    var account_two_ending_balance

    var amount = 10

    return meta.getBalance.call(account_one).then(function (balance) {
      account_one_starting_balance = parseInt(balance.toString())
      return meta.getBalance.call(account_two)
    }).then(function (balance) {
      account_two_starting_balance = parseInt(balance.toString())
      return meta.sendCoin(account_two, amount, {from: account_one})
    }).then(function () {
      return meta.getBalance.call(account_one)
    }).then(function (balance) {
      account_one_ending_balance = parseInt(balance.toString())
      return meta.getBalance.call(account_two)
    }).then(function (balance) {
      account_two_ending_balance = parseInt(balance.toString())

      assert.equal(account_one_ending_balance, account_one_starting_balance - amount, "Amount wasn't correctly taken from the sender")
      assert.equal(account_two_ending_balance, account_two_starting_balance + amount, "Amount wasn't correctly sent to the receiver")
    })
  })
})
