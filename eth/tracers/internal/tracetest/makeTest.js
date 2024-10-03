// makeTest generates a test for the configured tracer by running
// a prestate reassembled and a call trace run, assembling all the
// gathered information into a test case.
var makeTest = function(tx, traceConfig) {
    // Generate the genesis block from the block, transaction and prestate data
    var block   = eth.getBlock(eth.getTransaction(tx).blockHash);
    var genesis = eth.getBlock(block.parentHash);

    delete genesis.gasUsed;
    delete genesis.logsBloom;
    delete genesis.parentHash;
    delete genesis.receiptsRoot;
    delete genesis.sha3Uncles;
    delete genesis.size;
    delete genesis.transactions;
    delete genesis.transactionsRoot;
    delete genesis.uncles;

    genesis.gasLimit  = genesis.gasLimit.toString();
    genesis.number    = genesis.number.toString();
    genesis.timestamp = genesis.timestamp.toString();

    genesis.alloc = debug.traceTransaction(tx, {tracer: "prestateTracer"});
    for (var key in genesis.alloc) {
        var nonce = genesis.alloc[key].nonce;
        if (nonce) {
            genesis.alloc[key].nonce = nonce.toString();
        }
    }
    genesis.config = admin.nodeInfo.protocols.eth.config;

    // Generate the call trace and produce the test input
    var result = debug.traceTransaction(tx, traceConfig);
    delete result.time;

    var context = {
        number:     block.number.toString(),
        difficulty: block.difficulty,
        timestamp:  block.timestamp.toString(),
        gasLimit:   block.gasLimit.toString(),
        miner:      block.miner,
    };
    if (block.baseFeePerGas) {
        context.baseFeePerGas = block.baseFeePerGas.toString();
    }

    console.log(JSON.stringify({
        genesis: genesis,
        context: context,
        input:  eth.getRawTransaction(tx),
        result: result,
        tracerConfig: traceConfig.tracerConfig,
    }, null, 2));
}
