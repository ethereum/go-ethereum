// Copyright 2019 The go-ethereum Authors
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

package openrpc

// This file contains a string constant containing the JSON schema data for OpenRPC.

// OpenRPCSchema defines the default full suite of possibly available go-ethereum RPC
// methods.
const OpenRPCSchema = `
{
  "openrpc": "1.0.0",
  "info": {
    "version": "1.0.10",
    "title": "Ubiq JSON-RPC",
    "description": "This API lets you interact with an EVM-based client via JSON-RPC",
    "license": {
      "name": "Apache 2.0",
      "url": "https://www.apache.org/licenses/LICENSE-2.0.html"
    }
  },
  "methods": [
    {
      "name": "web3_clientVersion",
      "summary": "Returns the version of the current client.",
      "params": [],
      "result": {
        "name": "clientVersion",
        "description": "client version",
        "schema": {
          "title": "clientVersion",
          "type": "string"
        }
      }
    },
    {
      "name": "web3_sha3",
      "summary": "Hashes data using the Keccak-256 algorithm.",
      "params": [
        {
          "name": "data",
          "description": "data to hash using the Keccak-256 algorithm",
          "summary": "data to hash",
          "schema": {
            "title": "data",
            "type": "string",
            "pattern": "^0x[a-fA-F\\d]+$"
          }
        }
      ],
      "result": {
        "name": "hashedData",
        "description": "Keccak-256 hash of the given data",
        "schema": {
          "$ref": "#/components/schemas/Keccak"
        }
      },
      "examples": [
        {
          "name": "sha3Example",
          "params": [
            {
              "name": "sha3ParamExample",
              "value": "0x68656c6c6f20776f726c64"
            }
          ],
          "result": {
            "name": "sha3ResultExample",
            "value": "0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"
          }
        }
      ]
    },
    {
      "name": "net_listening",
      "summary": "Returns listening status.",
      "description": "Determines if this client is listening for new network connections.",
      "params": [],
      "result": {
        "name": "netListeningResult",
        "description": "` + "`" + `true` + "`" + ` if listening is active or ` + "`" + `false` + "`" + ` if listening is not active",
        "schema": {
          "title": "isNetListening",
          "type": "boolean"
        }
      },
      "examples": [
        {
          "name": "netListeningTrueExample",
          "description": "example of true result for net_listening",
          "params": [],
          "result": {
            "name": "netListeningExampleFalseResult",
            "value": true
          }
        }
      ]
    },
    {
      "name": "net_peerCount",
      "summary": "Returns the number of peers currently connected to this client.",
      "params": [],
      "result": {
        "name": "quantity",
        "schema": {
          "$ref": "#/components/schemas/Integer"
        }
      }
    },
    {
      "name": "net_version",
      "summary": "Returns the network ID associated with the current network.",
      "params": [],
      "result": {
        "name": "networkID",
        "description": "Network ID associated with the current network",
        "schema": {
          "title": "networkID",
          "type": "string",
          "pattern": "^[\\d]+$"
        }
      }
    },
    {
      "name": "eth_blockNumber",
      "summary": "Returns the number of most recent block.",
      "params": [],
      "result": {
        "name": "blockNumber",
        "schema": {
          "$ref": "#/components/schemas/BlockNumber"
        }
      }
    },
    {
      "name": "eth_call",
      "summary": "Executes a new message call (locally) immediately without creating a transaction on the block chain.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/Transaction"
        },
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        }
      ],
      "result": {
        "name": "returnValue",
        "description": "The return value of the executed contract",
        "schema": {
          "$ref": "#/components/schemas/Bytes"
        }
      }
    },
    {
      "name": "eth_chainId",
      "summary": "Returns the currently configured chain id.",
      "description": "Returns the currently configured chain id, a value used in replay-protected transaction signing as introduced by [EIP-155](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md).",
      "params": [],
      "result": {
        "name": "chainId",
        "description": "hex format integer of the current chain id. Defaults are UBQ=8, ETC=61, ETH=1",
        "schema": {
          "title": "chainId",
          "type": "string",
          "pattern": "^0x[a-fA-F\\d]+$"
        }
      }
    },
    {
      "name": "eth_coinbase",
      "summary": "Returns the client coinbase address.",
      "params": [],
      "result": {
        "name": "address",
        "description": "The address owned by the client that is used as default for things like the mining reward",
        "schema": {
          "$ref": "#/components/schemas/Address"
        }
      }
    },
    {
      "name": "eth_estimateGas",
      "summary": "Generates and returns an estimate of how much gas is necessary to allow the transaction to complete.",
      "description": "Generates and returns an estimate of how much gas is necessary to allow the transaction to complete. The transaction will not be added to the blockchain. Note that the estimate may be significantly more than the amount of gas actually used by the transaction, for a variety of reasons including EVM mechanics and node performance.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/Transaction"
        }
      ],
      "result": {
        "name": "gasUsed",
        "description": "The amount of gas used",
        "schema": {
          "$ref": "#/components/schemas/Integer"
        }
      }
    },
    {
      "name": "eth_gasPrice",
      "summary": "Returns the current price per gas in wei.",
      "params": [],
      "result": {
        "$ref": "#/components/contentDescriptors/GasPrice"
      }
    },
    {
      "name": "eth_getBalance",
      "summary": "Returns the UBQ balance of a given account or contract in wei.",
      "params": [
        {
          "name": "address",
          "required": true,
          "description": "The address of the account or contract",
          "schema": {
            "$ref": "#/components/schemas/Address"
          }
        },
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        }
      ],
      "result": {
        "name": "getBalanceResult",
        "schema": {
          "title": "getBalanceResult",
          "oneOf": [
            {
              "$ref": "#/components/schemas/Integer"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      }
    },
    {
      "name": "eth_getBlockByHash",
      "summary": "Gets a block for a given hash.",
      "params": [
        {
          "name": "blockHash",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/BlockHash"
          }
        },
        {
          "name": "includeTransactions",
          "description": "If ` + "`" + `true` + "`" + ` it returns the full transaction objects, if ` + "`" + `false` + "`" + ` only the hashes of the transactions.",
          "required": true,
          "schema": {
            "title": "isTransactionsIncluded",
            "type": "boolean"
          }
        }
      ],
      "result": {
        "name": "getBlockByHashResult",
        "schema": {
          "title": "getBlockByHashResult",
          "oneOf": [
            {
              "$ref": "#/components/schemas/Block"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      }
    },
    {
      "name": "eth_getBlockByNumber",
      "summary": "Gets a block for a given number salad.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        },
        {
          "name": "includeTransactions",
          "description": "If ` + "`" + `true` + "`" + ` it returns the full transaction objects, if ` + "`" + `false` + "`" + ` only the hashes of the transactions.",
          "required": true,
          "schema": {
            "title": "isTransactionsIncluded",
            "type": "boolean"
          }
        }
      ],
      "result": {
        "name": "getBlockByNumberResult",
        "schema": {
          "title": "getBlockByNumberResult",
          "oneOf": [
            {
              "$ref": "#/components/schemas/Block"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      }
    },
    {
      "name": "eth_getBlockTransactionCountByHash",
      "summary": "Returns the number of transactions in a block from a block matching the given block hash.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/BlockHash"
        }
      ],
      "result": {
        "name": "blockTransactionCountByHash",
        "description": "The Number of total transactions in the given block",
        "schema": {
          "title": "blockTransactionCountByHash",
          "oneOf": [
            {
              "$ref": "#/components/schemas/Integer"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      }
    },
    {
      "name": "eth_getBlockTransactionCountByNumber",
      "summary": "Returns the number of transactions in a block from a block matching the given block number.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        }
      ],
      "result": {
        "name": "blockTransactionCountByHash",
        "description": "The Number of total transactions in the given block",
        "schema": {
          "title": "blockTransactionCountByHash",
          "oneOf": [
            {
              "$ref": "#/components/schemas/Integer"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      }
    },
    {
      "name": "eth_getCode",
      "summary": "Returns code at a given contract address",
      "params": [
        {
          "name": "address",
          "required": true,
          "description": "The address of the contract",
          "schema": {
            "$ref": "#/components/schemas/Address"
          }
        },
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        }
      ],
      "result": {
        "name": "bytes",
        "schema": {
          "$ref": "#/components/schemas/Bytes"
        }
      }
    },
    {
      "name": "eth_getFilterChanges",
      "summary": "Polling method for a filter, which returns an array of logs which occurred since last poll.",
      "params": [
        {
          "name": "filterId",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/FilterId"
          }
        }
      ],
      "result": {
        "name": "logResult",
        "schema": {
          "title": "logResult",
          "type": "array",
          "items": {
            "$ref": "#/components/schemas/Log"
          }
        }
      }
    },
    {
      "name": "eth_getFilterLogs",
      "summary": "Returns an array of all logs matching filter with given id.",
      "params": [
        {
          "name": "filterId",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/FilterId"
          }
        }
      ],
      "result": {
        "$ref": "#/components/contentDescriptors/Logs"
      }
    },
    {
      "name": "eth_getRawTransactionByHash",
      "summary": "Returns raw transaction data of a transaction with the given hash.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/TransactionHash"
        }
      ],
      "result": {
        "name": "rawTransactionByHash",
        "description": "The raw transaction data",
        "schema": {
          "$ref": "#/components/schemas/Bytes"
        }
      }
    },
    {
      "name": "eth_getRawTransactionByBlockHashAndIndex",
      "summary": "Returns raw transaction data of a transaction with the given hash.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/BlockHash"
        },
        {
          "name": "index",
          "description": "The ordering in which a transaction is mined within its block.",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/Integer"
          }
        }
      ],
      "result": {
        "name": "rawTransaction",
        "description": "The raw transaction data",
        "schema": {
          "$ref": "#/components/schemas/Bytes"
        }
      }
    },
    {
      "name": "eth_getRawTransactionByBlockNumberAndIndex",
      "summary": "Returns raw transaction data of a transaction with the given hash.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        },
        {
          "name": "index",
          "description": "The ordering in which a transaction is mined within its block.",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/Integer"
          }
        }
      ],
      "result": {
        "name": "rawTransaction",
        "description": "The raw transaction data",
        "schema": {
          "$ref": "#/components/schemas/Bytes"
        }
      }
    },
    {
      "name": "eth_getLogs",
      "summary": "Returns an array of all logs matching a given filter object.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/Filter"
        }
      ],
      "result": {
        "$ref": "#/components/contentDescriptors/Logs"
      }
    },
    {
      "name": "eth_getStorageAt",
      "summary": "Gets a storage value from a contract address, a position, and an optional blockNumber",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/Address"
        },
        {
          "$ref": "#/components/contentDescriptors/Position"
        },
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        }
      ],
      "result": {
        "name": "dataWord",
        "schema": {
          "$ref": "#/components/schemas/DataWord"
        }
      }
    },
    {
      "name": "eth_getTransactionByBlockHashAndIndex",
      "summary": "Returns the information about a transaction requested by the block hash and index of which it was mined.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/BlockHash"
        },
        {
          "name": "index",
          "description": "The ordering in which a transaction is mined within its block.",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/Integer"
          }
        }
      ],
      "result": {
        "$ref": "#/components/contentDescriptors/TransactionResult"
      },
      "examples": [
        {
          "name": "nullExample",
          "params": [
            {
              "name": "blockHashExample",
              "value": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
            },
            {
              "name": "indexExample",
              "value": "0x0"
            }
          ],
          "result": {
            "name": "nullResultExample",
            "value": null
          }
        }
      ]
    },
    {
      "name": "eth_getTransactionByBlockNumberAndIndex",
      "summary": "Returns the information about a transaction requested by the block hash and index of which it was mined.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        },
        {
          "name": "index",
          "description": "The ordering in which a transaction is mined within its block.",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/Integer"
          }
        }
      ],
      "result": {
        "$ref": "#/components/contentDescriptors/TransactionResult"
      }
    },
    {
      "name": "eth_getTransactionByHash",
      "summary": "Returns the information about a transaction requested by transaction hash.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/TransactionHash"
        }
      ],
      "result": {
        "$ref": "#/components/contentDescriptors/TransactionResult"
      }
    },
    {
      "name": "eth_getTransactionCount",
      "summary": "Returns the number of transactions sent from an address",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/Address"
        },
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        }
      ],
      "result": {
        "name": "transactionCount",
        "schema": {
          "$ref": "#/components/schemas/Integer"
        }
      }
    },
    {
      "name": "eth_getTransactionReceipt",
      "summary": "Returns the receipt information of a transaction by its hash.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/TransactionHash"
        }
      ],
      "result": {
        "name": "transactionReceiptResult",
        "description": "returns either a receipt or null",
        "schema": {
          "title": "transactionReceiptOrNull",
          "oneOf": [
            {
              "$ref": "#/components/schemas/Receipt"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      }
    },
    {
      "name": "eth_getUncleByBlockHashAndIndex",
      "summary": "Returns information about a uncle of a block by hash and uncle index position.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/BlockHash"
        },
        {
          "name": "index",
          "description": "The ordering in which a uncle is included within its block.",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/Integer"
          }
        }
      ],
      "result": {
        "name": "uncle",
        "schema": {
          "title": "uncleOrNull",
          "oneOf": [
            {
              "$ref": "#/components/schemas/Uncle"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      }
    },
    {
      "name": "eth_getUncleByBlockNumberAndIndex",
      "summary": "Returns information about a uncle of a block by hash and uncle index position.",
      "params": [
        {
          "name": "uncleBlockNumber",
          "description": "The block in which the uncle was included",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/BlockNumber"
          }
        },
        {
          "name": "index",
          "description": "The ordering in which a uncle is included within its block.",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/Integer"
          }
        }
      ],
      "result": {
        "name": "uncleResult",
        "description": "returns an uncle or null",
        "schema": {
          "oneOf": [
            {
              "$ref": "#/components/schemas/Uncle"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      },
      "examples": [
        {
          "name": "nullResultExample",
          "params": [
            {
              "name": "uncleBlockNumberExample",
              "value": "0x0"
            },
            {
              "name": "uncleBlockNumberIndexExample",
              "value": "0x0"
            }
          ],
          "result": {
            "name": "nullResultExample",
            "value": null
          }
        }
      ]
    },
    {
      "name": "eth_getUncleCountByBlockHash",
      "summary": "Returns the number of uncles in a block from a block matching the given block hash.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/BlockHash"
        }
      ],
      "result": {
        "name": "uncleCountResult",
        "schema": {
          "title": "uncleCountOrNull",
          "oneOf": [
            {
              "description": "The Number of total uncles in the given block",
              "$ref": "#/components/schemas/Integer"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      }
    },
    {
      "name": "eth_getUncleCountByBlockNumber",
      "summary": "Returns the number of uncles in a block from a block matching the given block number.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        }
      ],
      "result": {
        "name": "uncleCountResult",
        "schema": {
          "title": "uncleCountOrNull",
          "oneOf": [
            {
              "description": "The Number of total uncles in the given block",
              "$ref": "#/components/schemas/Integer"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      }
    },
    {
      "name": "eth_getProof",
      "summary": "Returns the account- and storage-values of the specified account including the Merkle-proof.",
      "params": [
        {
          "name": "address",
          "description": "The address of the account or contract",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/Address"
          }
        },
        {
          "name": "storageKeys",
          "required": true,
          "schema": {
            "title": "storageKeys",
            "description": "The storage keys of all the storage slots being requested",
            "items": {
              "description": "A storage key is indexed from the solidity compiler by the order it is declared. For mappings it uses the keccak of the mapping key with its position (and recursively for X-dimensional mappings)",
              "$ref": "#/components/schemas/Integer"
            }
          }
        },
        {
          "$ref": "#/components/contentDescriptors/BlockNumber"
        }
      ],
      "result": {
        "name": "account",
        "schema": {
          "title": "proofAccountOrNull",
          "oneOf": [
            {
              "title": "proofAccount",
              "type": "object",
              "description": "The merkle proofs of the specified account connecting them to the blockhash of the block specified",
              "properties": {
                "address": {
                  "description": "The address of the account or contract of the request",
                  "$ref": "#/components/schemas/Address"
                },
                "accountProof": {
                  "$ref": "#/components/schemas/AccountProof"
                },
                "balance": {
                  "description": "The Ubiq balance of the account or contract of the request",
                  "$ref": "#/components/schemas/Integer"
                },
                "codeHash": {
                  "description": "The code hash of the contract of the request (keccak(NULL) if external account)",
                  "$ref": "#/components/schemas/Keccak"
                },
                "nonce": {
                  "description": "The transaction count of the account or contract of the request",
                  "$ref": "#/components/schemas/Nonce"
                },
                "storageHash": {
                  "description": "The storage hash of the contract of the request (keccak(rlp(NULL)) if external account)",
                  "$ref": "#/components/schemas/Keccak"
                },
                "storageProof": {
                  "$ref": "#/components/schemas/StorageProof"
                }
              }
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      }
    },
    {
      "name": "eth_getWork",
      "summary": "Returns the hash of the current block, the seedHash, and the boundary condition to be met ('target').",
      "params": [],
      "result": {
        "name": "work",
        "schema": {
          "type": "array",
          "items": [
            {
              "$ref": "#/components/schemas/PowHash"
            },
            {
              "$ref": "#/components/schemas/SeedHash"
            },
            {
              "$ref": "#/components/schemas/Difficulty"
            }
          ]
        }
      }
    },
    {
      "name": "eth_hashrate",
      "summary": "Returns the number of hashes per second that the node is mining with.",
      "params": [],
      "result": {
        "name": "hashesPerSecond",
        "schema": {
          "description": "Integer of the number of hashes per second",
          "$ref": "#/components/schemas/Integer"
        }
      }
    },
    {
      "name": "eth_mining",
      "summary": "Returns true if client is actively mining new blocks.",
      "params": [],
      "result": {
        "name": "mining",
        "schema": {
          "description": "Whether of not the client is mining",
          "type": "boolean"
        }
      }
    },
    {
      "name": "eth_newBlockFilter",
      "summary": "Creates a filter in the node, to notify when a new block arrives. To check if the state has changed, call eth_getFilterChanges.",
      "params": [],
      "result": {
        "$ref": "#/components/contentDescriptors/FilterId"
      }
    },
    {
      "name": "eth_newFilter",
      "summary": "Creates a filter object, based on filter options, to notify when the state changes (logs). To check if the state has changed, call eth_getFilterChanges.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/Filter"
        }
      ],
      "result": {
        "name": "filterId",
        "schema": {
          "description": "The filter ID for use in ` + "`" + `eth_getFilterChanges` + "`" + `",
          "$ref": "#/components/schemas/Integer"
        }
      }
    },
    {
      "name": "eth_newPendingTransactionFilter",
      "summary": "Creates a filter in the node, to notify when new pending transactions arrive. To check if the state has changed, call eth_getFilterChanges.",
      "params": [],
      "result": {
        "$ref": "#/components/contentDescriptors/FilterId"
      }
    },
    {
      "name": "eth_pendingTransactions",
      "summary": "Returns the pending transactions list.",
      "params": [],
      "result": {
        "name": "pendingTransactions",
        "schema": {
          "type": "array",
          "items": {
            "$ref": "#/components/schemas/Transaction"
          }
        }
      }
    },
    {
      "name": "eth_protocolVersion",
      "summary": "Returns the current ubiq protocol version.",
      "params": [],
      "result": {
        "name": "protocolVersion",
        "schema": {
          "description": "The current ubiq protocol version",
          "$ref": "#/components/schemas/Integer"
        }
      }
    },
    {
      "name": "eth_sendRawTransaction",
      "summary": "Creates new message call transaction or a contract creation for signed transactions.",
      "params": [
        {
          "name": "signedTransactionData",
          "required": true,
          "description": "The signed transaction data",
          "schema": {
            "$ref": "#/components/schemas/Bytes"
          }
        }
      ],
      "result": {
        "name": "transactionHash",
        "schema": {
          "description": "The transaction hash, or the zero hash if the transaction is not yet available.",
          "$ref": "#/components/schemas/Keccak"
        }
      }
    },
    {
      "name": "eth_submitHashrate",
      "summary": "Returns an array of all logs matching a given filter object.",
      "params": [
        {
          "name": "hashRate",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/DataWord"
          }
        },
        {
          "name": "id",
          "required": true,
          "description": "String identifying the client",
          "schema": {
            "$ref": "#/components/schemas/DataWord"
          }
        }
      ],
      "result": {
        "name": "submitHashRateSuccess",
        "schema": {
          "type": "boolean",
          "description": "whether of not submitting went through successfully"
        }
      }
    },
    {
      "name": "eth_submitWork",
      "summary": "Used for submitting a proof-of-work solution.",
      "params": [
        {
          "$ref": "#/components/contentDescriptors/Nonce"
        },
        {
          "name": "powHash",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/PowHash"
          }
        },
        {
          "name": "mixHash",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/MixHash"
          }
        }
      ],
      "result": {
        "name": "solutionValid",
        "description": "returns true if the provided solution is valid, otherwise false.",
        "schema": {
          "type": "boolean",
          "description": "Whether or not the provided solution is valid"
        }
      },
      "examples": [
        {
          "name": "submitWorkExample",
          "params": [
            {
              "name": "nonceExample",
              "description": "example of a number only used once",
              "value": "0x0000000000000001"
            },
            {
              "name": "powHashExample",
              "description": "proof of work to submit",
              "value": "0x6bf2cAE0dE3ec3ecA5E194a6C6e02cf42aADfe1C2c4Fff12E5D36C3Cf7297F22"
            },
            {
              "name": "mixHashExample",
              "description": "the mix digest example",
              "value": "0xD1FE5700000000000000000000000000D1FE5700000000000000000000000000"
            }
          ],
          "result": {
            "name": "solutionInvalidExample",
            "description": "this example should return ` + "`" + `false` + "`" + ` as it is not a valid pow to submit",
            "value": false
          }
        }
      ]
    },
    {
      "name": "eth_syncing",
      "summary": "Returns an object with data about the sync status or false.",
      "params": [],
      "result": {
        "name": "syncing",
        "schema": {
          "oneOf": [
            {
              "description": "An object with sync status data",
              "type": "object",
              "properties": {
                "startingBlock": {
                  "description": "Block at which the import started (will only be reset, after the sync reached his head)",
                  "$ref": "#/components/schemas/Integer"
                },
                "currentBlock": {
                  "description": "The current block, same as eth_blockNumber",
                  "$ref": "#/components/schemas/Integer"
                },
                "highestBlock": {
                  "description": "The estimated highest block",
                  "$ref": "#/components/schemas/Integer"
                },
                "knownStates": {
                  "description": "The known states",
                  "$ref": "#/components/schemas/Integer"
                },
                "pulledStates": {
                  "description": "The pulled states",
                  "$ref": "#/components/schemas/Integer"
                }
              }
            },
            {
              "type": "boolean",
              "description": "The value ` + "`" + `false` + "`" + ` indicating that syncing is complete"
            }
          ]
        }
      }
    },
    {
      "name": "eth_uninstallFilter",
      "summary": "Uninstalls a filter with given id. Should always be called when watch is no longer needed. Additionally Filters timeout when they aren't requested with eth_getFilterChanges for a period of time.",
      "params": [
        {
          "name": "filterId",
          "required": true,
          "schema": {
            "$ref": "#/components/schemas/FilterId"
          }
        }
      ],
      "result": {
        "name": "filterUninstalledSuccess",
        "schema": {
          "type": "boolean",
          "description": "Whether of not the filter was successfully uninstalled"
        }
      }
    }
  ],
  "components": {
    "schemas": {
      "ProofNode": {
        "type": "string",
        "description": "An individual node used to prove a path down a merkle-patricia-tree",
        "$ref": "#/components/schemas/Bytes"
      },
      "AccountProof": {
        "$ref": "#/components/schemas/ProofNodes"
      },
      "StorageProof": {
        "type": "array",
        "description": "Current block header PoW hash.",
        "items": {
          "type": "object",
          "description": "Object proving a relationship of a storage value to an account's storageHash.",
          "properties": {
            "key": {
              "description": "The key used to get the storage slot in its account tree",
              "$ref": "#/components/schemas/Integer"
            },
            "value": {
              "description": "The value of the storage slot in its account tree",
              "$ref": "#/components/schemas/Integer"
            },
            "proof": {
              "$ref": "#/components/schemas/ProofNodes"
            }
          }
        }
      },
      "ProofNodes": {
        "type": "array",
        "description": "The set of node values needed to traverse a patricia merkle tree (from root to leaf) to retrieve a value",
        "items": {
          "$ref": "#/components/schemas/ProofNode"
        }
      },
      "PowHash": {
        "description": "Current block header PoW hash.",
        "$ref": "#/components/schemas/DataWord"
      },
      "SeedHash": {
        "description": "The seed hash used for the DAG.",
        "$ref": "#/components/schemas/DataWord"
      },
      "MixHash": {
        "description": "The mix digest.",
        "$ref": "#/components/schemas/DataWord"
      },
      "Difficulty": {
        "description": "The boundary condition ('target'), 2^256 / difficulty.",
        "$ref": "#/components/schemas/DataWord"
      },
      "FilterId": {
        "type": "string",
        "description": "An identifier used to reference the filter."
      },
      "BlockHash": {
        "type": "string",
        "pattern": "^0x[a-fA-F\\d]{64}$",
        "description": "The hex representation of the Keccak 256 of the RLP encoded block"
      },
      "BlockNumber": {
        "type": "string",
        "pattern": "^0x[a-fA-F\\d]+$",
        "description": "The hex representation of the block's height"
      },
      "BlockNumberTag": {
        "type": "string",
        "description": "The block's height description",
        "enum": [
          "earliest",
          "latest",
          "pending"
        ]
      },
      "Receipt": {
        "type": "object",
        "description": "The receipt of a transaction",
        "required": [
          "blockHash",
          "blockNumber",
          "contractAddress",
          "cumulativeGasUsed",
          "from",
          "gasUsed",
          "logs",
          "logsBloom",
          "to",
          "transactionHash",
          "transactionIndex"
        ],
        "properties": {
          "blockHash": {
            "description": "BlockHash of the block in which the transaction was mined",
            "$ref": "#/components/schemas/BlockHash"
          },
          "blockNumber": {
            "description": "BlockNumber of the block in which the transaction was mined",
            "$ref": "#/components/schemas/BlockNumber"
          },
          "contractAddress": {
            "description": "The contract address created, if the transaction was a contract creation, otherwise null",
            "$ref": "#/components/schemas/Address"
          },
          "cumulativeGasUsed": {
            "description": "The gas units used by the transaction",
            "$ref": "#/components/schemas/Integer"
          },
          "from": {
            "description": "The sender of the transaction",
            "$ref": "#/components/schemas/Address"
          },
          "gasUsed": {
            "description": "The total gas used by the transaction",
            "$ref": "#/components/schemas/Integer"
          },
          "logs": {
            "type": "array",
            "description": "An array of all the logs triggered during the transaction",
            "items": {
              "$ref": "#/components/schemas/Log"
            }
          },
          "logsBloom": {
            "$ref": "#/components/schemas/BloomFilter"
          },
          "to": {
            "description": "Destination address of the transaction",
            "$ref": "#/components/schemas/Address"
          },
          "transactionHash": {
            "description": "Keccak 256 of the transaction",
            "$ref": "#/components/schemas/Keccak"
          },
          "transactionIndex": {
            "description": "An array of all the logs triggered during the transaction",
            "$ref": "#/components/schemas/BloomFilter"
          },
          "postTransactionState": {
            "description": "The intermediate stateRoot directly after transaction execution.",
            "$ref": "#/components/schemas/Keccak"
          },
          "status": {
            "description": "Whether or not the transaction threw an error.",
            "type": "boolean"
          }
        }
      },
      "BloomFilter": {
        "type": "string",
        "description": "A 2048 bit bloom filter from the logs of the transaction. Each log sets 3 bits though taking the low-order 11 bits of each of the first three pairs of bytes in a Keccak 256 hash of the log's byte series"
      },
      "Log": {
        "type": "object",
        "description": "An indexed event generated during a transaction",
        "properties": {
          "address": {
            "description": "Sender of the transaction",
            "$ref": "#/components/schemas/Address"
          },
          "blockHash": {
            "description": "BlockHash of the block in which the transaction was mined",
            "$ref": "#/components/schemas/BlockHash"
          },
          "blockNumber": {
            "description": "BlockNumber of the block in which the transaction was mined",
            "$ref": "#/components/schemas/BlockNumber"
          },
          "data": {
            "description": "The data/input string sent along with the transaction",
            "$ref": "#/components/schemas/Bytes"
          },
          "logIndex": {
            "description": "The index of the event within its transaction, null when its pending",
            "$ref": "#/components/schemas/Integer"
          },
          "removed": {
            "schema": {
              "description": "Whether or not the log was orphaned off the main chain",
              "type": "boolean"
            }
          },
          "topics": {
            "type": "array",
            "items": {
              "topic": {
                "description": "32 Bytes DATA of indexed log arguments. (In solidity: The first topic is the hash of the signature of the event (e.g. Deposit(address,bytes32,uint256))",
                "$ref": "#/components/schemas/DataWord"
              }
            }
          },
          "transactionHash": {
            "description": "The hash of the transaction in which the log occurred",
            "$ref": "#/components/schemas/Keccak"
          },
          "transactionIndex": {
            "description": "The index of the transaction in which the log occurred",
            "$ref": "#/components/schemas/Integer"
          }
        }
      },
      "Uncle": {
        "type": "object",
        "description": "Orphaned blocks that can be included in the chain but at a lower block reward. NOTE: An uncle doesn’t contain individual transactions.",
        "properties": {
          "number": {
            "description": "The block number or null when its the pending block",
            "$ref": "#/components/schemas/IntOrPending"
          },
          "hash": {
            "description": "The block hash or null when its the pending block",
            "$ref": "#/components/schemas/KeccakOrPending"
          },
          "parentHash": {
            "description": "Hash of the parent block",
            "$ref": "#/components/schemas/Keccak"
          },
          "nonce": {
            "description": "Randomly selected number to satisfy the proof-of-work or null when its the pending block",
            "$ref": "#/components/schemas/IntOrPending"
          },
          "sha3Uncles": {
            "description": "Keccak hash of the uncles data in the block",
            "$ref": "#/components/schemas/Keccak"
          },
          "logsBloom": {
            "type": "string",
            "description": "The bloom filter for the logs of the block or null when its the pending block",
            "pattern": "^0x[a-fA-F\\d]+$"
          },
          "transactionsRoot": {
            "description": "The root of the transactions trie of the block.",
            "$ref": "#/components/schemas/Keccak"
          },
          "stateRoot": {
            "description": "The root of the final state trie of the block",
            "$ref": "#/components/schemas/Keccak"
          },
          "receiptsRoot": {
            "description": "The root of the receipts trie of the block",
            "$ref": "#/components/schemas/Keccak"
          },
          "miner": {
            "description": "The address of the beneficiary to whom the mining rewards were given or null when its the pending block",
            "oneOf": [
              {
                "$ref": "#/components/schemas/Address"
              },
              {
                "$ref": "#/components/schemas/Null"
              }
            ]
          },
          "difficulty": {
            "type": "string",
            "description": "Integer of the difficulty for this block"
          },
          "totalDifficulty": {
            "description": "Integer of the total difficulty of the chain until this block",
            "$ref": "#/components/schemas/IntOrPending"
          },
          "extraData": {
            "type": "string",
            "description": "The 'extra data' field of this block"
          },
          "size": {
            "type": "string",
            "description": "Integer the size of this block in bytes"
          },
          "gasLimit": {
            "type": "string",
            "description": "The maximum gas allowed in this block"
          },
          "gasUsed": {
            "type": "string",
            "description": "The total used gas by all transactions in this block"
          },
          "timestamp": {
            "type": "string",
            "description": "The unix timestamp for when the block was collated"
          },
          "uncles": {
            "description": "Array of uncle hashes",
            "type": "array",
            "items": {
              "description": "Block hash of the RLP encoding of an uncle block",
              "$ref": "#/components/schemas/Keccak"
            }
          }
        }
      },
      "Block": {
        "type": "object",
        "properties": {
          "number": {
            "description": "The block number or null when its the pending block",
            "$ref": "#/components/schemas/IntOrPending"
          },
          "hash": {
            "description": "The block hash or null when its the pending block",
            "$ref": "#/components/schemas/KeccakOrPending"
          },
          "parentHash": {
            "description": "Hash of the parent block",
            "$ref": "#/components/schemas/Keccak"
          },
          "nonce": {
            "description": "Randomly selected number to satisfy the proof-of-work or null when its the pending block",
            "$ref": "#/components/schemas/IntOrPending"
          },
          "sha3Uncles": {
            "description": "Keccak hash of the uncles data in the block",
            "$ref": "#/components/schemas/Keccak"
          },
          "logsBloom": {
            "type": "string",
            "description": "The bloom filter for the logs of the block or null when its the pending block",
            "pattern": "^0x[a-fA-F\\d]+$"
          },
          "transactionsRoot": {
            "description": "The root of the transactions trie of the block.",
            "$ref": "#/components/schemas/Keccak"
          },
          "stateRoot": {
            "description": "The root of the final state trie of the block",
            "$ref": "#/components/schemas/Keccak"
          },
          "receiptsRoot": {
            "description": "The root of the receipts trie of the block",
            "$ref": "#/components/schemas/Keccak"
          },
          "miner": {
            "description": "The address of the beneficiary to whom the mining rewards were given or null when its the pending block",
            "oneOf": [
              {
                "$ref": "#/components/schemas/Address"
              },
              {
                "$ref": "#/components/schemas/Null"
              }
            ]
          },
          "difficulty": {
            "type": "string",
            "description": "Integer of the difficulty for this block"
          },
          "totalDifficulty": {
            "description": "Integer of the total difficulty of the chain until this block",
            "$ref": "#/components/schemas/IntOrPending"
          },
          "extraData": {
            "type": "string",
            "description": "The 'extra data' field of this block"
          },
          "size": {
            "type": "string",
            "description": "Integer the size of this block in bytes"
          },
          "gasLimit": {
            "type": "string",
            "description": "The maximum gas allowed in this block"
          },
          "gasUsed": {
            "type": "string",
            "description": "The total used gas by all transactions in this block"
          },
          "timestamp": {
            "type": "string",
            "description": "The unix timestamp for when the block was collated"
          },
          "transactions": {
            "description": "Array of transaction objects, or 32 Bytes transaction hashes depending on the last given parameter",
            "type": "array",
            "items": {
              "oneOf": [
                {
                  "$ref": "#/components/schemas/Transaction"
                },
                {
                  "$ref": "#/components/schemas/TransactionHash"
                }
              ]
            }
          },
          "uncles": {
            "description": "Array of uncle hashes",
            "type": "array",
            "items": {
              "description": "Block hash of the RLP encoding of an uncle block",
              "$ref": "#/components/schemas/Keccak"
            }
          }
        }
      },
      "Transaction": {
        "type": "object",
        "required": [
          "gas",
          "gasPrice",
          "nonce"
        ],
        "properties": {
          "blockHash": {
            "description": "Hash of the block where this transaction was in. null when its pending",
            "$ref": "#/components/schemas/KeccakOrPending"
          },
          "blockNumber": {
            "description": "Block number where this transaction was in. null when its pending",
            "$ref": "#/components/schemas/IntOrPending"
          },
          "from": {
            "description": "Address of the sender",
            "$ref": "#/components/schemas/Address"
          },
          "gas": {
            "type": "string",
            "description": "The gas limit provided by the sender in Wei"
          },
          "gasPrice": {
            "type": "string",
            "description": "The gas price willing to be paid by the sender in Wei"
          },
          "hash": {
            "$ref": "#/components/schemas/TransactionHash"
          },
          "input": {
            "type": "string",
            "description": "The data field sent with the transaction"
          },
          "nonce": {
            "description": "The total number of prior transactions made by the sender",
            "$ref": "#/components/schemas/Nonce"
          },
          "to": {
            "description": "address of the receiver. null when its a contract creation transaction",
            "$ref": "#/components/schemas/Address"
          },
          "transactionIndex": {
            "description": "Integer of the transaction's index position in the block. null when its pending",
            "$ref": "#/components/schemas/IntOrPending"
          },
          "value": {
            "description": "Value of Ubiq being transferred in Wei",
            "$ref": "#/components/schemas/Keccak"
          },
          "v": {
            "type": "string",
            "description": "ECDSA recovery id"
          },
          "r": {
            "type": "string",
            "description": "ECDSA signature r"
          },
          "s": {
            "type": "string",
            "description": "ECDSA signature s"
          }
        }
      },
      "TransactionHash": {
        "type": "string",
        "description": "Keccak 256 Hash of the RLP encoding of a transaction",
        "$ref": "#/components/schemas/Keccak"
      },
      "KeccakOrPending": {
        "oneOf": [
          {
            "$ref": "#/components/schemas/Keccak"
          },
          {
            "$ref": "#/components/schemas/Null"
          }
        ]
      },
      "IntOrPending": {
        "oneOf": [
          {
            "$ref": "#/components/schemas/Integer"
          },
          {
            "$ref": "#/components/schemas/Null"
          }
        ]
      },
      "Keccak": {
        "type": "string",
        "description": "Hex representation of a Keccak 256 hash",
        "pattern": "^0x[a-fA-F\\d]{64}$"
      },
      "Nonce": {
        "description": "A number only to be used once",
        "pattern": "^0x[a-fA-F0-9]+$",
        "type": "string"
      },
      "Null": {
        "type": "null",
        "description": "Null"
      },
      "Integer": {
        "type": "string",
        "pattern": "^0x[a-fA-F0-9]+$",
        "description": "Hex representation of an integer"
      },
      "Address": {
        "type": "string",
        "pattern": "^0x[a-fA-F\\d]{40}$"
      },
      "Position": {
        "type": "string",
        "description": "Hex representation of the storage slot where the variable exists",
        "pattern": "^0x([a-fA-F0-9]?)+$"
      },
      "DataWord": {
        "type": "string",
        "description": "Hex representation of a 256 bit unit of data",
        "pattern": "^0x([a-fA-F\\d]{64})?$"
      },
      "Bytes": {
        "type": "string",
        "description": "Hex representation of a variable length byte array",
        "pattern": "^0x([a-fA-F0-9]?)+$"
      }
    },
    "contentDescriptors": {
      "Block": {
        "name": "block",
        "summary": "A block",
        "description": "A block object",
        "schema": {
          "$ref": "#/components/schemas/Block"
        }
      },
      "Null": {
        "name": "Null",
        "description": "JSON Null value",
        "summary": "Null value",
        "schema": {
          "type": "null",
          "description": "Null value"
        }
      },
      "Signature": {
        "name": "signature",
        "summary": "The signature.",
        "required": true,
        "schema": {
          "$ref": "#/components/schemas/Bytes",
          "pattern": "0x^([A-Fa-f0-9]{2}){65}$"
        }
      },
      "GasPrice": {
        "name": "gasPrice",
        "required": true,
        "schema": {
          "description": "Integer of the current gas price",
          "$ref": "#/components/schemas/Integer"
        }
      },
      "Transaction": {
        "required": true,
        "name": "transaction",
        "schema": {
          "$ref": "#/components/schemas/Transaction"
        }
      },
      "TransactionResult": {
        "name": "transactionResult",
        "description": "Returns a transaction or null",
        "schema": {
          "oneOf": [
            {
              "$ref": "#/components/schemas/Transaction"
            },
            {
              "$ref": "#/components/schemas/Null"
            }
          ]
        }
      },
      "Message": {
        "name": "message",
        "required": true,
        "schema": {
          "$ref": "#/components/schemas/Bytes"
        }
      },
      "Filter": {
        "name": "filter",
        "required": true,
        "schema": {
          "type": "object",
          "description": "A filter used to monitor the blockchain for log/events",
          "properties": {
            "fromBlock": {
              "description": "Block from which to begin filtering events",
              "$ref": "#/components/schemas/BlockNumber"
            },
            "toBlock": {
              "description": "Block from which to end filtering events",
              "$ref": "#/components/schemas/BlockNumber"
            },
            "address": {
              "oneOf": [
                {
                  "type": "string",
                  "description": "Address of the contract from which to monitor events",
                  "$ref": "#/components/schemas/Address"
                },
                {
                  "type": "array",
                  "description": "List of contract addresses from which to monitor events",
                  "items": {
                    "$ref": "#/components/schemas/Address"
                  }
                }
              ]
            },
            "topics": {
              "type": "array",
              "description": "Array of 32 Bytes DATA topics. Topics are order-dependent. Each topic can also be an array of DATA with 'or' options",
              "items": {
                "description": "Indexable 32 bytes piece of data (made from the event's function signature in solidity)",
                "$ref": "#/components/schemas/DataWord"
              }
            }
          }
        }
      },
      "Address": {
        "name": "address",
        "required": true,
        "schema": {
          "$ref": "#/components/schemas/Address"
        }
      },
      "BlockHash": {
        "name": "blockHash",
        "required": true,
        "schema": {
          "$ref": "#/components/schemas/BlockHash"
        }
      },
      "Nonce": {
        "name": "nonce",
        "required": true,
        "schema": {
          "$ref": "#/components/schemas/Nonce"
        }
      },
      "Position": {
        "name": "key",
        "required": true,
        "schema": {
          "$ref": "#/components/schemas/Position"
        }
      },
      "Logs": {
        "name": "logs",
        "description": "An array of all logs matching filter with given id.",
        "schema": {
          "type": "array",
          "items": {
            "$ref": "#/components/schemas/Log"
          }
        }
      },
      "FilterId": {
        "name": "filterId",
        "schema": {
          "description": "The filter ID for use in ` + "`" + `eth_getFilterChanges` + "`" + `",
          "$ref": "#/components/schemas/Integer"
        }
      },
      "BlockNumber": {
        "name": "blockNumber",
        "required": true,
        "schema": {
          "oneOf": [
            {
              "$ref": "#/components/schemas/BlockNumber"
            },
            {
              "$ref": "#/components/schemas/BlockNumberTag"
            }
          ]
        }
      },
      "TransactionHash": {
        "name": "transactionHash",
        "required": true,
        "schema": {
          "$ref": "#/components/schemas/TransactionHash"
        }
      }
    }
  }
}

`
