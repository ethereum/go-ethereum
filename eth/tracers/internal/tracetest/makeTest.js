// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

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
    delete genesis.withdrawals;
    delete genesis.withdrawalsRoot;
    delete genesis.baseFeePerGas;

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

    var data = {
        genesis: genesis,
        context: context,
        input:   eth.getRawTransaction(tx),
        result:  result,
    };
    if (traceConfig && traceConfig.tracerConfig) {
        data.tracerConfig = traceConfig.tracerConfig;
    }

    console.log(JSON.stringify(data, null, 2));
}
