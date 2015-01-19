module.exports = {
  blockgenesis: require('./BasicTests/blockgenesistest'),
  genesishashes: require('./BasicTests/genesishashestest'),
  hexencode: require('./BasicTests/hexencodetest'),
  keyaddrtests: require('./BasicTests/keyaddrtest'),
  rlptest: require('./BasicTests/rlptest'),
  trieTests: {
    trietest: require('./TrieTests/trietest'),
    trietestnextprev: require('./TrieTests/trietestnextprev'),
    trieanyorder: require('./TrieTests/trieanyorder')
  },
  txtest: require('./BasicTests/txtest'),
  stateTests: require('require-all')(__dirname + '/StateTests/'),
  vmTests: {
    vmRandom: require('./VMTests/RandomTests/randomTest'),
    vmArithmeticTest: require('./VMTests/vmArithmeticTest'),
    vmBitwiseLogicOperationTest: require('./VMTests/vmBitwiseLogicOperationTest'),
    vmBlockInfoTest: require('./VMTests/vmBlockInfoTest'),
    vmEnvironmentalInfoTest: require('./VMTests/vmEnvironmentalInfoTest'),
    vmIOandFlowOperationsTest: require('./VMTests/vmIOandFlowOperationsTest'),
    vmLogTest: require('./VMTests/vmLogTest'),
    vmPushDupSwapTest: require('./VMTests/vmPushDupSwapTest'),
    vmSha3Test: require('./VMTests/vmSha3Test'),
    vmtests: require('./VMTests/vmtests')
  }
};
