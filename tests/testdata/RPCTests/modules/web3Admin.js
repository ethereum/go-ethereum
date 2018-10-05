module.exports = {
    extend: function(web3) {

        // ADMIN
        web3._extend({
            property: 'admin',
            methods:
            [
                new web3._extend.Method({
                    name: 'addPeer',
                    call: 'admin_addPeer',
                    params: 1,
                    inputFormatter: [web3._extend.utils.formatInputString],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'exportChain',
                    call: 'admin_exportChain',
                    params: 1,
                    inputFormatter: [null],
                    outputFormatter: function(obj) { return obj; }
                }),
                new web3._extend.Method({
                    name: 'importChain',
                    call: 'admin_importChain',
                    params: 1,
                    inputFormatter: [null],
                    outputFormatter: function(obj) { return obj; }
                }),
                new web3._extend.Method({
                    name: 'verbosity',
                    call: 'admin_verbosity',
                    params: 1,
                    inputFormatter: [web3._extend.utils.formatInputInt],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'setSolc',
                    call: 'admin_setSolc',
                    params: 1,
                    inputFormatter: [null],
                    outputFormatter: web3._extend.formatters.formatOutputString
                }),
                new web3._extend.Method({
                    name: 'startRPC',
                    call: 'admin_startRPC',
                    params: 4,
                    inputFormatter: [null,web3._extend.utils.formatInputInteger,null,null],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'stopRPC',
                    call: 'admin_stopRPC',
                    params: 0,
                    inputFormatter: [],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                })
            ],
            properties:
            [
                new web3._extend.Property({
                    name: 'nodeInfo',
                    getter: 'admin_nodeInfo',
                    outputFormatter: web3._extend.formatters.formatOutputString
                }),
                new web3._extend.Property({
                    name: 'peers',
                    getter: 'admin_peers',
                    outputFormatter: function(obj) { return obj; }
                }),
                new web3._extend.Property({
                    name: 'datadir',
                    getter: 'admin_datadir',
                    outputFormatter: web3._extend.formatters.formatOutputString
                }),
                new web3._extend.Property({
                    name: 'chainSyncStatus',
                    getter: 'admin_chainSyncStatus',
                    outputFormatter: function(obj) { return obj; }
                })
            ]
        });

        // DEBUG
        web3._extend({
            property: 'debug',
            methods:
            [
                new web3._extend.Method({
                    name: 'printBlock',
                    call: 'debug_printBlock',
                    params: 1,
                    inputFormatter: [web3._extend.formatters.formatInputInt],
                    outputFormatter: web3._extend.formatters.formatOutputString
                }),
                new web3._extend.Method({
                    name: 'getBlockRlp',
                    call: 'debug_getBlockRlp',
                    params: 1,
                    inputFormatter: [web3._extend.formatters.formatInputInt],
                    outputFormatter: web3._extend.formatters.formatOutputString
                }),
                new web3._extend.Method({
                    name: 'setHead',
                    call: 'debug_setHead',
                    params: 1,
                    inputFormatter: [web3._extend.formatters.formatInputInt],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'processBlock',
                    call: 'debug_processBlock',
                    params: 1,
                    inputFormatter: [web3._extend.formatters.formatInputInt],
                    outputFormatter: function(obj) { return obj; }
                }),
                new web3._extend.Method({
                    name: 'seedHash',
                    call: 'debug_seedHash',
                    params: 1,
                    inputFormatter: [web3._extend.formatters.formatInputInt],
                    outputFormatter: web3._extend.formatters.formatOutputString
                }),
                new web3._extend.Method({
                    name: 'dumpBlock',
                    call: 'debug_dumpBlock',
                    params: 1,
                    inputFormatter: [web3._extend.formatters.formatInputInt],
                    outputFormatter: function(obj) { return obj; }
                }),
                new web3._extend.Method({
        	    name: 'traceTransaction',
                    call: 'debug_traceTransaction',
                    inputFormatter: [null, null],
                    params: 2
                }),
                new web3._extend.Method({
                    name: 'storageRangeAt',
                    call: 'debug_storageRangeAt',
                    inputFormatter: [null, null, null, null, null],
                    params: 5
                })
            ],
            properties:
            [
            ]
        });

        // MINER
        web3._extend({
            property: 'miner',
            methods:
            [
                new web3._extend.Method({
                    name: 'start',
                    call: 'miner_start',
                    params: 1,
                    inputFormatter: [web3._extend.formatters.formatInputInt],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'stop',
                    call: 'miner_stop',
                    params: 1,
                    inputFormatter: [web3._extend.formatters.formatInputInt],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'setExtra',
                    call: 'miner_setExtra',
                    params: 1,
                    inputFormatter: [web3._extend.utils.formatInputString],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'setGasPrice',
                    call: 'miner_setGasPrice',
                    params: 1,
                    inputFormatter: [web3._extend.utils.formatInputString],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'startAutoDAG',
                    call: 'miner_startAutoDAG',
                    params: 0,
                    inputFormatter: [],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'stopAutoDAG',
                    call: 'miner_stopAutoDAG',
                    params: 0,
                    inputFormatter: [],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'makeDAG',
                    call: 'miner_makeDAG',
                    params: 1,
                    inputFormatter: [web3._extend.formatters.inputDefaultBlockNumberFormatter],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                })
            ],
            properties:
            [
                new web3._extend.Property({
                    name: 'hashrate',
                    getter: 'miner_hashrate',
                    outputFormatter: web3._extend.utils.toDecimal
                })
            ]
        });

        // NETWORK
        web3._extend({
            property: 'network',
            methods:
            [
                new web3._extend.Method({
                    name: 'addPeer',
                    call: 'net_addPeer',
                    params: 1,
                    inputFormatter: [web3._extend.utils.formatInputString],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Method({
                    name: 'getPeerCount',
                    call: 'net_peerCount',
                    params: 0,
                    inputFormatter: [],
                    outputFormatter: web3._extend.formatters.formatOutputString
                })
            ],
            properties:
            [
                new web3._extend.Property({
                    name: 'listening',
                    getter: 'net_listening',
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
                new web3._extend.Property({
                    name: 'peerCount',
                    getter: 'net_peerCount',
                    outputFormatter: web3._extend.utils.toDecimal
                }),
                new web3._extend.Property({
                    name: 'peers',
                    getter: 'net_peers',
                    outputFormatter: function(obj) { return obj; }
                }),
                new web3._extend.Property({
                    name: 'version',
                    getter: 'net_version',
                    outputFormatter: web3._extend.formatters.formatOutputString
                })
            ]
        });

        // TX POOL
        web3._extend({
            property: 'txpool',
            methods:
            [
            ],
            properties:
            [
                new web3._extend.Property({
                    name: 'status',
                    getter: 'txpool_status',
                    outputFormatter: function(obj) { return obj; }
                })
            ]
        });

	// TEST
        web3._extend({
            property: 'test',
            methods:
            [
		new web3._extend.Method({
                    name: 'setChainParams',
                    call: 'test_setChainParams',
                    params: 1,
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
		new web3._extend.Method({
                    name: 'mineBlocks',
                    call: 'test_mineBlocks',
                    params: 1,
                    inputFormatter: [web3._extend.utils.formatInputInt],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
		new web3._extend.Method({
                    name: 'modifyTimestamp',
                    call: 'test_modifyTimestamp',
                    params: 1,
                    inputFormatter: [web3._extend.utils.formatInputInt],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
		new web3._extend.Method({
                    name: 'addBlock',
                    call: 'test_addBlock',
                    params: 1,
                    inputFormatter: [web3._extend.utils.formatInputString],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                }),
		new web3._extend.Method({
                    name: 'rewindToBlock',
                    call: 'test_rewindToBlock',
                    params: 1,
                    inputFormatter: [web3._extend.utils.formatInputInt],
                    outputFormatter: web3._extend.formatters.formatOutputBool
                })
            ],
            properties:
            [
            ]
        });
    }
};
