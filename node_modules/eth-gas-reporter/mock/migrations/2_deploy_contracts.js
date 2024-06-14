var ConvertLib = artifacts.require('./ConvertLib.sol')
var MetaCoin = artifacts.require('./MetaCoin.sol')
var Wallet = artifacts.require('./Wallets/Wallet.sol')
var VariableCosts = artifacts.require('./VariableCosts.sol')
var VariableConstructor = artifacts.require('./VariableConstructor')
var Undeployed = artifacts.require('./Undeployed')

module.exports = function (deployer) {
  deployer.deploy(ConvertLib)
  deployer.link(ConvertLib, MetaCoin)
  deployer.link(ConvertLib, Undeployed)
  deployer.deploy(MetaCoin)
  deployer.deploy(Wallet)
  deployer.deploy(VariableCosts)
  deployer.deploy(VariableConstructor, 'Exit Visa')
}
