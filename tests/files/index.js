module.exports = {
  blockgenesis: require('./blockgenesistest'),
  genesishashes: require('./genesishashestest'),
  hexencode: require('./hexencodetest'),
  keyaddrtests: require('./keyaddrtest'),
  namecoin: require('./namecoin'),
  rlptest: require('./rlptest'),
  trietest: require('./trietest'),
  trietestnextprev: require('./trietestnextprev'),
  txtest: require('./txtest'),
  vmtests: {
    random: require('./vmtests/random'),
    vmArithmeticTest: require('./vmtests/vmArithmeticTest'),
    vmBitwiseLogicOperationTest: require('./vmtests/vmBitwiseLogicOperationTest'),
    vmBlockInfoTest: require('./vmtests/vmBlockInfoTest'),
    vmEnvironmentalInfoTest: require('./vmtests/vmEnvironmentalInfoTest'),
    vmIOandFlowOperationsTest: require('./vmtests/vmIOandFlowOperationsTest'),
    vmPushDupSwapTest: require('./vmtests/vmPushDupSwapTest'),
    vmSha3Test: require('./vmtests/vmSha3Test'),
    vmSystemOperationsTest: require('./vmtests/vmSystemOperationsTest'),
    vmtests: require('./vmtests/vmtests')
  }
};
