import "../chunk-NHABU752.mjs";

// src/test/abis.ts
var customSolidityErrorsAbi = [
  { inputs: [], stateMutability: "nonpayable", type: "constructor" },
  { inputs: [], name: "ApprovalCallerNotOwnerNorApproved", type: "error" },
  { inputs: [], name: "ApprovalQueryForNonexistentToken", type: "error" }
];
var ensAbi = [
  {
    constant: true,
    inputs: [{ name: "node", type: "bytes32" }],
    name: "resolver",
    outputs: [{ type: "address" }],
    payable: false,
    type: "function"
  },
  {
    constant: true,
    inputs: [{ name: "node", type: "bytes32" }],
    name: "owner",
    outputs: [{ type: "address" }],
    payable: false,
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "node", type: "bytes32" },
      { name: "label", type: "bytes32" },
      { name: "owner", type: "address" }
    ],
    name: "setSubnodeOwner",
    outputs: [],
    payable: false,
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "node", type: "bytes32" },
      { name: "ttl", type: "uint64" }
    ],
    name: "setTTL",
    outputs: [],
    payable: false,
    type: "function"
  },
  {
    constant: true,
    inputs: [{ name: "node", type: "bytes32" }],
    name: "ttl",
    outputs: [{ type: "uint64" }],
    payable: false,
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "node", type: "bytes32" },
      { name: "resolver", type: "address" }
    ],
    name: "setResolver",
    outputs: [],
    payable: false,
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "node", type: "bytes32" },
      { name: "owner", type: "address" }
    ],
    name: "setOwner",
    outputs: [],
    payable: false,
    type: "function"
  },
  {
    inputs: [
      { indexed: true, name: "node", type: "bytes32" },
      { indexed: false, name: "owner", type: "address" }
    ],
    name: "Transfer",
    type: "event"
  },
  {
    inputs: [
      { indexed: true, name: "node", type: "bytes32" },
      { indexed: true, name: "label", type: "bytes32" },
      { indexed: false, name: "owner", type: "address" }
    ],
    name: "NewOwner",
    type: "event"
  },
  {
    inputs: [
      { indexed: true, name: "node", type: "bytes32" },
      { indexed: false, name: "resolver", type: "address" }
    ],
    name: "NewResolver",
    type: "event"
  },
  {
    inputs: [
      { indexed: true, name: "node", type: "bytes32" },
      { indexed: false, name: "ttl", type: "uint64" }
    ],
    name: "NewTTL",
    type: "event"
  }
];
var ensRegistryWithFallbackAbi = [
  {
    inputs: [{ internalType: "contract ENS", name: "_old", type: "address" }],
    payable: false,
    stateMutability: "nonpayable",
    type: "constructor"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "owner",
        type: "address"
      },
      {
        indexed: true,
        name: "operator",
        type: "address"
      },
      { indexed: false, name: "approved", type: "bool" }
    ],
    name: "ApprovalForAll",
    type: "event"
  },
  {
    inputs: [
      { indexed: true, name: "node", type: "bytes32" },
      {
        indexed: true,
        name: "label",
        type: "bytes32"
      },
      {
        indexed: false,
        name: "owner",
        type: "address"
      }
    ],
    name: "NewOwner",
    type: "event"
  },
  {
    inputs: [
      { indexed: true, name: "node", type: "bytes32" },
      {
        indexed: false,
        name: "resolver",
        type: "address"
      }
    ],
    name: "NewResolver",
    type: "event"
  },
  {
    inputs: [
      { indexed: true, name: "node", type: "bytes32" },
      { indexed: false, name: "ttl", type: "uint64" }
    ],
    name: "NewTTL",
    type: "event"
  },
  {
    inputs: [
      { indexed: true, name: "node", type: "bytes32" },
      {
        indexed: false,
        name: "owner",
        type: "address"
      }
    ],
    name: "Transfer",
    type: "event"
  },
  {
    constant: true,
    inputs: [
      { name: "owner", type: "address" },
      { name: "operator", type: "address" }
    ],
    name: "isApprovedForAll",
    outputs: [{ type: "bool" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  {
    constant: true,
    inputs: [],
    name: "old",
    outputs: [{ internalType: "contract ENS", type: "address" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  {
    constant: true,
    inputs: [{ name: "node", type: "bytes32" }],
    name: "owner",
    outputs: [{ type: "address" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  {
    constant: true,
    inputs: [{ name: "node", type: "bytes32" }],
    name: "recordExists",
    outputs: [{ type: "bool" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  {
    constant: true,
    inputs: [{ name: "node", type: "bytes32" }],
    name: "resolver",
    outputs: [{ type: "address" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "operator", type: "address" },
      { name: "approved", type: "bool" }
    ],
    name: "setApprovalForAll",
    outputs: [],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "node", type: "bytes32" },
      { name: "owner", type: "address" }
    ],
    name: "setOwner",
    outputs: [],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "node", type: "bytes32" },
      { name: "owner", type: "address" },
      { name: "resolver", type: "address" },
      { name: "ttl", type: "uint64" }
    ],
    name: "setRecord",
    outputs: [],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "node", type: "bytes32" },
      { name: "resolver", type: "address" }
    ],
    name: "setResolver",
    outputs: [],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "node", type: "bytes32" },
      { name: "label", type: "bytes32" },
      { name: "owner", type: "address" }
    ],
    name: "setSubnodeOwner",
    outputs: [{ type: "bytes32" }],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "node", type: "bytes32" },
      { name: "label", type: "bytes32" },
      { name: "owner", type: "address" },
      { name: "resolver", type: "address" },
      { name: "ttl", type: "uint64" }
    ],
    name: "setSubnodeRecord",
    outputs: [],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "node", type: "bytes32" },
      { name: "ttl", type: "uint64" }
    ],
    name: "setTTL",
    outputs: [],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: true,
    inputs: [{ name: "node", type: "bytes32" }],
    name: "ttl",
    outputs: [{ type: "uint64" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  }
];
var erc20Abi = [
  {
    type: "event",
    name: "Approval",
    inputs: [
      {
        indexed: true,
        name: "owner",
        type: "address"
      },
      {
        indexed: true,
        name: "spender",
        type: "address"
      },
      {
        indexed: false,
        name: "value",
        type: "uint256"
      }
    ]
  },
  {
    type: "event",
    name: "Transfer",
    inputs: [
      {
        indexed: true,
        name: "from",
        type: "address"
      },
      {
        indexed: true,
        name: "to",
        type: "address"
      },
      {
        indexed: false,
        name: "value",
        type: "uint256"
      }
    ]
  },
  {
    type: "function",
    name: "allowance",
    stateMutability: "view",
    inputs: [
      {
        name: "owner",
        type: "address"
      },
      {
        name: "spender",
        type: "address"
      }
    ],
    outputs: [
      {
        name: "",
        type: "uint256"
      }
    ]
  },
  {
    type: "function",
    name: "approve",
    stateMutability: "nonpayable",
    inputs: [
      {
        name: "spender",
        type: "address"
      },
      {
        name: "amount",
        type: "uint256"
      }
    ],
    outputs: [
      {
        name: "",
        type: "bool"
      }
    ]
  },
  {
    type: "function",
    name: "balanceOf",
    stateMutability: "view",
    inputs: [
      {
        name: "account",
        type: "address"
      }
    ],
    outputs: [
      {
        name: "",
        type: "uint256"
      }
    ]
  },
  {
    type: "function",
    name: "decimals",
    stateMutability: "view",
    inputs: [],
    outputs: [
      {
        name: "",
        type: "uint8"
      }
    ]
  },
  {
    type: "function",
    name: "name",
    stateMutability: "view",
    inputs: [],
    outputs: [
      {
        name: "",
        type: "string"
      }
    ]
  },
  {
    type: "function",
    name: "symbol",
    stateMutability: "view",
    inputs: [],
    outputs: [
      {
        name: "",
        type: "string"
      }
    ]
  },
  {
    type: "function",
    name: "totalSupply",
    stateMutability: "view",
    inputs: [],
    outputs: [
      {
        name: "",
        type: "uint256"
      }
    ]
  },
  {
    type: "function",
    name: "transfer",
    stateMutability: "nonpayable",
    inputs: [
      {
        name: "recipient",
        type: "address"
      },
      {
        name: "amount",
        type: "uint256"
      }
    ],
    outputs: [
      {
        name: "",
        type: "bool"
      }
    ]
  },
  {
    type: "function",
    name: "transferFrom",
    stateMutability: "nonpayable",
    inputs: [
      {
        name: "sender",
        type: "address"
      },
      {
        name: "recipient",
        type: "address"
      },
      {
        name: "amount",
        type: "uint256"
      }
    ],
    outputs: [
      {
        name: "",
        type: "bool"
      }
    ]
  }
];
var nestedTupleArrayAbi = [
  {
    inputs: [
      {
        name: "s",
        type: "tuple",
        components: [
          {
            name: "a",
            type: "uint8"
          },
          {
            name: "b",
            type: "uint8[]"
          },
          {
            name: "c",
            type: "tuple[]",
            components: [
              {
                name: "x",
                type: "uint8"
              },
              {
                name: "y",
                type: "uint8"
              }
            ]
          }
        ]
      },
      {
        name: "t",
        type: "tuple",
        components: [
          {
            name: "x",
            type: "uint"
          },
          {
            name: "y",
            type: "uint"
          }
        ]
      },
      {
        name: "a",
        type: "uint256"
      }
    ],
    name: "f",
    outputs: [
      {
        name: "t",
        type: "tuple[]",
        components: [
          {
            name: "x",
            type: "uint256"
          },
          {
            name: "y",
            type: "uint256"
          }
        ]
      }
    ],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [
      {
        name: "s",
        type: "tuple[2]",
        components: [
          {
            name: "a",
            type: "uint8"
          },
          {
            name: "b",
            type: "uint8[]"
          }
        ]
      },
      {
        name: "t",
        type: "tuple",
        components: [
          {
            name: "x",
            type: "uint"
          },
          {
            name: "y",
            type: "uint"
          }
        ]
      },
      {
        name: "a",
        type: "uint256"
      }
    ],
    name: "v",
    outputs: [],
    stateMutability: "view",
    type: "function"
  }
];
var nounsAuctionHouseAbi = [
  {
    inputs: [
      {
        indexed: true,
        name: "nounId",
        type: "uint256"
      },
      {
        indexed: false,
        name: "sender",
        type: "address"
      },
      {
        indexed: false,
        name: "value",
        type: "uint256"
      },
      { indexed: false, name: "extended", type: "bool" }
    ],
    name: "AuctionBid",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "nounId",
        type: "uint256"
      },
      {
        indexed: false,
        name: "startTime",
        type: "uint256"
      },
      {
        indexed: false,
        name: "endTime",
        type: "uint256"
      }
    ],
    name: "AuctionCreated",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "nounId",
        type: "uint256"
      },
      {
        indexed: false,
        name: "endTime",
        type: "uint256"
      }
    ],
    name: "AuctionExtended",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: false,
        name: "minBidIncrementPercentage",
        type: "uint256"
      }
    ],
    name: "AuctionMinBidIncrementPercentageUpdated",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: false,
        name: "reservePrice",
        type: "uint256"
      }
    ],
    name: "AuctionReservePriceUpdated",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "nounId",
        type: "uint256"
      },
      {
        indexed: false,
        name: "winner",
        type: "address"
      },
      {
        indexed: false,
        name: "amount",
        type: "uint256"
      }
    ],
    name: "AuctionSettled",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: false,
        name: "timeBuffer",
        type: "uint256"
      }
    ],
    name: "AuctionTimeBufferUpdated",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "previousOwner",
        type: "address"
      },
      {
        indexed: true,
        name: "newOwner",
        type: "address"
      }
    ],
    name: "OwnershipTransferred",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: false,
        name: "account",
        type: "address"
      }
    ],
    name: "Paused",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: false,
        name: "account",
        type: "address"
      }
    ],
    name: "Unpaused",
    type: "event"
  },
  {
    inputs: [],
    name: "auction",
    outputs: [
      { name: "nounId", type: "uint256" },
      { name: "amount", type: "uint256" },
      { name: "startTime", type: "uint256" },
      { name: "endTime", type: "uint256" },
      { internalType: "address payable", name: "bidder", type: "address" },
      { name: "settled", type: "bool" }
    ],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [{ name: "nounId", type: "uint256" }],
    name: "createBid",
    outputs: [],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [],
    name: "duration",
    outputs: [{ type: "uint256" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      { internalType: "contract INounsToken", name: "_nouns", type: "address" },
      { name: "_weth", type: "address" },
      { name: "_timeBuffer", type: "uint256" },
      { name: "_reservePrice", type: "uint256" },
      {
        name: "_minBidIncrementPercentage",
        type: "uint8"
      },
      { name: "_duration", type: "uint256" }
    ],
    name: "initialize",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "minBidIncrementPercentage",
    outputs: [{ type: "uint8" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "nouns",
    outputs: [{ internalType: "contract INounsToken", type: "address" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "owner",
    outputs: [{ type: "address" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "pause",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "paused",
    outputs: [{ type: "bool" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "renounceOwnership",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "reservePrice",
    outputs: [{ type: "uint256" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      {
        name: "_minBidIncrementPercentage",
        type: "uint8"
      }
    ],
    name: "setMinBidIncrementPercentage",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [{ name: "_reservePrice", type: "uint256" }],
    name: "setReservePrice",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [{ name: "_timeBuffer", type: "uint256" }],
    name: "setTimeBuffer",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "settleAuction",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "settleCurrentAndCreateNewAuction",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "timeBuffer",
    outputs: [{ type: "uint256" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [{ name: "newOwner", type: "address" }],
    name: "transferOwnership",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "unpause",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "weth",
    outputs: [{ type: "address" }],
    stateMutability: "view",
    type: "function"
  }
];
var seaportAbi = [
  {
    inputs: [{ name: "conduitController", type: "address" }],
    stateMutability: "nonpayable",
    type: "constructor"
  },
  {
    inputs: [
      {
        components: [
          { name: "offerer", type: "address" },
          { name: "zone", type: "address" },
          {
            components: [
              { name: "itemType", type: "uint8" },
              { name: "token", type: "address" },
              {
                name: "identifierOrCriteria",
                type: "uint256"
              },
              { name: "startAmount", type: "uint256" },
              { name: "endAmount", type: "uint256" }
            ],
            name: "offer",
            type: "tuple[]"
          },
          {
            components: [
              { name: "itemType", type: "uint8" },
              { name: "token", type: "address" },
              {
                name: "identifierOrCriteria",
                type: "uint256"
              },
              { name: "startAmount", type: "uint256" },
              { name: "endAmount", type: "uint256" },
              {
                name: "recipient",
                type: "address"
              }
            ],
            name: "consideration",
            type: "tuple[]"
          },
          { name: "orderType", type: "uint8" },
          { name: "startTime", type: "uint256" },
          { name: "endTime", type: "uint256" },
          { name: "zoneHash", type: "bytes32" },
          { name: "salt", type: "uint256" },
          { name: "conduitKey", type: "bytes32" },
          { name: "counter", type: "uint256" }
        ],
        name: "orders",
        type: "tuple[]"
      }
    ],
    name: "cancel",
    outputs: [{ name: "cancelled", type: "bool" }],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          {
            components: [
              { name: "offerer", type: "address" },
              { name: "zone", type: "address" },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  }
                ],
                name: "offer",
                type: "tuple[]"
              },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  },
                  {
                    name: "recipient",
                    type: "address"
                  }
                ],
                name: "consideration",
                type: "tuple[]"
              },
              {
                name: "orderType",
                type: "uint8"
              },
              { name: "startTime", type: "uint256" },
              { name: "endTime", type: "uint256" },
              { name: "zoneHash", type: "bytes32" },
              { name: "salt", type: "uint256" },
              { name: "conduitKey", type: "bytes32" },
              {
                name: "totalOriginalConsiderationItems",
                type: "uint256"
              }
            ],
            internalType: "structOrderParameters",
            name: "parameters",
            type: "tuple"
          },
          { name: "numerator", type: "uint120" },
          { name: "denominator", type: "uint120" },
          { name: "signature", type: "bytes" },
          { name: "extraData", type: "bytes" }
        ],
        internalType: "structAdvancedOrder",
        name: "advancedOrder",
        type: "tuple"
      },
      {
        components: [
          { name: "orderIndex", type: "uint256" },
          { internalType: "enumSide", name: "side", type: "uint8" },
          { name: "index", type: "uint256" },
          { name: "identifier", type: "uint256" },
          {
            name: "criteriaProof",
            type: "bytes32[]"
          }
        ],
        internalType: "structCriteriaResolver[]",
        name: "criteriaResolvers",
        type: "tuple[]"
      },
      { name: "fulfillerConduitKey", type: "bytes32" },
      { name: "recipient", type: "address" }
    ],
    name: "fulfillAdvancedOrder",
    outputs: [{ name: "fulfilled", type: "bool" }],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          {
            components: [
              { name: "offerer", type: "address" },
              { name: "zone", type: "address" },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  }
                ],
                name: "offer",
                type: "tuple[]"
              },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  },
                  {
                    name: "recipient",
                    type: "address"
                  }
                ],
                name: "consideration",
                type: "tuple[]"
              },
              {
                name: "orderType",
                type: "uint8"
              },
              { name: "startTime", type: "uint256" },
              { name: "endTime", type: "uint256" },
              { name: "zoneHash", type: "bytes32" },
              { name: "salt", type: "uint256" },
              { name: "conduitKey", type: "bytes32" },
              {
                name: "totalOriginalConsiderationItems",
                type: "uint256"
              }
            ],
            internalType: "structOrderParameters",
            name: "parameters",
            type: "tuple"
          },
          { name: "numerator", type: "uint120" },
          { name: "denominator", type: "uint120" },
          { name: "signature", type: "bytes" },
          { name: "extraData", type: "bytes" }
        ],
        internalType: "structAdvancedOrder[]",
        name: "advancedOrders",
        type: "tuple[]"
      },
      {
        components: [
          { name: "orderIndex", type: "uint256" },
          { internalType: "enumSide", name: "side", type: "uint8" },
          { name: "index", type: "uint256" },
          { name: "identifier", type: "uint256" },
          {
            name: "criteriaProof",
            type: "bytes32[]"
          }
        ],
        internalType: "structCriteriaResolver[]",
        name: "criteriaResolvers",
        type: "tuple[]"
      },
      {
        components: [
          { name: "orderIndex", type: "uint256" },
          { name: "itemIndex", type: "uint256" }
        ],
        internalType: "structFulfillmentComponent[][]",
        name: "offerFulfillments",
        type: "tuple[][]"
      },
      {
        components: [
          { name: "orderIndex", type: "uint256" },
          { name: "itemIndex", type: "uint256" }
        ],
        internalType: "structFulfillmentComponent[][]",
        name: "considerationFulfillments",
        type: "tuple[][]"
      },
      { name: "fulfillerConduitKey", type: "bytes32" },
      { name: "recipient", type: "address" },
      { name: "maximumFulfilled", type: "uint256" }
    ],
    name: "fulfillAvailableAdvancedOrders",
    outputs: [
      { name: "availableOrders", type: "bool[]" },
      {
        components: [
          {
            components: [
              { name: "itemType", type: "uint8" },
              { name: "token", type: "address" },
              { name: "identifier", type: "uint256" },
              { name: "amount", type: "uint256" },
              {
                name: "recipient",
                type: "address"
              }
            ],
            internalType: "structReceivedItem",
            name: "item",
            type: "tuple"
          },
          { name: "offerer", type: "address" },
          { name: "conduitKey", type: "bytes32" }
        ],
        internalType: "structExecution[]",
        name: "executions",
        type: "tuple[]"
      }
    ],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          {
            components: [
              { name: "offerer", type: "address" },
              { name: "zone", type: "address" },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  }
                ],
                name: "offer",
                type: "tuple[]"
              },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  },
                  {
                    name: "recipient",
                    type: "address"
                  }
                ],
                name: "consideration",
                type: "tuple[]"
              },
              {
                name: "orderType",
                type: "uint8"
              },
              { name: "startTime", type: "uint256" },
              { name: "endTime", type: "uint256" },
              { name: "zoneHash", type: "bytes32" },
              { name: "salt", type: "uint256" },
              { name: "conduitKey", type: "bytes32" },
              {
                name: "totalOriginalConsiderationItems",
                type: "uint256"
              }
            ],
            internalType: "structOrderParameters",
            name: "parameters",
            type: "tuple"
          },
          { name: "signature", type: "bytes" }
        ],
        internalType: "structOrder[]",
        name: "orders",
        type: "tuple[]"
      },
      {
        components: [
          { name: "orderIndex", type: "uint256" },
          { name: "itemIndex", type: "uint256" }
        ],
        internalType: "structFulfillmentComponent[][]",
        name: "offerFulfillments",
        type: "tuple[][]"
      },
      {
        components: [
          { name: "orderIndex", type: "uint256" },
          { name: "itemIndex", type: "uint256" }
        ],
        internalType: "structFulfillmentComponent[][]",
        name: "considerationFulfillments",
        type: "tuple[][]"
      },
      { name: "fulfillerConduitKey", type: "bytes32" },
      { name: "maximumFulfilled", type: "uint256" }
    ],
    name: "fulfillAvailableOrders",
    outputs: [
      { name: "availableOrders", type: "bool[]" },
      {
        components: [
          {
            components: [
              { name: "itemType", type: "uint8" },
              { name: "token", type: "address" },
              { name: "identifier", type: "uint256" },
              { name: "amount", type: "uint256" },
              {
                name: "recipient",
                type: "address"
              }
            ],
            internalType: "structReceivedItem",
            name: "item",
            type: "tuple"
          },
          { name: "offerer", type: "address" },
          { name: "conduitKey", type: "bytes32" }
        ],
        internalType: "structExecution[]",
        name: "executions",
        type: "tuple[]"
      }
    ],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          {
            name: "considerationToken",
            type: "address"
          },
          {
            name: "considerationIdentifier",
            type: "uint256"
          },
          {
            name: "considerationAmount",
            type: "uint256"
          },
          { name: "offerer", type: "address" },
          { name: "zone", type: "address" },
          { name: "offerToken", type: "address" },
          { name: "offerIdentifier", type: "uint256" },
          { name: "offerAmount", type: "uint256" },
          {
            internalType: "enumBasicOrderType",
            name: "basicOrderType",
            type: "uint8"
          },
          { name: "startTime", type: "uint256" },
          { name: "endTime", type: "uint256" },
          { name: "zoneHash", type: "bytes32" },
          { name: "salt", type: "uint256" },
          {
            name: "offererConduitKey",
            type: "bytes32"
          },
          {
            name: "fulfillerConduitKey",
            type: "bytes32"
          },
          {
            name: "totalOriginalAdditionalRecipients",
            type: "uint256"
          },
          {
            components: [
              { name: "amount", type: "uint256" },
              {
                name: "recipient",
                type: "address"
              }
            ],
            internalType: "structAdditionalRecipient[]",
            name: "additionalRecipients",
            type: "tuple[]"
          },
          { name: "signature", type: "bytes" }
        ],
        internalType: "structBasicOrderParameters",
        name: "parameters",
        type: "tuple"
      }
    ],
    name: "fulfillBasicOrder",
    outputs: [{ name: "fulfilled", type: "bool" }],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          {
            name: "considerationToken",
            type: "address"
          },
          {
            name: "considerationIdentifier",
            type: "uint256"
          },
          {
            name: "considerationAmount",
            type: "uint256"
          },
          { name: "offerer", type: "address" },
          { name: "zone", type: "address" },
          { name: "offerToken", type: "address" },
          { name: "offerIdentifier", type: "uint256" },
          { name: "offerAmount", type: "uint256" },
          {
            internalType: "enumBasicOrderType",
            name: "basicOrderType",
            type: "uint8"
          },
          { name: "startTime", type: "uint256" },
          { name: "endTime", type: "uint256" },
          { name: "zoneHash", type: "bytes32" },
          { name: "salt", type: "uint256" },
          {
            name: "offererConduitKey",
            type: "bytes32"
          },
          {
            name: "fulfillerConduitKey",
            type: "bytes32"
          },
          {
            name: "totalOriginalAdditionalRecipients",
            type: "uint256"
          },
          {
            components: [
              { name: "amount", type: "uint256" },
              {
                name: "recipient",
                type: "address"
              }
            ],
            internalType: "structAdditionalRecipient[]",
            name: "additionalRecipients",
            type: "tuple[]"
          },
          { name: "signature", type: "bytes" }
        ],
        internalType: "structBasicOrderParameters",
        name: "parameters",
        type: "tuple"
      }
    ],
    name: "fulfillBasicOrder_efficient_6GL6yc",
    outputs: [{ name: "fulfilled", type: "bool" }],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          {
            components: [
              { name: "offerer", type: "address" },
              { name: "zone", type: "address" },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  }
                ],
                name: "offer",
                type: "tuple[]"
              },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  },
                  {
                    name: "recipient",
                    type: "address"
                  }
                ],
                name: "consideration",
                type: "tuple[]"
              },
              {
                name: "orderType",
                type: "uint8"
              },
              { name: "startTime", type: "uint256" },
              { name: "endTime", type: "uint256" },
              { name: "zoneHash", type: "bytes32" },
              { name: "salt", type: "uint256" },
              { name: "conduitKey", type: "bytes32" },
              {
                name: "totalOriginalConsiderationItems",
                type: "uint256"
              }
            ],
            internalType: "structOrderParameters",
            name: "parameters",
            type: "tuple"
          },
          { name: "signature", type: "bytes" }
        ],
        internalType: "structOrder",
        name: "order",
        type: "tuple"
      },
      { name: "fulfillerConduitKey", type: "bytes32" }
    ],
    name: "fulfillOrder",
    outputs: [{ name: "fulfilled", type: "bool" }],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [{ name: "contractOfferer", type: "address" }],
    name: "getContractOffererNonce",
    outputs: [{ name: "nonce", type: "uint256" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [{ name: "offerer", type: "address" }],
    name: "getCounter",
    outputs: [{ name: "counter", type: "uint256" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          { name: "offerer", type: "address" },
          { name: "zone", type: "address" },
          {
            components: [
              { name: "itemType", type: "uint8" },
              { name: "token", type: "address" },
              {
                name: "identifierOrCriteria",
                type: "uint256"
              },
              { name: "startAmount", type: "uint256" },
              { name: "endAmount", type: "uint256" }
            ],
            name: "offer",
            type: "tuple[]"
          },
          {
            components: [
              { name: "itemType", type: "uint8" },
              { name: "token", type: "address" },
              {
                name: "identifierOrCriteria",
                type: "uint256"
              },
              { name: "startAmount", type: "uint256" },
              { name: "endAmount", type: "uint256" },
              {
                name: "recipient",
                type: "address"
              }
            ],
            name: "consideration",
            type: "tuple[]"
          },
          { name: "orderType", type: "uint8" },
          { name: "startTime", type: "uint256" },
          { name: "endTime", type: "uint256" },
          { name: "zoneHash", type: "bytes32" },
          { name: "salt", type: "uint256" },
          { name: "conduitKey", type: "bytes32" },
          { name: "counter", type: "uint256" }
        ],
        internalType: "structOrderComponents",
        name: "order",
        type: "tuple"
      }
    ],
    name: "getOrderHash",
    outputs: [{ name: "orderHash", type: "bytes32" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [{ name: "orderHash", type: "bytes32" }],
    name: "getOrderStatus",
    outputs: [
      { name: "isValidated", type: "bool" },
      { name: "isCancelled", type: "bool" },
      { name: "totalFilled", type: "uint256" },
      { name: "totalSize", type: "uint256" }
    ],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "incrementCounter",
    outputs: [{ name: "newCounter", type: "uint256" }],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "information",
    outputs: [
      { name: "version", type: "string" },
      { name: "domainSeparator", type: "bytes32" },
      { name: "conduitController", type: "address" }
    ],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          {
            components: [
              { name: "offerer", type: "address" },
              { name: "zone", type: "address" },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  }
                ],
                name: "offer",
                type: "tuple[]"
              },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  },
                  {
                    name: "recipient",
                    type: "address"
                  }
                ],
                name: "consideration",
                type: "tuple[]"
              },
              {
                name: "orderType",
                type: "uint8"
              },
              { name: "startTime", type: "uint256" },
              { name: "endTime", type: "uint256" },
              { name: "zoneHash", type: "bytes32" },
              { name: "salt", type: "uint256" },
              { name: "conduitKey", type: "bytes32" },
              {
                name: "totalOriginalConsiderationItems",
                type: "uint256"
              }
            ],
            internalType: "structOrderParameters",
            name: "parameters",
            type: "tuple"
          },
          { name: "numerator", type: "uint120" },
          { name: "denominator", type: "uint120" },
          { name: "signature", type: "bytes" },
          { name: "extraData", type: "bytes" }
        ],
        internalType: "structAdvancedOrder[]",
        name: "orders",
        type: "tuple[]"
      },
      {
        components: [
          { name: "orderIndex", type: "uint256" },
          { internalType: "enumSide", name: "side", type: "uint8" },
          { name: "index", type: "uint256" },
          { name: "identifier", type: "uint256" },
          {
            name: "criteriaProof",
            type: "bytes32[]"
          }
        ],
        internalType: "structCriteriaResolver[]",
        name: "criteriaResolvers",
        type: "tuple[]"
      },
      {
        components: [
          {
            components: [
              { name: "orderIndex", type: "uint256" },
              { name: "itemIndex", type: "uint256" }
            ],
            internalType: "structFulfillmentComponent[]",
            name: "offerComponents",
            type: "tuple[]"
          },
          {
            components: [
              { name: "orderIndex", type: "uint256" },
              { name: "itemIndex", type: "uint256" }
            ],
            internalType: "structFulfillmentComponent[]",
            name: "considerationComponents",
            type: "tuple[]"
          }
        ],
        internalType: "structFulfillment[]",
        name: "fulfillments",
        type: "tuple[]"
      },
      { name: "recipient", type: "address" }
    ],
    name: "matchAdvancedOrders",
    outputs: [
      {
        components: [
          {
            components: [
              { name: "itemType", type: "uint8" },
              { name: "token", type: "address" },
              { name: "identifier", type: "uint256" },
              { name: "amount", type: "uint256" },
              {
                name: "recipient",
                type: "address"
              }
            ],
            internalType: "structReceivedItem",
            name: "item",
            type: "tuple"
          },
          { name: "offerer", type: "address" },
          { name: "conduitKey", type: "bytes32" }
        ],
        internalType: "structExecution[]",
        name: "executions",
        type: "tuple[]"
      }
    ],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          {
            components: [
              { name: "offerer", type: "address" },
              { name: "zone", type: "address" },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  }
                ],
                name: "offer",
                type: "tuple[]"
              },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  },
                  {
                    name: "recipient",
                    type: "address"
                  }
                ],
                name: "consideration",
                type: "tuple[]"
              },
              {
                name: "orderType",
                type: "uint8"
              },
              { name: "startTime", type: "uint256" },
              { name: "endTime", type: "uint256" },
              { name: "zoneHash", type: "bytes32" },
              { name: "salt", type: "uint256" },
              { name: "conduitKey", type: "bytes32" },
              {
                name: "totalOriginalConsiderationItems",
                type: "uint256"
              }
            ],
            internalType: "structOrderParameters",
            name: "parameters",
            type: "tuple"
          },
          { name: "signature", type: "bytes" }
        ],
        internalType: "structOrder[]",
        name: "orders",
        type: "tuple[]"
      },
      {
        components: [
          {
            components: [
              { name: "orderIndex", type: "uint256" },
              { name: "itemIndex", type: "uint256" }
            ],
            internalType: "structFulfillmentComponent[]",
            name: "offerComponents",
            type: "tuple[]"
          },
          {
            components: [
              { name: "orderIndex", type: "uint256" },
              { name: "itemIndex", type: "uint256" }
            ],
            internalType: "structFulfillmentComponent[]",
            name: "considerationComponents",
            type: "tuple[]"
          }
        ],
        internalType: "structFulfillment[]",
        name: "fulfillments",
        type: "tuple[]"
      }
    ],
    name: "matchOrders",
    outputs: [
      {
        components: [
          {
            components: [
              { name: "itemType", type: "uint8" },
              { name: "token", type: "address" },
              { name: "identifier", type: "uint256" },
              { name: "amount", type: "uint256" },
              {
                name: "recipient",
                type: "address"
              }
            ],
            internalType: "structReceivedItem",
            name: "item",
            type: "tuple"
          },
          { name: "offerer", type: "address" },
          { name: "conduitKey", type: "bytes32" }
        ],
        internalType: "structExecution[]",
        name: "executions",
        type: "tuple[]"
      }
    ],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [],
    name: "name",
    outputs: [{ name: "contractName", type: "string" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          {
            components: [
              { name: "offerer", type: "address" },
              { name: "zone", type: "address" },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  }
                ],
                name: "offer",
                type: "tuple[]"
              },
              {
                components: [
                  {
                    internalType: "enumItemType",
                    name: "itemType",
                    type: "uint8"
                  },
                  { name: "token", type: "address" },
                  {
                    name: "identifierOrCriteria",
                    type: "uint256"
                  },
                  {
                    name: "startAmount",
                    type: "uint256"
                  },
                  {
                    name: "endAmount",
                    type: "uint256"
                  },
                  {
                    name: "recipient",
                    type: "address"
                  }
                ],
                name: "consideration",
                type: "tuple[]"
              },
              {
                name: "orderType",
                type: "uint8"
              },
              { name: "startTime", type: "uint256" },
              { name: "endTime", type: "uint256" },
              { name: "zoneHash", type: "bytes32" },
              { name: "salt", type: "uint256" },
              { name: "conduitKey", type: "bytes32" },
              {
                name: "totalOriginalConsiderationItems",
                type: "uint256"
              }
            ],
            internalType: "structOrderParameters",
            name: "parameters",
            type: "tuple"
          },
          { name: "signature", type: "bytes" }
        ],
        internalType: "structOrder[]",
        name: "orders",
        type: "tuple[]"
      }
    ],
    name: "validate",
    outputs: [{ name: "validated", type: "bool" }],
    stateMutability: "nonpayable",
    type: "function"
  },
  { inputs: [], name: "BadContractSignature", type: "error" },
  { inputs: [], name: "BadFraction", type: "error" },
  {
    inputs: [
      { name: "token", type: "address" },
      { name: "from", type: "address" },
      { name: "to", type: "address" },
      { name: "amount", type: "uint256" }
    ],
    name: "BadReturnValueFromERC20OnTransfer",
    type: "error"
  },
  {
    inputs: [{ name: "v", type: "uint8" }],
    name: "BadSignatureV",
    type: "error"
  },
  { inputs: [], name: "CannotCancelOrder", type: "error" },
  {
    inputs: [],
    name: "ConsiderationCriteriaResolverOutOfRange",
    type: "error"
  },
  {
    inputs: [],
    name: "ConsiderationLengthNotEqualToTotalOriginal",
    type: "error"
  },
  {
    inputs: [
      { name: "orderIndex", type: "uint256" },
      { name: "considerationIndex", type: "uint256" },
      { name: "shortfallAmount", type: "uint256" }
    ],
    name: "ConsiderationNotMet",
    type: "error"
  },
  { inputs: [], name: "CriteriaNotEnabledForItem", type: "error" },
  {
    inputs: [
      { name: "token", type: "address" },
      { name: "from", type: "address" },
      { name: "to", type: "address" },
      { name: "identifiers", type: "uint256[]" },
      { name: "amounts", type: "uint256[]" }
    ],
    name: "ERC1155BatchTransferGenericFailure",
    type: "error"
  },
  { inputs: [], name: "InexactFraction", type: "error" },
  { inputs: [], name: "InsufficientNativeTokensSupplied", type: "error" },
  { inputs: [], name: "Invalid1155BatchTransferEncoding", type: "error" },
  { inputs: [], name: "InvalidBasicOrderParameterEncoding", type: "error" },
  {
    inputs: [{ name: "conduit", type: "address" }],
    name: "InvalidCallToConduit",
    type: "error"
  },
  {
    inputs: [
      { name: "conduitKey", type: "bytes32" },
      { name: "conduit", type: "address" }
    ],
    name: "InvalidConduit",
    type: "error"
  },
  {
    inputs: [{ name: "orderHash", type: "bytes32" }],
    name: "InvalidContractOrder",
    type: "error"
  },
  {
    inputs: [{ name: "amount", type: "uint256" }],
    name: "InvalidERC721TransferAmount",
    type: "error"
  },
  { inputs: [], name: "InvalidFulfillmentComponentData", type: "error" },
  {
    inputs: [{ name: "value", type: "uint256" }],
    name: "InvalidMsgValue",
    type: "error"
  },
  { inputs: [], name: "InvalidNativeOfferItem", type: "error" },
  { inputs: [], name: "InvalidProof", type: "error" },
  {
    inputs: [{ name: "orderHash", type: "bytes32" }],
    name: "InvalidRestrictedOrder",
    type: "error"
  },
  { inputs: [], name: "InvalidSignature", type: "error" },
  { inputs: [], name: "InvalidSigner", type: "error" },
  {
    inputs: [
      { name: "startTime", type: "uint256" },
      { name: "endTime", type: "uint256" }
    ],
    name: "InvalidTime",
    type: "error"
  },
  {
    inputs: [{ name: "fulfillmentIndex", type: "uint256" }],
    name: "MismatchedFulfillmentOfferAndConsiderationComponents",
    type: "error"
  },
  {
    inputs: [{ internalType: "enumSide", name: "side", type: "uint8" }],
    name: "MissingFulfillmentComponentOnAggregation",
    type: "error"
  },
  { inputs: [], name: "MissingItemAmount", type: "error" },
  { inputs: [], name: "MissingOriginalConsiderationItems", type: "error" },
  {
    inputs: [
      { name: "account", type: "address" },
      { name: "amount", type: "uint256" }
    ],
    name: "NativeTokenTransferGenericFailure",
    type: "error"
  },
  {
    inputs: [{ name: "account", type: "address" }],
    name: "NoContract",
    type: "error"
  },
  { inputs: [], name: "NoReentrantCalls", type: "error" },
  { inputs: [], name: "NoSpecifiedOrdersAvailable", type: "error" },
  {
    inputs: [],
    name: "OfferAndConsiderationRequiredOnFulfillment",
    type: "error"
  },
  { inputs: [], name: "OfferCriteriaResolverOutOfRange", type: "error" },
  {
    inputs: [{ name: "orderHash", type: "bytes32" }],
    name: "OrderAlreadyFilled",
    type: "error"
  },
  {
    inputs: [{ internalType: "enumSide", name: "side", type: "uint8" }],
    name: "OrderCriteriaResolverOutOfRange",
    type: "error"
  },
  {
    inputs: [{ name: "orderHash", type: "bytes32" }],
    name: "OrderIsCancelled",
    type: "error"
  },
  {
    inputs: [{ name: "orderHash", type: "bytes32" }],
    name: "OrderPartiallyFilled",
    type: "error"
  },
  { inputs: [], name: "PartialFillsNotEnabledForOrder", type: "error" },
  {
    inputs: [
      { name: "token", type: "address" },
      { name: "from", type: "address" },
      { name: "to", type: "address" },
      { name: "identifier", type: "uint256" },
      { name: "amount", type: "uint256" }
    ],
    name: "TokenTransferGenericFailure",
    type: "error"
  },
  {
    inputs: [
      { name: "orderIndex", type: "uint256" },
      { name: "considerationIndex", type: "uint256" }
    ],
    name: "UnresolvedConsiderationCriteria",
    type: "error"
  },
  {
    inputs: [
      { name: "orderIndex", type: "uint256" },
      { name: "offerIndex", type: "uint256" }
    ],
    name: "UnresolvedOfferCriteria",
    type: "error"
  },
  { inputs: [], name: "UnusedItemParameters", type: "error" },
  {
    inputs: [
      {
        indexed: false,
        name: "newCounter",
        type: "uint256"
      },
      {
        indexed: true,
        name: "offerer",
        type: "address"
      }
    ],
    name: "CounterIncremented",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: false,
        name: "orderHash",
        type: "bytes32"
      },
      {
        indexed: true,
        name: "offerer",
        type: "address"
      },
      { indexed: true, name: "zone", type: "address" }
    ],
    name: "OrderCancelled",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: false,
        name: "orderHash",
        type: "bytes32"
      },
      {
        indexed: true,
        name: "offerer",
        type: "address"
      },
      { indexed: true, name: "zone", type: "address" },
      {
        indexed: false,
        name: "recipient",
        type: "address"
      },
      {
        components: [
          { name: "itemType", type: "uint8" },
          { name: "token", type: "address" },
          { name: "identifier", type: "uint256" },
          { name: "amount", type: "uint256" }
        ],
        indexed: false,
        internalType: "structSpentItem[]",
        name: "offer",
        type: "tuple[]"
      },
      {
        components: [
          { name: "itemType", type: "uint8" },
          { name: "token", type: "address" },
          { name: "identifier", type: "uint256" },
          { name: "amount", type: "uint256" },
          {
            name: "recipient",
            type: "address"
          }
        ],
        indexed: false,
        internalType: "structReceivedItem[]",
        name: "consideration",
        type: "tuple[]"
      }
    ],
    name: "OrderFulfilled",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: false,
        name: "orderHash",
        type: "bytes32"
      },
      {
        components: [
          { name: "offerer", type: "address" },
          { name: "zone", type: "address" },
          {
            components: [
              { name: "itemType", type: "uint8" },
              { name: "token", type: "address" },
              {
                name: "identifierOrCriteria",
                type: "uint256"
              },
              { name: "startAmount", type: "uint256" },
              { name: "endAmount", type: "uint256" }
            ],
            name: "offer",
            type: "tuple[]"
          },
          {
            components: [
              { name: "itemType", type: "uint8" },
              { name: "token", type: "address" },
              {
                name: "identifierOrCriteria",
                type: "uint256"
              },
              { name: "startAmount", type: "uint256" },
              { name: "endAmount", type: "uint256" },
              {
                name: "recipient",
                type: "address"
              }
            ],
            name: "consideration",
            type: "tuple[]"
          },
          { name: "orderType", type: "uint8" },
          { name: "startTime", type: "uint256" },
          { name: "endTime", type: "uint256" },
          { name: "zoneHash", type: "bytes32" },
          { name: "salt", type: "uint256" },
          { name: "conduitKey", type: "bytes32" },
          {
            name: "totalOriginalConsiderationItems",
            type: "uint256"
          }
        ],
        indexed: false,
        internalType: "structOrderParameters",
        name: "orderParameters",
        type: "tuple"
      }
    ],
    name: "OrderValidated",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: false,
        name: "orderHashes",
        type: "bytes32[]"
      }
    ],
    name: "OrdersMatched",
    type: "event"
  }
];
var wagmiMintExampleAbi = [
  { inputs: [], stateMutability: "nonpayable", type: "constructor" },
  {
    inputs: [
      {
        name: "owner",
        type: "address",
        indexed: true
      },
      {
        name: "approved",
        type: "address",
        indexed: true
      },
      {
        name: "tokenId",
        type: "uint256",
        indexed: true
      }
    ],
    name: "Approval",
    type: "event"
  },
  {
    inputs: [
      {
        name: "owner",
        type: "address",
        indexed: true
      },
      {
        name: "operator",
        type: "address",
        indexed: true
      },
      {
        name: "approved",
        type: "bool",
        indexed: false
      }
    ],
    name: "ApprovalForAll",
    type: "event"
  },
  {
    inputs: [
      {
        name: "from",
        type: "address",
        indexed: true
      },
      { name: "to", type: "address", indexed: true },
      {
        name: "tokenId",
        type: "uint256",
        indexed: true
      }
    ],
    name: "Transfer",
    type: "event"
  },
  {
    inputs: [
      { name: "to", type: "address" },
      { name: "tokenId", type: "uint256" }
    ],
    name: "approve",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [{ name: "owner", type: "address" }],
    name: "balanceOf",
    outputs: [{ type: "uint256" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [{ name: "tokenId", type: "uint256" }],
    name: "getApproved",
    outputs: [{ type: "address" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      { name: "owner", type: "address" },
      { name: "operator", type: "address" }
    ],
    name: "isApprovedForAll",
    outputs: [{ type: "bool" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "mint",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "name",
    outputs: [{ type: "string" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [{ name: "tokenId", type: "uint256" }],
    name: "ownerOf",
    outputs: [{ type: "address" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      { name: "from", type: "address" },
      { name: "to", type: "address" },
      { name: "tokenId", type: "uint256" }
    ],
    name: "safeTransferFrom",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [
      { name: "from", type: "address" },
      { name: "to", type: "address" },
      { name: "tokenId", type: "uint256" },
      { name: "_data", type: "bytes" }
    ],
    name: "safeTransferFrom",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [
      { name: "operator", type: "address" },
      { name: "approved", type: "bool" }
    ],
    name: "setApprovalForAll",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [{ name: "interfaceId", type: "bytes4" }],
    name: "supportsInterface",
    outputs: [{ type: "bool" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "symbol",
    outputs: [{ type: "string" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [{ name: "tokenId", type: "uint256" }],
    name: "tokenURI",
    outputs: [{ type: "string" }],
    stateMutability: "pure",
    type: "function"
  },
  {
    inputs: [],
    name: "totalSupply",
    outputs: [{ type: "uint256" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      { name: "from", type: "address" },
      { name: "to", type: "address" },
      { name: "tokenId", type: "uint256" }
    ],
    name: "transferFrom",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  }
];
var wethAbi = [
  {
    constant: true,
    inputs: [],
    name: "name",
    outputs: [{ type: "string" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "guy", type: "address" },
      { name: "wad", type: "uint256" }
    ],
    name: "approve",
    outputs: [{ type: "bool" }],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: true,
    inputs: [],
    name: "totalSupply",
    outputs: [{ type: "uint256" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "src", type: "address" },
      { name: "dst", type: "address" },
      { name: "wad", type: "uint256" }
    ],
    name: "transferFrom",
    outputs: [{ type: "bool" }],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: false,
    inputs: [{ name: "wad", type: "uint256" }],
    name: "withdraw",
    outputs: [],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: true,
    inputs: [],
    name: "decimals",
    outputs: [{ type: "uint8" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  {
    constant: true,
    inputs: [{ type: "address" }],
    name: "balanceOf",
    outputs: [{ type: "uint256" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  {
    constant: true,
    inputs: [],
    name: "symbol",
    outputs: [{ type: "string" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  {
    constant: false,
    inputs: [
      { name: "dst", type: "address" },
      { name: "wad", type: "uint256" }
    ],
    name: "transfer",
    outputs: [{ type: "bool" }],
    payable: false,
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    constant: false,
    inputs: [],
    name: "deposit",
    outputs: [],
    payable: true,
    stateMutability: "payable",
    type: "function"
  },
  {
    constant: true,
    inputs: [{ type: "address" }, { type: "address" }],
    name: "allowance",
    outputs: [{ type: "uint256" }],
    payable: false,
    stateMutability: "view",
    type: "function"
  },
  { payable: true, stateMutability: "payable", type: "fallback" },
  {
    inputs: [
      { indexed: true, name: "src", type: "address" },
      { indexed: true, name: "guy", type: "address" },
      { indexed: false, name: "wad", type: "uint256" }
    ],
    name: "Approval",
    type: "event"
  },
  {
    inputs: [
      { indexed: true, name: "src", type: "address" },
      { indexed: true, name: "dst", type: "address" },
      { indexed: false, name: "wad", type: "uint256" }
    ],
    name: "Transfer",
    type: "event"
  },
  {
    inputs: [
      { indexed: true, name: "dst", type: "address" },
      { indexed: false, name: "wad", type: "uint256" }
    ],
    name: "Deposit",
    type: "event"
  },
  {
    inputs: [
      { indexed: true, name: "src", type: "address" },
      { indexed: false, name: "wad", type: "uint256" }
    ],
    name: "Withdrawal",
    type: "event"
  }
];
var writingEditionsFactoryAbi = [
  {
    inputs: [
      { name: "_owner", type: "address" },
      {
        name: "_treasuryConfiguration",
        type: "address"
      },
      { name: "_maxLimit", type: "uint256" },
      { name: "_guardOn", type: "bool" }
    ],
    stateMutability: "nonpayable",
    type: "constructor"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "clone",
        type: "address"
      },
      {
        indexed: false,
        name: "oldBaseDescriptionURI",
        type: "string"
      },
      {
        indexed: false,
        name: "newBaseDescriptionURI",
        type: "string"
      }
    ],
    name: "BaseDescriptionURISet",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "factory",
        type: "address"
      },
      {
        indexed: true,
        name: "owner",
        type: "address"
      },
      {
        indexed: true,
        name: "clone",
        type: "address"
      }
    ],
    name: "CloneDeployed",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "owner",
        type: "address"
      },
      {
        indexed: true,
        name: "clone",
        type: "address"
      },
      {
        indexed: true,
        name: "implementation",
        type: "address"
      }
    ],
    name: "EditionsDeployed",
    type: "event"
  },
  {
    inputs: [{ indexed: false, name: "guard", type: "bool" }],
    name: "FactoryGuardSet",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "factory",
        type: "address"
      },
      {
        indexed: true,
        name: "oldImplementation",
        type: "address"
      },
      {
        indexed: true,
        name: "newImplementation",
        type: "address"
      }
    ],
    name: "FactoryImplementationSet",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "factory",
        type: "address"
      },
      {
        indexed: false,
        name: "oldLimit",
        type: "uint256"
      },
      {
        indexed: false,
        name: "newLimit",
        type: "uint256"
      }
    ],
    name: "FactoryLimitSet",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "clone",
        type: "address"
      },
      {
        indexed: true,
        name: "oldFundingRecipient",
        type: "address"
      },
      {
        indexed: true,
        name: "newFundingRecipient",
        type: "address"
      }
    ],
    name: "FundingRecipientSet",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "oldImplementation",
        type: "address"
      },
      {
        indexed: true,
        name: "newImplementation",
        type: "address"
      }
    ],
    name: "NewImplementation",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "previousOwner",
        type: "address"
      },
      {
        indexed: true,
        name: "newOwner",
        type: "address"
      }
    ],
    name: "OwnershipTransferred",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "clone",
        type: "address"
      },
      {
        indexed: false,
        name: "oldLimit",
        type: "uint256"
      },
      {
        indexed: false,
        name: "newLimit",
        type: "uint256"
      }
    ],
    name: "PriceSet",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "clone",
        type: "address"
      },
      {
        indexed: true,
        name: "renderer",
        type: "address"
      }
    ],
    name: "RendererSet",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "clone",
        type: "address"
      },
      {
        indexed: true,
        name: "oldRoyaltyRecipient",
        type: "address"
      },
      {
        indexed: false,
        name: "oldRoyaltyBPS",
        type: "uint256"
      },
      {
        indexed: true,
        name: "newRoyaltyRecipient",
        type: "address"
      },
      {
        indexed: false,
        name: "newRoyaltyBPS",
        type: "uint256"
      }
    ],
    name: "RoyaltyChange",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "clone",
        type: "address"
      },
      { indexed: true, name: "from", type: "address" },
      { indexed: true, name: "to", type: "address" },
      {
        indexed: false,
        name: "tokenId",
        type: "uint256"
      }
    ],
    name: "Transfer",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "factory",
        type: "address"
      },
      {
        indexed: true,
        name: "clone",
        type: "address"
      },
      {
        indexed: false,
        name: "oldTributary",
        type: "address"
      },
      {
        indexed: true,
        name: "newTributary",
        type: "address"
      }
    ],
    name: "TributarySet",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "clone",
        type: "address"
      },
      {
        indexed: false,
        name: "oldLimit",
        type: "uint256"
      },
      {
        indexed: false,
        name: "newLimit",
        type: "uint256"
      }
    ],
    name: "WritingEditionLimitSet",
    type: "event"
  },
  {
    inputs: [
      {
        indexed: true,
        name: "clone",
        type: "address"
      },
      {
        indexed: false,
        name: "tokenId",
        type: "uint256"
      },
      {
        indexed: true,
        name: "recipient",
        type: "address"
      },
      {
        indexed: false,
        name: "price",
        type: "uint256"
      },
      {
        indexed: false,
        name: "message",
        type: "string"
      }
    ],
    name: "WritingEditionPurchased",
    type: "event"
  },
  {
    inputs: [],
    name: "CREATE_TYPEHASH",
    outputs: [{ type: "bytes32" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "DOMAIN_SEPARATOR",
    outputs: [{ type: "bytes32" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "VERSION",
    outputs: [{ type: "uint8" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "acceptOwnership",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "baseDescriptionURI",
    outputs: [{ type: "string" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "cancelOwnershipTransfer",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [
      {
        components: [
          { name: "name", type: "string" },
          { name: "symbol", type: "string" },
          { name: "description", type: "string" },
          { name: "imageURI", type: "string" },
          { name: "contentURI", type: "string" },
          { name: "price", type: "uint8" },
          { name: "limit", type: "uint256" },
          {
            name: "fundingRecipient",
            type: "address"
          },
          { name: "renderer", type: "address" },
          { name: "nonce", type: "uint256" },
          { name: "fee", type: "uint16" }
        ],
        internalType: "struct IWritingEditions.WritingEdition",
        name: "edition",
        type: "tuple"
      }
    ],
    name: "create",
    outputs: [{ name: "clone", type: "address" }],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [
      { name: "owner", type: "address" },
      {
        components: [
          { name: "name", type: "string" },
          { name: "symbol", type: "string" },
          { name: "description", type: "string" },
          { name: "imageURI", type: "string" },
          { name: "contentURI", type: "string" },
          { name: "price", type: "uint256" },
          { name: "limit", type: "uint256" },
          {
            name: "fundingRecipient",
            type: "address"
          },
          { name: "renderer", type: "address" },
          { name: "nonce", type: "uint256" },
          { name: "fee", type: "uint16" }
        ],
        internalType: "struct IWritingEditions.WritingEdition",
        name: "edition",
        type: "tuple"
      },
      { name: "v", type: "uint8" },
      { name: "r", type: "bytes32" },
      { name: "s", type: "bytes32" },
      { name: "tokenRecipient", type: "address" },
      { name: "message", type: "string" }
    ],
    name: "createWithSignature",
    outputs: [{ name: "clone", type: "address" }],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [
      { name: "owner", type: "address" },
      {
        components: [
          { name: "name", type: "string" },
          { name: "symbol", type: "string" },
          { name: "description", type: "string" },
          { name: "imageURI", type: "string" },
          { name: "contentURI", type: "string" },
          { name: "price", type: "uint8" },
          { name: "limit", type: "uint256" },
          {
            name: "fundingRecipient",
            type: "address"
          },
          { name: "renderer", type: "address" },
          { name: "nonce", type: "uint256" },
          { name: "fee", type: "uint16" }
        ],
        internalType: "struct IWritingEditions.WritingEdition",
        name: "edition",
        type: "tuple"
      }
    ],
    name: "getSalt",
    outputs: [{ type: "bytes32" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "guardOn",
    outputs: [{ type: "bool" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "implementation",
    outputs: [{ type: "address" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "isNextOwner",
    outputs: [{ type: "bool" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "isOwner",
    outputs: [{ type: "bool" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      { name: "owner", type: "address" },
      { name: "salt", type: "bytes32" },
      { name: "v", type: "uint8" },
      { name: "r", type: "bytes32" },
      { name: "s", type: "bytes32" }
    ],
    name: "isValid",
    outputs: [{ type: "bool" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "maxLimit",
    outputs: [{ type: "uint256" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "o11y",
    outputs: [{ type: "address" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [],
    name: "owner",
    outputs: [{ type: "address" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      { name: "_implementation", type: "address" },
      { name: "salt", type: "bytes32" }
    ],
    name: "predictDeterministicAddress",
    outputs: [{ type: "address" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [
      { name: "clone", type: "address" },
      { name: "tokenRecipient", type: "address" },
      { name: "message", type: "string" }
    ],
    name: "purchaseThroughFactory",
    outputs: [{ name: "tokenId", type: "uint256" }],
    stateMutability: "payable",
    type: "function"
  },
  {
    inputs: [],
    name: "renounceOwnership",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [{ type: "bytes32" }],
    name: "salts",
    outputs: [{ type: "bool" }],
    stateMutability: "view",
    type: "function"
  },
  {
    inputs: [{ name: "_guardOn", type: "bool" }],
    name: "setGuard",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [{ name: "_implementation", type: "address" }],
    name: "setImplementation",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [{ name: "_maxLimit", type: "uint256" }],
    name: "setLimit",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [
      { name: "clone", type: "address" },
      { name: "_tributary", type: "address" }
    ],
    name: "setTributary",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [{ name: "nextOwner_", type: "address" }],
    name: "transferOwnership",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function"
  },
  {
    inputs: [],
    name: "treasuryConfiguration",
    outputs: [{ type: "address" }],
    stateMutability: "view",
    type: "function"
  }
];

// src/test/human-readable.ts
var customSolidityErrorsHumanReadableAbi = [
  "constructor()",
  "error ApprovalCallerNotOwnerNorApproved()",
  "error ApprovalQueryForNonexistentToken()"
];
var ensHumanReadableAbi = [
  "function resolver(bytes32 node) view returns (address)",
  "function owner(bytes32 node) view returns (address)",
  "function setSubnodeOwner(bytes32 node, bytes32 label, address owner)",
  "function setTTL(bytes32 node, uint64 ttl)",
  "function ttl(bytes32 node) view returns (uint64)",
  "function setResolver(bytes32 node, address resolver)",
  "function setOwner(bytes32 node, address owner)",
  "event Transfer(bytes32 indexed node, address owner)",
  "event NewOwner(bytes32 indexed node, bytes32 indexed label, address owner)",
  "event NewResolver(bytes32 indexed node, address resolver)",
  "event NewTTL(bytes32 indexed node, uint64 ttl)"
];
var ensRegistryWithFallbackHumanReadableAbi = [
  "constructor(address _old)",
  "event ApprovalForAll(address indexed owner, address indexed operator, bool approved)",
  "event NewOwner(bytes32 indexed node, bytes32 indexed label, address owner)",
  "event NewResolver(bytes32 indexed node, address resolver)",
  "event NewTTL(bytes32 indexed node, uint64 ttl)",
  "event Transfer(bytes32 indexed node, address owner)",
  "function isApprovedForAll(address owner, address operator) view returns (bool)",
  "function old() view returns (address)",
  "function owner(bytes32 node) view returns (address)",
  "function recordExists(bytes32 node) view returns (bool)",
  "function resolver(bytes32 node) view returns (address)",
  "function setApprovalForAll(address operator, bool approved)",
  "function setOwner(bytes32 node, address owner)",
  "function setRecord(bytes32 node, address owner, address resolver, uint64 ttl)",
  "function setResolver(bytes32 node, address resolver)",
  "function setSubnodeOwner(bytes32 node, bytes32 label, address owner)",
  "function setSubnodeRecord(bytes32 node, bytes32 label, address owner, address resolver, uint64 ttl)",
  "function setTTL(bytes32 node, uint64 ttl)",
  "function ttl(bytes32 node) view returns (uint64)"
];
var erc20HumanReadableAbi = [
  "event Approval(address indexed owner, address indexed spender, uint256 value)",
  "event Transfer(address indexed from, address indexed to, uint256 value)",
  "function allowance(address owner, address spender) view returns (uint256)",
  "function approve(address spender, uint256 amount) returns (bool)",
  "function balanceOf(address account) view returns (uint256)",
  "function decimals() view returns (uint8)",
  "function name() view returns (string)",
  "function symbol() view returns (string)",
  "function totalSupply() view returns (uint256)",
  "function transfer(address recipient, uint256 amount) returns (bool)",
  "function transferFrom(address sender, address recipient, uint256 amount) returns (bool)"
];
var nestedTupleArrayHumanReadableAbi = [
  "function f((uint8 a, uint8[] b, (uint8 x, uint8 y)[] c) s, (uint x, uint y) t, uint256 a) returns ((uint256 x, uint256 y)[] t)",
  "function v((uint8 a, uint8[] b) s, (uint x, uint y) t, uint256 a)"
];
var nounsAuctionHouseHumanReadableAbi = [
  "event AuctionBid(uint256 indexed nounId, address sender, uint256 value, bool extended)",
  "event AuctionCreated(uint256 indexed nounId, uint256 startTime, uint256 endTime)",
  "event AuctionExtended(uint256 indexed nounId, uint256 endTime)",
  "event AuctionMinBidIncrementPercentageUpdated(uint256 minBidIncrementPercentage)",
  "event AuctionReservePriceUpdated(uint256 reservePrice)",
  "event AuctionSettled(uint256 indexed nounId, address winner, uint256 amount)",
  "event AuctionTimeBufferUpdated(uint256 timeBuffer)",
  "event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)",
  "event Paused(address account)",
  "event Unpaused(address account)",
  "function auction(uint256 nounId) view returns (uint256 nounId, uint256 amount, uint256 startTime, uint256 endTime, address bidder, bool settled)",
  "function createBid(uint256 nounId) payable",
  "function duration() view returns (uint256)",
  "function initialize(address _nouns, address _weth, uint256 _timeBuffer, uint256 _reservePrice, uint8 _minBidIncrementPercentage, uint256 _duration)",
  "function minBidIncrementPercentage() view returns (uint8)",
  "function nouns() view returns (address)",
  "function owner() view returns (address)",
  "function pause()",
  "function paused() view returns (bool)",
  "function renounceOwnership()",
  "function reservePrice() view returns (uint256)",
  "function setMinBidIncrementPercentage(uint8 _minBidIncrementPercentage)",
  "function setReservePrice(uint256 _reservePrice)",
  "function setTimeBuffer(uint256 _timeBuffer)",
  "function settleAuction()",
  "function settleCurrentAndCreateNewAuction()",
  "function timeBuffer() view returns (uint256)",
  "function newOwner() view returns (address)",
  "function unpause()",
  "function weth() view returns (address)"
];
var seaportHumanReadableAbi = [
  "constructor(address conduitController)",
  // structs
  "struct AdditionalRecipient { uint256 amount; address recipient; }",
  "struct AdvancedOrder { OrderParameters parameters; uint120 numerator; uint120 denominator; bytes signature; bytes extraData; }",
  "struct BasicOrderParameters { address considerationToken; uint256 considerationIdentifier; uint256 considerationAmount; address offerer; address zone; address offerToken; uint256 offerIdentifier; uint256 offerAmount; uint8 basicOrderType; uint256 startTime; uint256 endTime; bytes32 zoneHash; uint256 salt; bytes32 offererConduitKey; bytes32 fulfillerConduitKey; uint256 totalOriginalAdditionalRecipients; AdditionalRecipient[] additionalRecipients; bytes signature; }",
  "struct ConsiderationItem { uint8 itemType; address token; uint256 identifierOrCriteria; uint256 startAmount; uint256 endAmount; address recipient; }",
  "struct CriteriaResolver { uint256 orderIndex; uint8 side; uint256 index; uint256 identifier; bytes32[] criteriaProof; }",
  "struct Execution { ReceivedItem item; address offerer; bytes32 conduitKey; }",
  "struct Fulfillment { FulfillmentComponent[] offerComponents; FulfillmentComponent[] considerationComponents; }",
  "struct FulfillmentComponent { uint256 orderIndex; uint256 itemIndex; }",
  "struct OfferItem { uint8 itemType; address token; uint256 identifierOrCriteria; uint256 startAmount; uint256 endAmount; }",
  "struct Order { OrderParameters parameters; bytes signature; }",
  "struct OrderComponents { address offerer; address zone; OfferItem[] offer; ConsiderationItem[] consideration; uint8 orderType; uint256 startTime; uint256 endTime; bytes32 zoneHash; uint256 salt; bytes32 conduitKey; uint256 counter; }",
  "struct OrderParameters { address offerer; address zone; OfferItem[] offer; ConsiderationItem[] consideration; uint8 orderType; uint256 startTime; uint256 endTime; bytes32 zoneHash; uint256 salt; bytes32 conduitKey; uint256 totalOriginalConsiderationItems; }",
  "struct OrderStatus { bool isValidated; bool isCancelled; uint120 numerator; uint120 denominator; }",
  "struct ReceivedItem { uint8 itemType; address token; uint256 identifier; uint256 amount; address recipient; }",
  "struct SpentItem { uint8 itemType; address token; uint256 identifier; uint256 amount; }",
  // functions
  "function cancel(OrderComponents[] orders) external returns (bool cancelled)",
  "function fulfillBasicOrder(BasicOrderParameters parameters) external payable returns (bool fulfilled)",
  "function fulfillBasicOrder_efficient_6GL6yc(BasicOrderParameters parameters) external payable returns (bool fulfilled)",
  "function fulfillOrder(Order order, bytes32 fulfillerConduitKey) external payable returns (bool fulfilled)",
  "function fulfillAdvancedOrder(AdvancedOrder advancedOrder, CriteriaResolver[] criteriaResolvers, bytes32 fulfillerConduitKey, address recipient) external payable returns (bool fulfilled)",
  "function fulfillAvailableOrders(Order[] orders, FulfillmentComponent[][] offerFulfillments, FulfillmentComponent[][] considerationFulfillments, bytes32 fulfillerConduitKey, uint256 maximumFulfilled) external payable returns (bool[] availableOrders, Execution[] executions)",
  "function fulfillAvailableAdvancedOrders(AdvancedOrder[] advancedOrders, CriteriaResolver[] criteriaResolvers, FulfillmentComponent[][] offerFulfillments, FulfillmentComponent[][] considerationFulfillments, bytes32 fulfillerConduitKey, address recipient, uint256 maximumFulfilled) external payable returns (bool[] availableOrders, Execution[] executions)",
  "function getContractOffererNonce(address contractOfferer) external view returns (uint256 nonce)",
  "function getOrderHash(OrderComponents order) external view returns (bytes32 orderHash)",
  "function getOrderStatus(bytes32 orderHash) external view returns (bool isValidated, bool isCancelled, uint256 totalFilled, uint256 totalSize)",
  "function getCounter(address offerer) external view returns (uint256 counter)",
  "function incrementCounter() external returns (uint256 newCounter)",
  "function information() external view returns (string version, bytes32 domainSeparator, address conduitController)",
  "function name() external view returns (string contractName)",
  "function matchAdvancedOrders(AdvancedOrder[] orders, CriteriaResolver[] criteriaResolvers, Fulfillment[] fulfillments) external payable returns (Execution[] executions)",
  "function matchOrders(Order[] orders, Fulfillment[] fulfillments) external payable returns (Execution[] executions)",
  "function validate(Order[] orders) external returns (bool validated)",
  // events
  "event CounterIncremented(uint256 newCounter, address offerer)",
  "event OrderCancelled(bytes32 orderHash, address offerer, address zone)",
  "event OrderFulfilled(bytes32 orderHash, address offerer, address zone, address recipient, SpentItem[] offer, ReceivedItem[] consideration)",
  "event OrdersMatched(bytes32[] orderHashes)",
  "event OrderValidated(bytes32 orderHash, address offerer, address zone)",
  // errors
  "error BadContractSignature()",
  "error BadFraction()",
  "error BadReturnValueFromERC20OnTransfer(address token, address from, address to, uint amount)",
  "error BadSignatureV(uint8 v)",
  "error CannotCancelOrder()",
  "error ConsiderationCriteriaResolverOutOfRange()",
  "error ConsiderationLengthNotEqualToTotalOriginal()",
  "error ConsiderationNotMet(uint orderIndex, uint considerationAmount, uint shortfallAmount)",
  "error CriteriaNotEnabledForItem()",
  "error ERC1155BatchTransferGenericFailure(address token, address from, address to, uint[] identifiers, uint[] amounts)",
  "error InexactFraction()",
  "error InsufficientNativeTokensSupplied()",
  "error Invalid1155BatchTransferEncoding()",
  "error InvalidBasicOrderParameterEncoding()",
  "error InvalidCallToConduit(address conduit)",
  "error InvalidConduit(bytes32 conduitKey, address conduit)",
  "error InvalidContractOrder(bytes32 orderHash)",
  "error InvalidERC721TransferAmount(uint256 amount)",
  "error InvalidFulfillmentComponentData()",
  "error InvalidMsgValue(uint256 value)",
  "error InvalidNativeOfferItem()",
  "error InvalidProof()",
  "error InvalidRestrictedOrder(bytes32 orderHash)",
  "error InvalidSignature()",
  "error InvalidSigner()",
  "error InvalidTime(uint256 startTime, uint256 endTime)",
  "error MismatchedFulfillmentOfferAndConsiderationComponents(uint256 fulfillmentIndex)",
  "error MissingFulfillmentComponentOnAggregation(uint8 side)",
  "error MissingItemAmount()",
  "error MissingOriginalConsiderationItems()",
  "error NativeTokenTransferGenericFailure(address account, uint256 amount)",
  "error NoContract(address account)",
  "error NoReentrantCalls()",
  "error NoSpecifiedOrdersAvailable()",
  "error OfferAndConsiderationRequiredOnFulfillment()",
  "error OfferCriteriaResolverOutOfRange()",
  "error OrderAlreadyFilled(bytes32 orderHash)",
  "error OrderCriteriaResolverOutOfRange(uint8 side)",
  "error OrderIsCancelled(bytes32 orderHash)",
  "error OrderPartiallyFilled(bytes32 orderHash)",
  "error PartialFillsNotEnabledForOrder()",
  "error TokenTransferGenericFailure(address token, address from, address to, uint identifier, uint amount)",
  "error UnresolvedConsiderationCriteria(uint orderIndex, uint considerationIndex)",
  "error UnresolvedOfferCriteria(uint256 orderIndex, uint256 offerIndex)",
  "error UnusedItemParameters()"
];
var wagmiMintExampleHumanReadableAbi = [
  "constructor()",
  "event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)",
  "event ApprovalForAll(address indexed owner, address indexed operator, bool approved)",
  "event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)",
  "function approve(address to, uint256 tokenId)",
  "function balanceOf(address owner) view returns (uint256)",
  "function getApproved(uint256 tokenId) view returns (address)",
  "function isApprovedForAll(address owner, address operator) view returns (bool)",
  "function mint()",
  "function name() view returns (string)",
  "function ownerOf(uint256 tokenId) view returns (address)",
  "function safeTransferFrom(address from, address to, uint256 tokenId)",
  "function safeTransferFrom(address from, address to, uint256 tokenId, bytes _data)",
  "function setApprovalForAll(address operator, bool approved)",
  "function supportsInterface(bytes4 interfaceId) view returns (bool)",
  "function symbol() view returns (string)",
  "function tokenURI(uint256 tokenId) pure returns (string)",
  "function totalSupply() view returns (uint256)",
  "function transferFrom(address from, address to, uint256 tokenId)"
];
var wethHumanReadableAbi = [
  "function name() view returns (string)",
  "function approve(address guy, uint wad) returns (bool)",
  "function totalSupply() view returns (uint)",
  "function transferFrom(address src, address dst, uint wad) returns (bool)",
  "function withdraw(uint wad)",
  "function decimals() view returns (uint8)",
  "function symbol() view returns (string)",
  "function balanceOf(address guy) view returns (uint256)",
  "function symbol() view returns (string)",
  "function transfer(address dst, uint wad) returns (bool)",
  "function deposit() payable",
  "function allowance(address src, address guy) view returns (uint256)",
  "event Approval(address indexed src, address indexed guy, uint wad)",
  "event Transfer(address indexed src, address indexed dst, uint wad)",
  "event Deposit(address indexed dst, uint wad)",
  "event Withdrawal(address indexed src, uint wad)",
  "fallback()"
];
var writingEditionsFactoryHumanReadableAbi = [
  "constructor(address _owner, address _treasuryConfiguration, uint256 _maxLimit, bool _guardOn)",
  "event BaseDescriptionURISet(address indexed clone, string oldBaseDescriptionURI, string newBaseDescriptionURI)",
  "event CloneDeployed(address indexed factory, address indexed owner, address indexed clone)",
  // Convert JSON ABI below to Human-Readable ABI string format
  "event EditionsDeployed(address indexed owner, address indexed clone, address indexed implementation)",
  "event FactoryGuardSet(bool guard)",
  "event FactoryImplementationSet(address indexed factory, address indexed oldImplementation, address indexed newImplementation)",
  "event FactoryLimitSet(address indexed factory, uint256 oldLimit, uint256 newLimit)",
  "event FundingRecipientSet(address indexed clone, address indexed oldFundingRecipient, address indexed newFundingRecipient)",
  "event NewImplementation(address indexed oldImplementation, address indexed newImplementation)",
  "event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)",
  "event PriceSet(address indexed clone, uint256 oldLimit, uint256 newLimit)",
  "event RendererSet(address indexed clone, address indexed renderer)",
  "event RoyaltyChange(address indexed clone, address indexed oldRoyaltyRecipient, uint256 oldRoyaltyBPS, address indexed newRoyaltyRecipient, uint256 newRoyaltyBPS)",
  "event Transfer(address indexed clone, address indexed from, address indexed to, uint256 indexed tokenId)",
  "event TributarySet(address indexed factory, address indexed clone, address indexed oldTributary, address indexed newTributary)",
  "event WritingEditionLimitSet(address indexed clone, uint256 oldLimit, uint256 newLimit)",
  "event WritingEditionPurchased(address indexed clone, uint256 indexed tokenId, address indexed recipient, uint256 price, string message)",
  "function CREATE_TYPEHASH() view returns (bytes32)",
  "function DOMAIN_SEPARATOR() view returns (bytes32)",
  "function VERSION() view returns (uint8)",
  "function acceptOwnership()",
  "function baseDescriptionURI() view returns (string)",
  "function cancelOwnershipTransfer()",
  "struct WritingEdition { string name; string symbol; string description; string imageURI; string contentURI; uint256 price; uint256 limit; address fundingRecipient; address renderer; uint256 nonce; uint16 fee; }",
  "function create(WritingEdition edition) returns (address clone)",
  "function createWithSignature(address owner, WritingEdition edition, uint8 v, bytes32 r, bytes32 s, address tokenRecipient, string message) payable returns (address clone)",
  "function getSalt(address owner, WritingEdition edition) view returns (bytes32)",
  "function guardOn() view returns (bool)",
  "function implementation() view returns (address)",
  "function isNextOwner() view returns (bool)",
  "function isOwner() view returns (bool)",
  "function isValid(address owner, bytes32 salt, uint8 v, bytes32 r, bytes32 s) view returns (bool)",
  "function maxLimit() view returns (uint256)",
  "function o11y() view returns (address)",
  "function owner() view returns (address)",
  "function predictDeterministicAddress(address _implementation, bytes32 salt) view returns (address)",
  "function purchaseThroughFactory(address clone, address tokenRecipient, string message) payable returns (uint256 tokenId)",
  "function renounceOwnership()",
  "function salts(bytes32) view returns (bool)",
  "function setGuard(bool _guardOn)",
  "function setImplementation(address _implementation)",
  "function setLimit(uint256 _maxLimit)",
  "function setTributary(address clone, address _tributary)",
  "function transferOwnership(address nextOwner_)",
  "function treasuryConfiguration() view returns (address)"
];

// src/test/index.ts
var address = "0x0000000000000000000000000000000000000000";
export {
  address,
  customSolidityErrorsAbi,
  customSolidityErrorsHumanReadableAbi,
  ensAbi,
  ensHumanReadableAbi,
  ensRegistryWithFallbackAbi,
  ensRegistryWithFallbackHumanReadableAbi,
  erc20Abi,
  erc20HumanReadableAbi,
  nestedTupleArrayAbi,
  nestedTupleArrayHumanReadableAbi,
  nounsAuctionHouseAbi,
  nounsAuctionHouseHumanReadableAbi,
  seaportAbi,
  seaportHumanReadableAbi,
  wagmiMintExampleAbi,
  wagmiMintExampleHumanReadableAbi,
  wethAbi,
  wethHumanReadableAbi,
  writingEditionsFactoryAbi,
  writingEditionsFactoryHumanReadableAbi
};
