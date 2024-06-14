const Wallet = artifacts.require('./Wallet.sol')

contract('Wallet', accounts => {
  let walletA
  let walletB

  beforeEach(async function () {
    walletA = await Wallet.new()
    walletB = await Wallet.new()
  })

  it('should be very expensive to deploy', async() => {
    await Wallet.new()
  })

  it('should should allow transfers and sends', async () => {
    await walletA.sendTransaction({
      value: 100, from: accounts[0]
    })
    await walletA.sendPayment(50, walletB.address, {
      from: accounts[0]
    })
    await walletA.transferPayment(50, walletB.address, {
      from: accounts[0]
    })
    const balance = await walletB.getBalance()
    assert.equal(parseInt(balance.toString()), 100)
  })
})
