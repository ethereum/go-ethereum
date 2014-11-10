module.exports = {
  blockgenesis: require('./BasicTests/blockgenesistest'),
  genesishashes: require('./BasicTests/genesishashestest'),
  hexencode: require('./BasicTests/hexencodetest'),
  keyaddrtests: require('./BasicTests/keyaddrtest'),
  rlptest: require('./BasicTests/rlptest'),
  trietest: require('./TrieTests/trietest'),
  trietestnextprev: require('./TrieTests/trietestnextprev'),
  txtest: require('./BasicTests/txtest'),
  randomTests: {
    201410211705: require('./randomTests/201410211705'),
    201410211708: require('./randomTests/201410211708')
  },
  StateTests: {
    stPreCompiledContracts: require('./StateTests/stPreCompiledContracts'),
    stSystemOperationsTest: require('./StateTests/stSystemOperationsTest'),
  },
  VMTests: {
    vmArithmeticTest: require('./VMTests/vmArithmeticTest'),
    vmBitwiseLogicOperationTest: require('./VMTests/vmBitwiseLogicOperationTest'),
    vmBlockInfoTest: require('./VMTests/vmBlockInfoTest'),
    vmEnvironmentalInfoTest: require('./VMTests/vmEnvironmentalInfoTest'),
    vmIOandFlowOperationsTest: require('./VMTests/vmIOandFlowOperationsTest'),
    vmPushDupSwapTest: require('./VMTests/vmPushDupSwapTest'),
    vmSha3Test: require('./VMTests/vmSha3Test'),
    vmtestst: require('./VMTests/vmtests'),
  }
};
