module.exports = {
  blockgenesis: require('./BasicTests/blockgenesistest'),
  genesishashes: require('./BasicTests/genesishashestest'),
  hexencode: require('./BasicTests/hexencodetest'),
  keyaddrtests: require('./BasicTests/keyaddrtest'),
  rlptest: require('./BasicTests/rlptest'),
  trietest: require('./TrieTests/trietest'),
  trietestnextprev: require('./TrieTests/trietestnextprev'),
  txtest: require('./BasicTests/txtest'),
  StateTests: {
    stExample: require('./StateTests/stExample.json'),
    stInitCodeTest: require('./StateTests/stInitCodeTest.json'),
    stLogTests: require('./StateTests/stLogTests.json'),
    stPreCompiledContracts: require('./StateTests/stPreCompiledContracts'),
    stRecursiveCreate: require('./StateTests/stRecursiveCreate'),
    stSpecial: require('./StateTests/stSpecialTest'),
    stSystemOperationsTest: require('./StateTests/stSystemOperationsTest'),
    stTransactionTest: require('./StateTests/stTransactionTest')
  },
  VMTests: {
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
