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

    ["gasUsed", "logsBloom", "parentHash", "receiptsRoot", "sha3Uncles", 
     "size", "transactions", "transactionsRoot", "uncles"].forEach(prop => delete genesis[prop]);

    ["gasLimit", "number", "timestamp"].forEach(prop => genesis[prop] = genesis[prop].toString());

    genesis.alloc = debug.traceTransaction(tx, {tracer: "prestateTracer"});
    Object.keys(genesis.alloc).forEach(key => {
        if (genesis.alloc[key].nonce) {
            genesis.alloc[key].nonce = genesis.alloc[key].nonce.toString();
        }
    });
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
        genesis:       genesis,
        context:       context,
        input:         eth.getRawTransaction(tx),
        result:        result,
        tracerConfig:  traceConfig.tracerConfig,
    }, null, 2));
}
