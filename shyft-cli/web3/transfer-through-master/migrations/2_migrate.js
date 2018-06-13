var Transfers = artifacts.require('./Transfers.sol')
var Transfers2 = artifacts.require('./Transfers2.sol')

module.exports = function (deployer) {
  	deployer.deploy(Transfers)
  	deployer.deploy(Transfers2)
}
