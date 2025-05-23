declare const customSolidityErrorsAbi: readonly [{
    readonly inputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "constructor";
}, {
    readonly inputs: readonly [];
    readonly name: "ApprovalCallerNotOwnerNorApproved";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "ApprovalQueryForNonexistentToken";
    readonly type: "error";
}];
/**
 * ENS
 * https://etherscan.io/address/0x314159265dd8dbb310642f98f50c066173c1259b
 */
declare const ensAbi: readonly [{
    readonly constant: true;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "resolver";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly payable: false;
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "owner";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly payable: false;
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly name: "label";
        readonly type: "bytes32";
    }, {
        readonly name: "owner";
        readonly type: "address";
    }];
    readonly name: "setSubnodeOwner";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly name: "ttl";
        readonly type: "uint64";
    }];
    readonly name: "setTTL";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "ttl";
    readonly outputs: readonly [{
        readonly type: "uint64";
    }];
    readonly payable: false;
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly name: "resolver";
        readonly type: "address";
    }];
    readonly name: "setResolver";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly name: "owner";
        readonly type: "address";
    }];
    readonly name: "setOwner";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly name: "owner";
        readonly type: "address";
    }];
    readonly name: "Transfer";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: true;
        readonly name: "label";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly name: "owner";
        readonly type: "address";
    }];
    readonly name: "NewOwner";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly name: "resolver";
        readonly type: "address";
    }];
    readonly name: "NewResolver";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly name: "ttl";
        readonly type: "uint64";
    }];
    readonly name: "NewTTL";
    readonly type: "event";
}];
/**
 * ENSRegistryWithFallback
 * https://etherscan.io/address/0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e
 */
declare const ensRegistryWithFallbackAbi: readonly [{
    readonly inputs: readonly [{
        readonly internalType: "contract ENS";
        readonly name: "_old";
        readonly type: "address";
    }];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "constructor";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "operator";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "approved";
        readonly type: "bool";
    }];
    readonly name: "ApprovalForAll";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: true;
        readonly name: "label";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly name: "owner";
        readonly type: "address";
    }];
    readonly name: "NewOwner";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly name: "resolver";
        readonly type: "address";
    }];
    readonly name: "NewResolver";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly name: "ttl";
        readonly type: "uint64";
    }];
    readonly name: "NewTTL";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly name: "owner";
        readonly type: "address";
    }];
    readonly name: "Transfer";
    readonly type: "event";
}, {
    readonly constant: true;
    readonly inputs: readonly [{
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly name: "operator";
        readonly type: "address";
    }];
    readonly name: "isApprovedForAll";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [];
    readonly name: "old";
    readonly outputs: readonly [{
        readonly internalType: "contract ENS";
        readonly type: "address";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "owner";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "recordExists";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "resolver";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "operator";
        readonly type: "address";
    }, {
        readonly name: "approved";
        readonly type: "bool";
    }];
    readonly name: "setApprovalForAll";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly name: "owner";
        readonly type: "address";
    }];
    readonly name: "setOwner";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly name: "resolver";
        readonly type: "address";
    }, {
        readonly name: "ttl";
        readonly type: "uint64";
    }];
    readonly name: "setRecord";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly name: "resolver";
        readonly type: "address";
    }];
    readonly name: "setResolver";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly name: "label";
        readonly type: "bytes32";
    }, {
        readonly name: "owner";
        readonly type: "address";
    }];
    readonly name: "setSubnodeOwner";
    readonly outputs: readonly [{
        readonly type: "bytes32";
    }];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly name: "label";
        readonly type: "bytes32";
    }, {
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly name: "resolver";
        readonly type: "address";
    }, {
        readonly name: "ttl";
        readonly type: "uint64";
    }];
    readonly name: "setSubnodeRecord";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly name: "ttl";
        readonly type: "uint64";
    }];
    readonly name: "setTTL";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [{
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "ttl";
    readonly outputs: readonly [{
        readonly type: "uint64";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}];
/**
 * [ERC-20 Token Standard](https://ethereum.org/en/developers/docs/standards/tokens/erc-20)
 */
declare const erc20Abi: readonly [{
    readonly type: "event";
    readonly name: "Approval";
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "spender";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "value";
        readonly type: "uint256";
    }];
}, {
    readonly type: "event";
    readonly name: "Transfer";
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "from";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "value";
        readonly type: "uint256";
    }];
}, {
    readonly type: "function";
    readonly name: "allowance";
    readonly stateMutability: "view";
    readonly inputs: readonly [{
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly name: "spender";
        readonly type: "address";
    }];
    readonly outputs: readonly [{
        readonly name: "";
        readonly type: "uint256";
    }];
}, {
    readonly type: "function";
    readonly name: "approve";
    readonly stateMutability: "nonpayable";
    readonly inputs: readonly [{
        readonly name: "spender";
        readonly type: "address";
    }, {
        readonly name: "amount";
        readonly type: "uint256";
    }];
    readonly outputs: readonly [{
        readonly name: "";
        readonly type: "bool";
    }];
}, {
    readonly type: "function";
    readonly name: "balanceOf";
    readonly stateMutability: "view";
    readonly inputs: readonly [{
        readonly name: "account";
        readonly type: "address";
    }];
    readonly outputs: readonly [{
        readonly name: "";
        readonly type: "uint256";
    }];
}, {
    readonly type: "function";
    readonly name: "decimals";
    readonly stateMutability: "view";
    readonly inputs: readonly [];
    readonly outputs: readonly [{
        readonly name: "";
        readonly type: "uint8";
    }];
}, {
    readonly type: "function";
    readonly name: "name";
    readonly stateMutability: "view";
    readonly inputs: readonly [];
    readonly outputs: readonly [{
        readonly name: "";
        readonly type: "string";
    }];
}, {
    readonly type: "function";
    readonly name: "symbol";
    readonly stateMutability: "view";
    readonly inputs: readonly [];
    readonly outputs: readonly [{
        readonly name: "";
        readonly type: "string";
    }];
}, {
    readonly type: "function";
    readonly name: "totalSupply";
    readonly stateMutability: "view";
    readonly inputs: readonly [];
    readonly outputs: readonly [{
        readonly name: "";
        readonly type: "uint256";
    }];
}, {
    readonly type: "function";
    readonly name: "transfer";
    readonly stateMutability: "nonpayable";
    readonly inputs: readonly [{
        readonly name: "recipient";
        readonly type: "address";
    }, {
        readonly name: "amount";
        readonly type: "uint256";
    }];
    readonly outputs: readonly [{
        readonly name: "";
        readonly type: "bool";
    }];
}, {
    readonly type: "function";
    readonly name: "transferFrom";
    readonly stateMutability: "nonpayable";
    readonly inputs: readonly [{
        readonly name: "sender";
        readonly type: "address";
    }, {
        readonly name: "recipient";
        readonly type: "address";
    }, {
        readonly name: "amount";
        readonly type: "uint256";
    }];
    readonly outputs: readonly [{
        readonly name: "";
        readonly type: "bool";
    }];
}];
declare const nestedTupleArrayAbi: readonly [{
    readonly inputs: readonly [{
        readonly name: "s";
        readonly type: "tuple";
        readonly components: readonly [{
            readonly name: "a";
            readonly type: "uint8";
        }, {
            readonly name: "b";
            readonly type: "uint8[]";
        }, {
            readonly name: "c";
            readonly type: "tuple[]";
            readonly components: readonly [{
                readonly name: "x";
                readonly type: "uint8";
            }, {
                readonly name: "y";
                readonly type: "uint8";
            }];
        }];
    }, {
        readonly name: "t";
        readonly type: "tuple";
        readonly components: readonly [{
            readonly name: "x";
            readonly type: "uint";
        }, {
            readonly name: "y";
            readonly type: "uint";
        }];
    }, {
        readonly name: "a";
        readonly type: "uint256";
    }];
    readonly name: "f";
    readonly outputs: readonly [{
        readonly name: "t";
        readonly type: "tuple[]";
        readonly components: readonly [{
            readonly name: "x";
            readonly type: "uint256";
        }, {
            readonly name: "y";
            readonly type: "uint256";
        }];
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "s";
        readonly type: "tuple[2]";
        readonly components: readonly [{
            readonly name: "a";
            readonly type: "uint8";
        }, {
            readonly name: "b";
            readonly type: "uint8[]";
        }];
    }, {
        readonly name: "t";
        readonly type: "tuple";
        readonly components: readonly [{
            readonly name: "x";
            readonly type: "uint";
        }, {
            readonly name: "y";
            readonly type: "uint";
        }];
    }, {
        readonly name: "a";
        readonly type: "uint256";
    }];
    readonly name: "v";
    readonly outputs: readonly [];
    readonly stateMutability: "view";
    readonly type: "function";
}];
/**
 * NounsAuctionHouse
 * https://etherscan.io/address/0x5b2003ca8fe9ffb93684ce377f52b415c7dc0216
 */
declare const nounsAuctionHouseAbi: readonly [{
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "nounId";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "sender";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "value";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "extended";
        readonly type: "bool";
    }];
    readonly name: "AuctionBid";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "nounId";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "startTime";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "endTime";
        readonly type: "uint256";
    }];
    readonly name: "AuctionCreated";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "nounId";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "endTime";
        readonly type: "uint256";
    }];
    readonly name: "AuctionExtended";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "minBidIncrementPercentage";
        readonly type: "uint256";
    }];
    readonly name: "AuctionMinBidIncrementPercentageUpdated";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "reservePrice";
        readonly type: "uint256";
    }];
    readonly name: "AuctionReservePriceUpdated";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "nounId";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "winner";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "amount";
        readonly type: "uint256";
    }];
    readonly name: "AuctionSettled";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "timeBuffer";
        readonly type: "uint256";
    }];
    readonly name: "AuctionTimeBufferUpdated";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "previousOwner";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "newOwner";
        readonly type: "address";
    }];
    readonly name: "OwnershipTransferred";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "account";
        readonly type: "address";
    }];
    readonly name: "Paused";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "account";
        readonly type: "address";
    }];
    readonly name: "Unpaused";
    readonly type: "event";
}, {
    readonly inputs: readonly [];
    readonly name: "auction";
    readonly outputs: readonly [{
        readonly name: "nounId";
        readonly type: "uint256";
    }, {
        readonly name: "amount";
        readonly type: "uint256";
    }, {
        readonly name: "startTime";
        readonly type: "uint256";
    }, {
        readonly name: "endTime";
        readonly type: "uint256";
    }, {
        readonly internalType: "address payable";
        readonly name: "bidder";
        readonly type: "address";
    }, {
        readonly name: "settled";
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "nounId";
        readonly type: "uint256";
    }];
    readonly name: "createBid";
    readonly outputs: readonly [];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "duration";
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "contract INounsToken";
        readonly name: "_nouns";
        readonly type: "address";
    }, {
        readonly name: "_weth";
        readonly type: "address";
    }, {
        readonly name: "_timeBuffer";
        readonly type: "uint256";
    }, {
        readonly name: "_reservePrice";
        readonly type: "uint256";
    }, {
        readonly name: "_minBidIncrementPercentage";
        readonly type: "uint8";
    }, {
        readonly name: "_duration";
        readonly type: "uint256";
    }];
    readonly name: "initialize";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "minBidIncrementPercentage";
    readonly outputs: readonly [{
        readonly type: "uint8";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "nouns";
    readonly outputs: readonly [{
        readonly internalType: "contract INounsToken";
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "owner";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "pause";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "paused";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "renounceOwnership";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "reservePrice";
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "_minBidIncrementPercentage";
        readonly type: "uint8";
    }];
    readonly name: "setMinBidIncrementPercentage";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "_reservePrice";
        readonly type: "uint256";
    }];
    readonly name: "setReservePrice";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "_timeBuffer";
        readonly type: "uint256";
    }];
    readonly name: "setTimeBuffer";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "settleAuction";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "settleCurrentAndCreateNewAuction";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "timeBuffer";
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "newOwner";
        readonly type: "address";
    }];
    readonly name: "transferOwnership";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "unpause";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "weth";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}];
/**
 * Seaport
 * https://etherscan.io/address/0x00000000000001ad428e4906ae43d8f9852d0dd6
 */
declare const seaportAbi: readonly [{
    readonly inputs: readonly [{
        readonly name: "conduitController";
        readonly type: "address";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "constructor";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly name: "offerer";
            readonly type: "address";
        }, {
            readonly name: "zone";
            readonly type: "address";
        }, {
            readonly components: readonly [{
                readonly name: "itemType";
                readonly type: "uint8";
            }, {
                readonly name: "token";
                readonly type: "address";
            }, {
                readonly name: "identifierOrCriteria";
                readonly type: "uint256";
            }, {
                readonly name: "startAmount";
                readonly type: "uint256";
            }, {
                readonly name: "endAmount";
                readonly type: "uint256";
            }];
            readonly name: "offer";
            readonly type: "tuple[]";
        }, {
            readonly components: readonly [{
                readonly name: "itemType";
                readonly type: "uint8";
            }, {
                readonly name: "token";
                readonly type: "address";
            }, {
                readonly name: "identifierOrCriteria";
                readonly type: "uint256";
            }, {
                readonly name: "startAmount";
                readonly type: "uint256";
            }, {
                readonly name: "endAmount";
                readonly type: "uint256";
            }, {
                readonly name: "recipient";
                readonly type: "address";
            }];
            readonly name: "consideration";
            readonly type: "tuple[]";
        }, {
            readonly name: "orderType";
            readonly type: "uint8";
        }, {
            readonly name: "startTime";
            readonly type: "uint256";
        }, {
            readonly name: "endTime";
            readonly type: "uint256";
        }, {
            readonly name: "zoneHash";
            readonly type: "bytes32";
        }, {
            readonly name: "salt";
            readonly type: "uint256";
        }, {
            readonly name: "conduitKey";
            readonly type: "bytes32";
        }, {
            readonly name: "counter";
            readonly type: "uint256";
        }];
        readonly name: "orders";
        readonly type: "tuple[]";
    }];
    readonly name: "cancel";
    readonly outputs: readonly [{
        readonly name: "cancelled";
        readonly type: "bool";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "offerer";
                readonly type: "address";
            }, {
                readonly name: "zone";
                readonly type: "address";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }];
                readonly name: "offer";
                readonly type: "tuple[]";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "recipient";
                    readonly type: "address";
                }];
                readonly name: "consideration";
                readonly type: "tuple[]";
            }, {
                readonly name: "orderType";
                readonly type: "uint8";
            }, {
                readonly name: "startTime";
                readonly type: "uint256";
            }, {
                readonly name: "endTime";
                readonly type: "uint256";
            }, {
                readonly name: "zoneHash";
                readonly type: "bytes32";
            }, {
                readonly name: "salt";
                readonly type: "uint256";
            }, {
                readonly name: "conduitKey";
                readonly type: "bytes32";
            }, {
                readonly name: "totalOriginalConsiderationItems";
                readonly type: "uint256";
            }];
            readonly internalType: "structOrderParameters";
            readonly name: "parameters";
            readonly type: "tuple";
        }, {
            readonly name: "numerator";
            readonly type: "uint120";
        }, {
            readonly name: "denominator";
            readonly type: "uint120";
        }, {
            readonly name: "signature";
            readonly type: "bytes";
        }, {
            readonly name: "extraData";
            readonly type: "bytes";
        }];
        readonly internalType: "structAdvancedOrder";
        readonly name: "advancedOrder";
        readonly type: "tuple";
    }, {
        readonly components: readonly [{
            readonly name: "orderIndex";
            readonly type: "uint256";
        }, {
            readonly internalType: "enumSide";
            readonly name: "side";
            readonly type: "uint8";
        }, {
            readonly name: "index";
            readonly type: "uint256";
        }, {
            readonly name: "identifier";
            readonly type: "uint256";
        }, {
            readonly name: "criteriaProof";
            readonly type: "bytes32[]";
        }];
        readonly internalType: "structCriteriaResolver[]";
        readonly name: "criteriaResolvers";
        readonly type: "tuple[]";
    }, {
        readonly name: "fulfillerConduitKey";
        readonly type: "bytes32";
    }, {
        readonly name: "recipient";
        readonly type: "address";
    }];
    readonly name: "fulfillAdvancedOrder";
    readonly outputs: readonly [{
        readonly name: "fulfilled";
        readonly type: "bool";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "offerer";
                readonly type: "address";
            }, {
                readonly name: "zone";
                readonly type: "address";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }];
                readonly name: "offer";
                readonly type: "tuple[]";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "recipient";
                    readonly type: "address";
                }];
                readonly name: "consideration";
                readonly type: "tuple[]";
            }, {
                readonly name: "orderType";
                readonly type: "uint8";
            }, {
                readonly name: "startTime";
                readonly type: "uint256";
            }, {
                readonly name: "endTime";
                readonly type: "uint256";
            }, {
                readonly name: "zoneHash";
                readonly type: "bytes32";
            }, {
                readonly name: "salt";
                readonly type: "uint256";
            }, {
                readonly name: "conduitKey";
                readonly type: "bytes32";
            }, {
                readonly name: "totalOriginalConsiderationItems";
                readonly type: "uint256";
            }];
            readonly internalType: "structOrderParameters";
            readonly name: "parameters";
            readonly type: "tuple";
        }, {
            readonly name: "numerator";
            readonly type: "uint120";
        }, {
            readonly name: "denominator";
            readonly type: "uint120";
        }, {
            readonly name: "signature";
            readonly type: "bytes";
        }, {
            readonly name: "extraData";
            readonly type: "bytes";
        }];
        readonly internalType: "structAdvancedOrder[]";
        readonly name: "advancedOrders";
        readonly type: "tuple[]";
    }, {
        readonly components: readonly [{
            readonly name: "orderIndex";
            readonly type: "uint256";
        }, {
            readonly internalType: "enumSide";
            readonly name: "side";
            readonly type: "uint8";
        }, {
            readonly name: "index";
            readonly type: "uint256";
        }, {
            readonly name: "identifier";
            readonly type: "uint256";
        }, {
            readonly name: "criteriaProof";
            readonly type: "bytes32[]";
        }];
        readonly internalType: "structCriteriaResolver[]";
        readonly name: "criteriaResolvers";
        readonly type: "tuple[]";
    }, {
        readonly components: readonly [{
            readonly name: "orderIndex";
            readonly type: "uint256";
        }, {
            readonly name: "itemIndex";
            readonly type: "uint256";
        }];
        readonly internalType: "structFulfillmentComponent[][]";
        readonly name: "offerFulfillments";
        readonly type: "tuple[][]";
    }, {
        readonly components: readonly [{
            readonly name: "orderIndex";
            readonly type: "uint256";
        }, {
            readonly name: "itemIndex";
            readonly type: "uint256";
        }];
        readonly internalType: "structFulfillmentComponent[][]";
        readonly name: "considerationFulfillments";
        readonly type: "tuple[][]";
    }, {
        readonly name: "fulfillerConduitKey";
        readonly type: "bytes32";
    }, {
        readonly name: "recipient";
        readonly type: "address";
    }, {
        readonly name: "maximumFulfilled";
        readonly type: "uint256";
    }];
    readonly name: "fulfillAvailableAdvancedOrders";
    readonly outputs: readonly [{
        readonly name: "availableOrders";
        readonly type: "bool[]";
    }, {
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "itemType";
                readonly type: "uint8";
            }, {
                readonly name: "token";
                readonly type: "address";
            }, {
                readonly name: "identifier";
                readonly type: "uint256";
            }, {
                readonly name: "amount";
                readonly type: "uint256";
            }, {
                readonly name: "recipient";
                readonly type: "address";
            }];
            readonly internalType: "structReceivedItem";
            readonly name: "item";
            readonly type: "tuple";
        }, {
            readonly name: "offerer";
            readonly type: "address";
        }, {
            readonly name: "conduitKey";
            readonly type: "bytes32";
        }];
        readonly internalType: "structExecution[]";
        readonly name: "executions";
        readonly type: "tuple[]";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "offerer";
                readonly type: "address";
            }, {
                readonly name: "zone";
                readonly type: "address";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }];
                readonly name: "offer";
                readonly type: "tuple[]";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "recipient";
                    readonly type: "address";
                }];
                readonly name: "consideration";
                readonly type: "tuple[]";
            }, {
                readonly name: "orderType";
                readonly type: "uint8";
            }, {
                readonly name: "startTime";
                readonly type: "uint256";
            }, {
                readonly name: "endTime";
                readonly type: "uint256";
            }, {
                readonly name: "zoneHash";
                readonly type: "bytes32";
            }, {
                readonly name: "salt";
                readonly type: "uint256";
            }, {
                readonly name: "conduitKey";
                readonly type: "bytes32";
            }, {
                readonly name: "totalOriginalConsiderationItems";
                readonly type: "uint256";
            }];
            readonly internalType: "structOrderParameters";
            readonly name: "parameters";
            readonly type: "tuple";
        }, {
            readonly name: "signature";
            readonly type: "bytes";
        }];
        readonly internalType: "structOrder[]";
        readonly name: "orders";
        readonly type: "tuple[]";
    }, {
        readonly components: readonly [{
            readonly name: "orderIndex";
            readonly type: "uint256";
        }, {
            readonly name: "itemIndex";
            readonly type: "uint256";
        }];
        readonly internalType: "structFulfillmentComponent[][]";
        readonly name: "offerFulfillments";
        readonly type: "tuple[][]";
    }, {
        readonly components: readonly [{
            readonly name: "orderIndex";
            readonly type: "uint256";
        }, {
            readonly name: "itemIndex";
            readonly type: "uint256";
        }];
        readonly internalType: "structFulfillmentComponent[][]";
        readonly name: "considerationFulfillments";
        readonly type: "tuple[][]";
    }, {
        readonly name: "fulfillerConduitKey";
        readonly type: "bytes32";
    }, {
        readonly name: "maximumFulfilled";
        readonly type: "uint256";
    }];
    readonly name: "fulfillAvailableOrders";
    readonly outputs: readonly [{
        readonly name: "availableOrders";
        readonly type: "bool[]";
    }, {
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "itemType";
                readonly type: "uint8";
            }, {
                readonly name: "token";
                readonly type: "address";
            }, {
                readonly name: "identifier";
                readonly type: "uint256";
            }, {
                readonly name: "amount";
                readonly type: "uint256";
            }, {
                readonly name: "recipient";
                readonly type: "address";
            }];
            readonly internalType: "structReceivedItem";
            readonly name: "item";
            readonly type: "tuple";
        }, {
            readonly name: "offerer";
            readonly type: "address";
        }, {
            readonly name: "conduitKey";
            readonly type: "bytes32";
        }];
        readonly internalType: "structExecution[]";
        readonly name: "executions";
        readonly type: "tuple[]";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly name: "considerationToken";
            readonly type: "address";
        }, {
            readonly name: "considerationIdentifier";
            readonly type: "uint256";
        }, {
            readonly name: "considerationAmount";
            readonly type: "uint256";
        }, {
            readonly name: "offerer";
            readonly type: "address";
        }, {
            readonly name: "zone";
            readonly type: "address";
        }, {
            readonly name: "offerToken";
            readonly type: "address";
        }, {
            readonly name: "offerIdentifier";
            readonly type: "uint256";
        }, {
            readonly name: "offerAmount";
            readonly type: "uint256";
        }, {
            readonly internalType: "enumBasicOrderType";
            readonly name: "basicOrderType";
            readonly type: "uint8";
        }, {
            readonly name: "startTime";
            readonly type: "uint256";
        }, {
            readonly name: "endTime";
            readonly type: "uint256";
        }, {
            readonly name: "zoneHash";
            readonly type: "bytes32";
        }, {
            readonly name: "salt";
            readonly type: "uint256";
        }, {
            readonly name: "offererConduitKey";
            readonly type: "bytes32";
        }, {
            readonly name: "fulfillerConduitKey";
            readonly type: "bytes32";
        }, {
            readonly name: "totalOriginalAdditionalRecipients";
            readonly type: "uint256";
        }, {
            readonly components: readonly [{
                readonly name: "amount";
                readonly type: "uint256";
            }, {
                readonly name: "recipient";
                readonly type: "address";
            }];
            readonly internalType: "structAdditionalRecipient[]";
            readonly name: "additionalRecipients";
            readonly type: "tuple[]";
        }, {
            readonly name: "signature";
            readonly type: "bytes";
        }];
        readonly internalType: "structBasicOrderParameters";
        readonly name: "parameters";
        readonly type: "tuple";
    }];
    readonly name: "fulfillBasicOrder";
    readonly outputs: readonly [{
        readonly name: "fulfilled";
        readonly type: "bool";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly name: "considerationToken";
            readonly type: "address";
        }, {
            readonly name: "considerationIdentifier";
            readonly type: "uint256";
        }, {
            readonly name: "considerationAmount";
            readonly type: "uint256";
        }, {
            readonly name: "offerer";
            readonly type: "address";
        }, {
            readonly name: "zone";
            readonly type: "address";
        }, {
            readonly name: "offerToken";
            readonly type: "address";
        }, {
            readonly name: "offerIdentifier";
            readonly type: "uint256";
        }, {
            readonly name: "offerAmount";
            readonly type: "uint256";
        }, {
            readonly internalType: "enumBasicOrderType";
            readonly name: "basicOrderType";
            readonly type: "uint8";
        }, {
            readonly name: "startTime";
            readonly type: "uint256";
        }, {
            readonly name: "endTime";
            readonly type: "uint256";
        }, {
            readonly name: "zoneHash";
            readonly type: "bytes32";
        }, {
            readonly name: "salt";
            readonly type: "uint256";
        }, {
            readonly name: "offererConduitKey";
            readonly type: "bytes32";
        }, {
            readonly name: "fulfillerConduitKey";
            readonly type: "bytes32";
        }, {
            readonly name: "totalOriginalAdditionalRecipients";
            readonly type: "uint256";
        }, {
            readonly components: readonly [{
                readonly name: "amount";
                readonly type: "uint256";
            }, {
                readonly name: "recipient";
                readonly type: "address";
            }];
            readonly internalType: "structAdditionalRecipient[]";
            readonly name: "additionalRecipients";
            readonly type: "tuple[]";
        }, {
            readonly name: "signature";
            readonly type: "bytes";
        }];
        readonly internalType: "structBasicOrderParameters";
        readonly name: "parameters";
        readonly type: "tuple";
    }];
    readonly name: "fulfillBasicOrder_efficient_6GL6yc";
    readonly outputs: readonly [{
        readonly name: "fulfilled";
        readonly type: "bool";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "offerer";
                readonly type: "address";
            }, {
                readonly name: "zone";
                readonly type: "address";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }];
                readonly name: "offer";
                readonly type: "tuple[]";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "recipient";
                    readonly type: "address";
                }];
                readonly name: "consideration";
                readonly type: "tuple[]";
            }, {
                readonly name: "orderType";
                readonly type: "uint8";
            }, {
                readonly name: "startTime";
                readonly type: "uint256";
            }, {
                readonly name: "endTime";
                readonly type: "uint256";
            }, {
                readonly name: "zoneHash";
                readonly type: "bytes32";
            }, {
                readonly name: "salt";
                readonly type: "uint256";
            }, {
                readonly name: "conduitKey";
                readonly type: "bytes32";
            }, {
                readonly name: "totalOriginalConsiderationItems";
                readonly type: "uint256";
            }];
            readonly internalType: "structOrderParameters";
            readonly name: "parameters";
            readonly type: "tuple";
        }, {
            readonly name: "signature";
            readonly type: "bytes";
        }];
        readonly internalType: "structOrder";
        readonly name: "order";
        readonly type: "tuple";
    }, {
        readonly name: "fulfillerConduitKey";
        readonly type: "bytes32";
    }];
    readonly name: "fulfillOrder";
    readonly outputs: readonly [{
        readonly name: "fulfilled";
        readonly type: "bool";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "contractOfferer";
        readonly type: "address";
    }];
    readonly name: "getContractOffererNonce";
    readonly outputs: readonly [{
        readonly name: "nonce";
        readonly type: "uint256";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "offerer";
        readonly type: "address";
    }];
    readonly name: "getCounter";
    readonly outputs: readonly [{
        readonly name: "counter";
        readonly type: "uint256";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly name: "offerer";
            readonly type: "address";
        }, {
            readonly name: "zone";
            readonly type: "address";
        }, {
            readonly components: readonly [{
                readonly name: "itemType";
                readonly type: "uint8";
            }, {
                readonly name: "token";
                readonly type: "address";
            }, {
                readonly name: "identifierOrCriteria";
                readonly type: "uint256";
            }, {
                readonly name: "startAmount";
                readonly type: "uint256";
            }, {
                readonly name: "endAmount";
                readonly type: "uint256";
            }];
            readonly name: "offer";
            readonly type: "tuple[]";
        }, {
            readonly components: readonly [{
                readonly name: "itemType";
                readonly type: "uint8";
            }, {
                readonly name: "token";
                readonly type: "address";
            }, {
                readonly name: "identifierOrCriteria";
                readonly type: "uint256";
            }, {
                readonly name: "startAmount";
                readonly type: "uint256";
            }, {
                readonly name: "endAmount";
                readonly type: "uint256";
            }, {
                readonly name: "recipient";
                readonly type: "address";
            }];
            readonly name: "consideration";
            readonly type: "tuple[]";
        }, {
            readonly name: "orderType";
            readonly type: "uint8";
        }, {
            readonly name: "startTime";
            readonly type: "uint256";
        }, {
            readonly name: "endTime";
            readonly type: "uint256";
        }, {
            readonly name: "zoneHash";
            readonly type: "bytes32";
        }, {
            readonly name: "salt";
            readonly type: "uint256";
        }, {
            readonly name: "conduitKey";
            readonly type: "bytes32";
        }, {
            readonly name: "counter";
            readonly type: "uint256";
        }];
        readonly internalType: "structOrderComponents";
        readonly name: "order";
        readonly type: "tuple";
    }];
    readonly name: "getOrderHash";
    readonly outputs: readonly [{
        readonly name: "orderHash";
        readonly type: "bytes32";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "orderHash";
        readonly type: "bytes32";
    }];
    readonly name: "getOrderStatus";
    readonly outputs: readonly [{
        readonly name: "isValidated";
        readonly type: "bool";
    }, {
        readonly name: "isCancelled";
        readonly type: "bool";
    }, {
        readonly name: "totalFilled";
        readonly type: "uint256";
    }, {
        readonly name: "totalSize";
        readonly type: "uint256";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "incrementCounter";
    readonly outputs: readonly [{
        readonly name: "newCounter";
        readonly type: "uint256";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "information";
    readonly outputs: readonly [{
        readonly name: "version";
        readonly type: "string";
    }, {
        readonly name: "domainSeparator";
        readonly type: "bytes32";
    }, {
        readonly name: "conduitController";
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "offerer";
                readonly type: "address";
            }, {
                readonly name: "zone";
                readonly type: "address";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }];
                readonly name: "offer";
                readonly type: "tuple[]";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "recipient";
                    readonly type: "address";
                }];
                readonly name: "consideration";
                readonly type: "tuple[]";
            }, {
                readonly name: "orderType";
                readonly type: "uint8";
            }, {
                readonly name: "startTime";
                readonly type: "uint256";
            }, {
                readonly name: "endTime";
                readonly type: "uint256";
            }, {
                readonly name: "zoneHash";
                readonly type: "bytes32";
            }, {
                readonly name: "salt";
                readonly type: "uint256";
            }, {
                readonly name: "conduitKey";
                readonly type: "bytes32";
            }, {
                readonly name: "totalOriginalConsiderationItems";
                readonly type: "uint256";
            }];
            readonly internalType: "structOrderParameters";
            readonly name: "parameters";
            readonly type: "tuple";
        }, {
            readonly name: "numerator";
            readonly type: "uint120";
        }, {
            readonly name: "denominator";
            readonly type: "uint120";
        }, {
            readonly name: "signature";
            readonly type: "bytes";
        }, {
            readonly name: "extraData";
            readonly type: "bytes";
        }];
        readonly internalType: "structAdvancedOrder[]";
        readonly name: "orders";
        readonly type: "tuple[]";
    }, {
        readonly components: readonly [{
            readonly name: "orderIndex";
            readonly type: "uint256";
        }, {
            readonly internalType: "enumSide";
            readonly name: "side";
            readonly type: "uint8";
        }, {
            readonly name: "index";
            readonly type: "uint256";
        }, {
            readonly name: "identifier";
            readonly type: "uint256";
        }, {
            readonly name: "criteriaProof";
            readonly type: "bytes32[]";
        }];
        readonly internalType: "structCriteriaResolver[]";
        readonly name: "criteriaResolvers";
        readonly type: "tuple[]";
    }, {
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "orderIndex";
                readonly type: "uint256";
            }, {
                readonly name: "itemIndex";
                readonly type: "uint256";
            }];
            readonly internalType: "structFulfillmentComponent[]";
            readonly name: "offerComponents";
            readonly type: "tuple[]";
        }, {
            readonly components: readonly [{
                readonly name: "orderIndex";
                readonly type: "uint256";
            }, {
                readonly name: "itemIndex";
                readonly type: "uint256";
            }];
            readonly internalType: "structFulfillmentComponent[]";
            readonly name: "considerationComponents";
            readonly type: "tuple[]";
        }];
        readonly internalType: "structFulfillment[]";
        readonly name: "fulfillments";
        readonly type: "tuple[]";
    }, {
        readonly name: "recipient";
        readonly type: "address";
    }];
    readonly name: "matchAdvancedOrders";
    readonly outputs: readonly [{
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "itemType";
                readonly type: "uint8";
            }, {
                readonly name: "token";
                readonly type: "address";
            }, {
                readonly name: "identifier";
                readonly type: "uint256";
            }, {
                readonly name: "amount";
                readonly type: "uint256";
            }, {
                readonly name: "recipient";
                readonly type: "address";
            }];
            readonly internalType: "structReceivedItem";
            readonly name: "item";
            readonly type: "tuple";
        }, {
            readonly name: "offerer";
            readonly type: "address";
        }, {
            readonly name: "conduitKey";
            readonly type: "bytes32";
        }];
        readonly internalType: "structExecution[]";
        readonly name: "executions";
        readonly type: "tuple[]";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "offerer";
                readonly type: "address";
            }, {
                readonly name: "zone";
                readonly type: "address";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }];
                readonly name: "offer";
                readonly type: "tuple[]";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "recipient";
                    readonly type: "address";
                }];
                readonly name: "consideration";
                readonly type: "tuple[]";
            }, {
                readonly name: "orderType";
                readonly type: "uint8";
            }, {
                readonly name: "startTime";
                readonly type: "uint256";
            }, {
                readonly name: "endTime";
                readonly type: "uint256";
            }, {
                readonly name: "zoneHash";
                readonly type: "bytes32";
            }, {
                readonly name: "salt";
                readonly type: "uint256";
            }, {
                readonly name: "conduitKey";
                readonly type: "bytes32";
            }, {
                readonly name: "totalOriginalConsiderationItems";
                readonly type: "uint256";
            }];
            readonly internalType: "structOrderParameters";
            readonly name: "parameters";
            readonly type: "tuple";
        }, {
            readonly name: "signature";
            readonly type: "bytes";
        }];
        readonly internalType: "structOrder[]";
        readonly name: "orders";
        readonly type: "tuple[]";
    }, {
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "orderIndex";
                readonly type: "uint256";
            }, {
                readonly name: "itemIndex";
                readonly type: "uint256";
            }];
            readonly internalType: "structFulfillmentComponent[]";
            readonly name: "offerComponents";
            readonly type: "tuple[]";
        }, {
            readonly components: readonly [{
                readonly name: "orderIndex";
                readonly type: "uint256";
            }, {
                readonly name: "itemIndex";
                readonly type: "uint256";
            }];
            readonly internalType: "structFulfillmentComponent[]";
            readonly name: "considerationComponents";
            readonly type: "tuple[]";
        }];
        readonly internalType: "structFulfillment[]";
        readonly name: "fulfillments";
        readonly type: "tuple[]";
    }];
    readonly name: "matchOrders";
    readonly outputs: readonly [{
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "itemType";
                readonly type: "uint8";
            }, {
                readonly name: "token";
                readonly type: "address";
            }, {
                readonly name: "identifier";
                readonly type: "uint256";
            }, {
                readonly name: "amount";
                readonly type: "uint256";
            }, {
                readonly name: "recipient";
                readonly type: "address";
            }];
            readonly internalType: "structReceivedItem";
            readonly name: "item";
            readonly type: "tuple";
        }, {
            readonly name: "offerer";
            readonly type: "address";
        }, {
            readonly name: "conduitKey";
            readonly type: "bytes32";
        }];
        readonly internalType: "structExecution[]";
        readonly name: "executions";
        readonly type: "tuple[]";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "name";
    readonly outputs: readonly [{
        readonly name: "contractName";
        readonly type: "string";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly components: readonly [{
                readonly name: "offerer";
                readonly type: "address";
            }, {
                readonly name: "zone";
                readonly type: "address";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }];
                readonly name: "offer";
                readonly type: "tuple[]";
            }, {
                readonly components: readonly [{
                    readonly internalType: "enumItemType";
                    readonly name: "itemType";
                    readonly type: "uint8";
                }, {
                    readonly name: "token";
                    readonly type: "address";
                }, {
                    readonly name: "identifierOrCriteria";
                    readonly type: "uint256";
                }, {
                    readonly name: "startAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "endAmount";
                    readonly type: "uint256";
                }, {
                    readonly name: "recipient";
                    readonly type: "address";
                }];
                readonly name: "consideration";
                readonly type: "tuple[]";
            }, {
                readonly name: "orderType";
                readonly type: "uint8";
            }, {
                readonly name: "startTime";
                readonly type: "uint256";
            }, {
                readonly name: "endTime";
                readonly type: "uint256";
            }, {
                readonly name: "zoneHash";
                readonly type: "bytes32";
            }, {
                readonly name: "salt";
                readonly type: "uint256";
            }, {
                readonly name: "conduitKey";
                readonly type: "bytes32";
            }, {
                readonly name: "totalOriginalConsiderationItems";
                readonly type: "uint256";
            }];
            readonly internalType: "structOrderParameters";
            readonly name: "parameters";
            readonly type: "tuple";
        }, {
            readonly name: "signature";
            readonly type: "bytes";
        }];
        readonly internalType: "structOrder[]";
        readonly name: "orders";
        readonly type: "tuple[]";
    }];
    readonly name: "validate";
    readonly outputs: readonly [{
        readonly name: "validated";
        readonly type: "bool";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "BadContractSignature";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "BadFraction";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "token";
        readonly type: "address";
    }, {
        readonly name: "from";
        readonly type: "address";
    }, {
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly name: "amount";
        readonly type: "uint256";
    }];
    readonly name: "BadReturnValueFromERC20OnTransfer";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "v";
        readonly type: "uint8";
    }];
    readonly name: "BadSignatureV";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "CannotCancelOrder";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "ConsiderationCriteriaResolverOutOfRange";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "ConsiderationLengthNotEqualToTotalOriginal";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "orderIndex";
        readonly type: "uint256";
    }, {
        readonly name: "considerationIndex";
        readonly type: "uint256";
    }, {
        readonly name: "shortfallAmount";
        readonly type: "uint256";
    }];
    readonly name: "ConsiderationNotMet";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "CriteriaNotEnabledForItem";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "token";
        readonly type: "address";
    }, {
        readonly name: "from";
        readonly type: "address";
    }, {
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly name: "identifiers";
        readonly type: "uint256[]";
    }, {
        readonly name: "amounts";
        readonly type: "uint256[]";
    }];
    readonly name: "ERC1155BatchTransferGenericFailure";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "InexactFraction";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "InsufficientNativeTokensSupplied";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "Invalid1155BatchTransferEncoding";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "InvalidBasicOrderParameterEncoding";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "conduit";
        readonly type: "address";
    }];
    readonly name: "InvalidCallToConduit";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "conduitKey";
        readonly type: "bytes32";
    }, {
        readonly name: "conduit";
        readonly type: "address";
    }];
    readonly name: "InvalidConduit";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "orderHash";
        readonly type: "bytes32";
    }];
    readonly name: "InvalidContractOrder";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "amount";
        readonly type: "uint256";
    }];
    readonly name: "InvalidERC721TransferAmount";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "InvalidFulfillmentComponentData";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "value";
        readonly type: "uint256";
    }];
    readonly name: "InvalidMsgValue";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "InvalidNativeOfferItem";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "InvalidProof";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "orderHash";
        readonly type: "bytes32";
    }];
    readonly name: "InvalidRestrictedOrder";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "InvalidSignature";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "InvalidSigner";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "startTime";
        readonly type: "uint256";
    }, {
        readonly name: "endTime";
        readonly type: "uint256";
    }];
    readonly name: "InvalidTime";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "fulfillmentIndex";
        readonly type: "uint256";
    }];
    readonly name: "MismatchedFulfillmentOfferAndConsiderationComponents";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly internalType: "enumSide";
        readonly name: "side";
        readonly type: "uint8";
    }];
    readonly name: "MissingFulfillmentComponentOnAggregation";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "MissingItemAmount";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "MissingOriginalConsiderationItems";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "account";
        readonly type: "address";
    }, {
        readonly name: "amount";
        readonly type: "uint256";
    }];
    readonly name: "NativeTokenTransferGenericFailure";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "account";
        readonly type: "address";
    }];
    readonly name: "NoContract";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "NoReentrantCalls";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "NoSpecifiedOrdersAvailable";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "OfferAndConsiderationRequiredOnFulfillment";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "OfferCriteriaResolverOutOfRange";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "orderHash";
        readonly type: "bytes32";
    }];
    readonly name: "OrderAlreadyFilled";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly internalType: "enumSide";
        readonly name: "side";
        readonly type: "uint8";
    }];
    readonly name: "OrderCriteriaResolverOutOfRange";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "orderHash";
        readonly type: "bytes32";
    }];
    readonly name: "OrderIsCancelled";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "orderHash";
        readonly type: "bytes32";
    }];
    readonly name: "OrderPartiallyFilled";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "PartialFillsNotEnabledForOrder";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "token";
        readonly type: "address";
    }, {
        readonly name: "from";
        readonly type: "address";
    }, {
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly name: "identifier";
        readonly type: "uint256";
    }, {
        readonly name: "amount";
        readonly type: "uint256";
    }];
    readonly name: "TokenTransferGenericFailure";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "orderIndex";
        readonly type: "uint256";
    }, {
        readonly name: "considerationIndex";
        readonly type: "uint256";
    }];
    readonly name: "UnresolvedConsiderationCriteria";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly name: "orderIndex";
        readonly type: "uint256";
    }, {
        readonly name: "offerIndex";
        readonly type: "uint256";
    }];
    readonly name: "UnresolvedOfferCriteria";
    readonly type: "error";
}, {
    readonly inputs: readonly [];
    readonly name: "UnusedItemParameters";
    readonly type: "error";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "newCounter";
        readonly type: "uint256";
    }, {
        readonly indexed: true;
        readonly name: "offerer";
        readonly type: "address";
    }];
    readonly name: "CounterIncremented";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "orderHash";
        readonly type: "bytes32";
    }, {
        readonly indexed: true;
        readonly name: "offerer";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "zone";
        readonly type: "address";
    }];
    readonly name: "OrderCancelled";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "orderHash";
        readonly type: "bytes32";
    }, {
        readonly indexed: true;
        readonly name: "offerer";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "zone";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "recipient";
        readonly type: "address";
    }, {
        readonly components: readonly [{
            readonly name: "itemType";
            readonly type: "uint8";
        }, {
            readonly name: "token";
            readonly type: "address";
        }, {
            readonly name: "identifier";
            readonly type: "uint256";
        }, {
            readonly name: "amount";
            readonly type: "uint256";
        }];
        readonly indexed: false;
        readonly internalType: "structSpentItem[]";
        readonly name: "offer";
        readonly type: "tuple[]";
    }, {
        readonly components: readonly [{
            readonly name: "itemType";
            readonly type: "uint8";
        }, {
            readonly name: "token";
            readonly type: "address";
        }, {
            readonly name: "identifier";
            readonly type: "uint256";
        }, {
            readonly name: "amount";
            readonly type: "uint256";
        }, {
            readonly name: "recipient";
            readonly type: "address";
        }];
        readonly indexed: false;
        readonly internalType: "structReceivedItem[]";
        readonly name: "consideration";
        readonly type: "tuple[]";
    }];
    readonly name: "OrderFulfilled";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "orderHash";
        readonly type: "bytes32";
    }, {
        readonly components: readonly [{
            readonly name: "offerer";
            readonly type: "address";
        }, {
            readonly name: "zone";
            readonly type: "address";
        }, {
            readonly components: readonly [{
                readonly name: "itemType";
                readonly type: "uint8";
            }, {
                readonly name: "token";
                readonly type: "address";
            }, {
                readonly name: "identifierOrCriteria";
                readonly type: "uint256";
            }, {
                readonly name: "startAmount";
                readonly type: "uint256";
            }, {
                readonly name: "endAmount";
                readonly type: "uint256";
            }];
            readonly name: "offer";
            readonly type: "tuple[]";
        }, {
            readonly components: readonly [{
                readonly name: "itemType";
                readonly type: "uint8";
            }, {
                readonly name: "token";
                readonly type: "address";
            }, {
                readonly name: "identifierOrCriteria";
                readonly type: "uint256";
            }, {
                readonly name: "startAmount";
                readonly type: "uint256";
            }, {
                readonly name: "endAmount";
                readonly type: "uint256";
            }, {
                readonly name: "recipient";
                readonly type: "address";
            }];
            readonly name: "consideration";
            readonly type: "tuple[]";
        }, {
            readonly name: "orderType";
            readonly type: "uint8";
        }, {
            readonly name: "startTime";
            readonly type: "uint256";
        }, {
            readonly name: "endTime";
            readonly type: "uint256";
        }, {
            readonly name: "zoneHash";
            readonly type: "bytes32";
        }, {
            readonly name: "salt";
            readonly type: "uint256";
        }, {
            readonly name: "conduitKey";
            readonly type: "bytes32";
        }, {
            readonly name: "totalOriginalConsiderationItems";
            readonly type: "uint256";
        }];
        readonly indexed: false;
        readonly internalType: "structOrderParameters";
        readonly name: "orderParameters";
        readonly type: "tuple";
    }];
    readonly name: "OrderValidated";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "orderHashes";
        readonly type: "bytes32[]";
    }];
    readonly name: "OrdersMatched";
    readonly type: "event";
}];
/**
 * WagmiMintExample
 * https://etherscan.io/address/0xaf0326d92b97df1221759476b072abfd8084f9be
 */
declare const wagmiMintExampleAbi: readonly [{
    readonly inputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "constructor";
}, {
    readonly inputs: readonly [{
        readonly name: "owner";
        readonly type: "address";
        readonly indexed: true;
    }, {
        readonly name: "approved";
        readonly type: "address";
        readonly indexed: true;
    }, {
        readonly name: "tokenId";
        readonly type: "uint256";
        readonly indexed: true;
    }];
    readonly name: "Approval";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly name: "owner";
        readonly type: "address";
        readonly indexed: true;
    }, {
        readonly name: "operator";
        readonly type: "address";
        readonly indexed: true;
    }, {
        readonly name: "approved";
        readonly type: "bool";
        readonly indexed: false;
    }];
    readonly name: "ApprovalForAll";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly name: "from";
        readonly type: "address";
        readonly indexed: true;
    }, {
        readonly name: "to";
        readonly type: "address";
        readonly indexed: true;
    }, {
        readonly name: "tokenId";
        readonly type: "uint256";
        readonly indexed: true;
    }];
    readonly name: "Transfer";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly name: "tokenId";
        readonly type: "uint256";
    }];
    readonly name: "approve";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "owner";
        readonly type: "address";
    }];
    readonly name: "balanceOf";
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "tokenId";
        readonly type: "uint256";
    }];
    readonly name: "getApproved";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly name: "operator";
        readonly type: "address";
    }];
    readonly name: "isApprovedForAll";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "mint";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "name";
    readonly outputs: readonly [{
        readonly type: "string";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "tokenId";
        readonly type: "uint256";
    }];
    readonly name: "ownerOf";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "from";
        readonly type: "address";
    }, {
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly name: "tokenId";
        readonly type: "uint256";
    }];
    readonly name: "safeTransferFrom";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "from";
        readonly type: "address";
    }, {
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly name: "tokenId";
        readonly type: "uint256";
    }, {
        readonly name: "_data";
        readonly type: "bytes";
    }];
    readonly name: "safeTransferFrom";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "operator";
        readonly type: "address";
    }, {
        readonly name: "approved";
        readonly type: "bool";
    }];
    readonly name: "setApprovalForAll";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "interfaceId";
        readonly type: "bytes4";
    }];
    readonly name: "supportsInterface";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "symbol";
    readonly outputs: readonly [{
        readonly type: "string";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "tokenId";
        readonly type: "uint256";
    }];
    readonly name: "tokenURI";
    readonly outputs: readonly [{
        readonly type: "string";
    }];
    readonly stateMutability: "pure";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "totalSupply";
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "from";
        readonly type: "address";
    }, {
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly name: "tokenId";
        readonly type: "uint256";
    }];
    readonly name: "transferFrom";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}];
/**
 * WETH
 * https://etherscan.io/token/0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2
 */
declare const wethAbi: readonly [{
    readonly constant: true;
    readonly inputs: readonly [];
    readonly name: "name";
    readonly outputs: readonly [{
        readonly type: "string";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "guy";
        readonly type: "address";
    }, {
        readonly name: "wad";
        readonly type: "uint256";
    }];
    readonly name: "approve";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [];
    readonly name: "totalSupply";
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "src";
        readonly type: "address";
    }, {
        readonly name: "dst";
        readonly type: "address";
    }, {
        readonly name: "wad";
        readonly type: "uint256";
    }];
    readonly name: "transferFrom";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "wad";
        readonly type: "uint256";
    }];
    readonly name: "withdraw";
    readonly outputs: readonly [];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [];
    readonly name: "decimals";
    readonly outputs: readonly [{
        readonly type: "uint8";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [{
        readonly type: "address";
    }];
    readonly name: "balanceOf";
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [];
    readonly name: "symbol";
    readonly outputs: readonly [{
        readonly type: "string";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [{
        readonly name: "dst";
        readonly type: "address";
    }, {
        readonly name: "wad";
        readonly type: "uint256";
    }];
    readonly name: "transfer";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly payable: false;
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly constant: false;
    readonly inputs: readonly [];
    readonly name: "deposit";
    readonly outputs: readonly [];
    readonly payable: true;
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly constant: true;
    readonly inputs: readonly [{
        readonly type: "address";
    }, {
        readonly type: "address";
    }];
    readonly name: "allowance";
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
    readonly payable: false;
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly payable: true;
    readonly stateMutability: "payable";
    readonly type: "fallback";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "src";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "guy";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "wad";
        readonly type: "uint256";
    }];
    readonly name: "Approval";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "src";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "dst";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "wad";
        readonly type: "uint256";
    }];
    readonly name: "Transfer";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "dst";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "wad";
        readonly type: "uint256";
    }];
    readonly name: "Deposit";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "src";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "wad";
        readonly type: "uint256";
    }];
    readonly name: "Withdrawal";
    readonly type: "event";
}];
/**
 * WritingEditionsFactory
 * https://optimistic.etherscan.io/address/0x302f746eE2fDC10DDff63188f71639094717a766
 */
declare const writingEditionsFactoryAbi: readonly [{
    readonly inputs: readonly [{
        readonly name: "_owner";
        readonly type: "address";
    }, {
        readonly name: "_treasuryConfiguration";
        readonly type: "address";
    }, {
        readonly name: "_maxLimit";
        readonly type: "uint256";
    }, {
        readonly name: "_guardOn";
        readonly type: "bool";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "constructor";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "oldBaseDescriptionURI";
        readonly type: "string";
    }, {
        readonly indexed: false;
        readonly name: "newBaseDescriptionURI";
        readonly type: "string";
    }];
    readonly name: "BaseDescriptionURISet";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "factory";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }];
    readonly name: "CloneDeployed";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "implementation";
        readonly type: "address";
    }];
    readonly name: "EditionsDeployed";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "guard";
        readonly type: "bool";
    }];
    readonly name: "FactoryGuardSet";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "factory";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "oldImplementation";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "newImplementation";
        readonly type: "address";
    }];
    readonly name: "FactoryImplementationSet";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "factory";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "oldLimit";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "newLimit";
        readonly type: "uint256";
    }];
    readonly name: "FactoryLimitSet";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "oldFundingRecipient";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "newFundingRecipient";
        readonly type: "address";
    }];
    readonly name: "FundingRecipientSet";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "oldImplementation";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "newImplementation";
        readonly type: "address";
    }];
    readonly name: "NewImplementation";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "previousOwner";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "newOwner";
        readonly type: "address";
    }];
    readonly name: "OwnershipTransferred";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "oldLimit";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "newLimit";
        readonly type: "uint256";
    }];
    readonly name: "PriceSet";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "renderer";
        readonly type: "address";
    }];
    readonly name: "RendererSet";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "oldRoyaltyRecipient";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "oldRoyaltyBPS";
        readonly type: "uint256";
    }, {
        readonly indexed: true;
        readonly name: "newRoyaltyRecipient";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "newRoyaltyBPS";
        readonly type: "uint256";
    }];
    readonly name: "RoyaltyChange";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "from";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "tokenId";
        readonly type: "uint256";
    }];
    readonly name: "Transfer";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "factory";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "oldTributary";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly name: "newTributary";
        readonly type: "address";
    }];
    readonly name: "TributarySet";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "oldLimit";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "newLimit";
        readonly type: "uint256";
    }];
    readonly name: "WritingEditionLimitSet";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "tokenId";
        readonly type: "uint256";
    }, {
        readonly indexed: true;
        readonly name: "recipient";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly name: "price";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "message";
        readonly type: "string";
    }];
    readonly name: "WritingEditionPurchased";
    readonly type: "event";
}, {
    readonly inputs: readonly [];
    readonly name: "CREATE_TYPEHASH";
    readonly outputs: readonly [{
        readonly type: "bytes32";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "DOMAIN_SEPARATOR";
    readonly outputs: readonly [{
        readonly type: "bytes32";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "VERSION";
    readonly outputs: readonly [{
        readonly type: "uint8";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "acceptOwnership";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "baseDescriptionURI";
    readonly outputs: readonly [{
        readonly type: "string";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "cancelOwnershipTransfer";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly components: readonly [{
            readonly name: "name";
            readonly type: "string";
        }, {
            readonly name: "symbol";
            readonly type: "string";
        }, {
            readonly name: "description";
            readonly type: "string";
        }, {
            readonly name: "imageURI";
            readonly type: "string";
        }, {
            readonly name: "contentURI";
            readonly type: "string";
        }, {
            readonly name: "price";
            readonly type: "uint8";
        }, {
            readonly name: "limit";
            readonly type: "uint256";
        }, {
            readonly name: "fundingRecipient";
            readonly type: "address";
        }, {
            readonly name: "renderer";
            readonly type: "address";
        }, {
            readonly name: "nonce";
            readonly type: "uint256";
        }, {
            readonly name: "fee";
            readonly type: "uint16";
        }];
        readonly internalType: "struct IWritingEditions.WritingEdition";
        readonly name: "edition";
        readonly type: "tuple";
    }];
    readonly name: "create";
    readonly outputs: readonly [{
        readonly name: "clone";
        readonly type: "address";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly components: readonly [{
            readonly name: "name";
            readonly type: "string";
        }, {
            readonly name: "symbol";
            readonly type: "string";
        }, {
            readonly name: "description";
            readonly type: "string";
        }, {
            readonly name: "imageURI";
            readonly type: "string";
        }, {
            readonly name: "contentURI";
            readonly type: "string";
        }, {
            readonly name: "price";
            readonly type: "uint256";
        }, {
            readonly name: "limit";
            readonly type: "uint256";
        }, {
            readonly name: "fundingRecipient";
            readonly type: "address";
        }, {
            readonly name: "renderer";
            readonly type: "address";
        }, {
            readonly name: "nonce";
            readonly type: "uint256";
        }, {
            readonly name: "fee";
            readonly type: "uint16";
        }];
        readonly internalType: "struct IWritingEditions.WritingEdition";
        readonly name: "edition";
        readonly type: "tuple";
    }, {
        readonly name: "v";
        readonly type: "uint8";
    }, {
        readonly name: "r";
        readonly type: "bytes32";
    }, {
        readonly name: "s";
        readonly type: "bytes32";
    }, {
        readonly name: "tokenRecipient";
        readonly type: "address";
    }, {
        readonly name: "message";
        readonly type: "string";
    }];
    readonly name: "createWithSignature";
    readonly outputs: readonly [{
        readonly name: "clone";
        readonly type: "address";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly components: readonly [{
            readonly name: "name";
            readonly type: "string";
        }, {
            readonly name: "symbol";
            readonly type: "string";
        }, {
            readonly name: "description";
            readonly type: "string";
        }, {
            readonly name: "imageURI";
            readonly type: "string";
        }, {
            readonly name: "contentURI";
            readonly type: "string";
        }, {
            readonly name: "price";
            readonly type: "uint8";
        }, {
            readonly name: "limit";
            readonly type: "uint256";
        }, {
            readonly name: "fundingRecipient";
            readonly type: "address";
        }, {
            readonly name: "renderer";
            readonly type: "address";
        }, {
            readonly name: "nonce";
            readonly type: "uint256";
        }, {
            readonly name: "fee";
            readonly type: "uint16";
        }];
        readonly internalType: "struct IWritingEditions.WritingEdition";
        readonly name: "edition";
        readonly type: "tuple";
    }];
    readonly name: "getSalt";
    readonly outputs: readonly [{
        readonly type: "bytes32";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "guardOn";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "implementation";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "isNextOwner";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "isOwner";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly name: "salt";
        readonly type: "bytes32";
    }, {
        readonly name: "v";
        readonly type: "uint8";
    }, {
        readonly name: "r";
        readonly type: "bytes32";
    }, {
        readonly name: "s";
        readonly type: "bytes32";
    }];
    readonly name: "isValid";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "maxLimit";
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "o11y";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "owner";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "_implementation";
        readonly type: "address";
    }, {
        readonly name: "salt";
        readonly type: "bytes32";
    }];
    readonly name: "predictDeterministicAddress";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly name: "tokenRecipient";
        readonly type: "address";
    }, {
        readonly name: "message";
        readonly type: "string";
    }];
    readonly name: "purchaseThroughFactory";
    readonly outputs: readonly [{
        readonly name: "tokenId";
        readonly type: "uint256";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "renounceOwnership";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly type: "bytes32";
    }];
    readonly name: "salts";
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "_guardOn";
        readonly type: "bool";
    }];
    readonly name: "setGuard";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "_implementation";
        readonly type: "address";
    }];
    readonly name: "setImplementation";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "_maxLimit";
        readonly type: "uint256";
    }];
    readonly name: "setLimit";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "clone";
        readonly type: "address";
    }, {
        readonly name: "_tributary";
        readonly type: "address";
    }];
    readonly name: "setTributary";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly name: "nextOwner_";
        readonly type: "address";
    }];
    readonly name: "transferOwnership";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "treasuryConfiguration";
    readonly outputs: readonly [{
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}];

declare const customSolidityErrorsHumanReadableAbi: readonly ["constructor()", "error ApprovalCallerNotOwnerNorApproved()", "error ApprovalQueryForNonexistentToken()"];
/**
 * ENS
 * https://etherscan.io/address/0x314159265dd8dbb310642f98f50c066173c1259b
 */
declare const ensHumanReadableAbi: readonly ["function resolver(bytes32 node) view returns (address)", "function owner(bytes32 node) view returns (address)", "function setSubnodeOwner(bytes32 node, bytes32 label, address owner)", "function setTTL(bytes32 node, uint64 ttl)", "function ttl(bytes32 node) view returns (uint64)", "function setResolver(bytes32 node, address resolver)", "function setOwner(bytes32 node, address owner)", "event Transfer(bytes32 indexed node, address owner)", "event NewOwner(bytes32 indexed node, bytes32 indexed label, address owner)", "event NewResolver(bytes32 indexed node, address resolver)", "event NewTTL(bytes32 indexed node, uint64 ttl)"];
/**
 * ENSRegistryWithFallback
 * https://etherscan.io/address/0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e
 */
declare const ensRegistryWithFallbackHumanReadableAbi: readonly ["constructor(address _old)", "event ApprovalForAll(address indexed owner, address indexed operator, bool approved)", "event NewOwner(bytes32 indexed node, bytes32 indexed label, address owner)", "event NewResolver(bytes32 indexed node, address resolver)", "event NewTTL(bytes32 indexed node, uint64 ttl)", "event Transfer(bytes32 indexed node, address owner)", "function isApprovedForAll(address owner, address operator) view returns (bool)", "function old() view returns (address)", "function owner(bytes32 node) view returns (address)", "function recordExists(bytes32 node) view returns (bool)", "function resolver(bytes32 node) view returns (address)", "function setApprovalForAll(address operator, bool approved)", "function setOwner(bytes32 node, address owner)", "function setRecord(bytes32 node, address owner, address resolver, uint64 ttl)", "function setResolver(bytes32 node, address resolver)", "function setSubnodeOwner(bytes32 node, bytes32 label, address owner)", "function setSubnodeRecord(bytes32 node, bytes32 label, address owner, address resolver, uint64 ttl)", "function setTTL(bytes32 node, uint64 ttl)", "function ttl(bytes32 node) view returns (uint64)"];
/**
 * [ERC-20 Token Standard](https://ethereum.org/en/developers/docs/standards/tokens/erc-20)
 */
declare const erc20HumanReadableAbi: readonly ["event Approval(address indexed owner, address indexed spender, uint256 value)", "event Transfer(address indexed from, address indexed to, uint256 value)", "function allowance(address owner, address spender) view returns (uint256)", "function approve(address spender, uint256 amount) returns (bool)", "function balanceOf(address account) view returns (uint256)", "function decimals() view returns (uint8)", "function name() view returns (string)", "function symbol() view returns (string)", "function totalSupply() view returns (uint256)", "function transfer(address recipient, uint256 amount) returns (bool)", "function transferFrom(address sender, address recipient, uint256 amount) returns (bool)"];
declare const nestedTupleArrayHumanReadableAbi: readonly ["function f((uint8 a, uint8[] b, (uint8 x, uint8 y)[] c) s, (uint x, uint y) t, uint256 a) returns ((uint256 x, uint256 y)[] t)", "function v((uint8 a, uint8[] b) s, (uint x, uint y) t, uint256 a)"];
/**
 * NounsAuctionHouse
 * https://etherscan.io/address/0x5b2003ca8fe9ffb93684ce377f52b415c7dc0216
 */
declare const nounsAuctionHouseHumanReadableAbi: readonly ["event AuctionBid(uint256 indexed nounId, address sender, uint256 value, bool extended)", "event AuctionCreated(uint256 indexed nounId, uint256 startTime, uint256 endTime)", "event AuctionExtended(uint256 indexed nounId, uint256 endTime)", "event AuctionMinBidIncrementPercentageUpdated(uint256 minBidIncrementPercentage)", "event AuctionReservePriceUpdated(uint256 reservePrice)", "event AuctionSettled(uint256 indexed nounId, address winner, uint256 amount)", "event AuctionTimeBufferUpdated(uint256 timeBuffer)", "event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)", "event Paused(address account)", "event Unpaused(address account)", "function auction(uint256 nounId) view returns (uint256 nounId, uint256 amount, uint256 startTime, uint256 endTime, address bidder, bool settled)", "function createBid(uint256 nounId) payable", "function duration() view returns (uint256)", "function initialize(address _nouns, address _weth, uint256 _timeBuffer, uint256 _reservePrice, uint8 _minBidIncrementPercentage, uint256 _duration)", "function minBidIncrementPercentage() view returns (uint8)", "function nouns() view returns (address)", "function owner() view returns (address)", "function pause()", "function paused() view returns (bool)", "function renounceOwnership()", "function reservePrice() view returns (uint256)", "function setMinBidIncrementPercentage(uint8 _minBidIncrementPercentage)", "function setReservePrice(uint256 _reservePrice)", "function setTimeBuffer(uint256 _timeBuffer)", "function settleAuction()", "function settleCurrentAndCreateNewAuction()", "function timeBuffer() view returns (uint256)", "function newOwner() view returns (address)", "function unpause()", "function weth() view returns (address)"];
/**
 * Seaport
 * https://etherscan.io/address/0x00000000000001ad428e4906ae43d8f9852d0dd6
 */
declare const seaportHumanReadableAbi: readonly ["constructor(address conduitController)", "struct AdditionalRecipient { uint256 amount; address recipient; }", "struct AdvancedOrder { OrderParameters parameters; uint120 numerator; uint120 denominator; bytes signature; bytes extraData; }", "struct BasicOrderParameters { address considerationToken; uint256 considerationIdentifier; uint256 considerationAmount; address offerer; address zone; address offerToken; uint256 offerIdentifier; uint256 offerAmount; uint8 basicOrderType; uint256 startTime; uint256 endTime; bytes32 zoneHash; uint256 salt; bytes32 offererConduitKey; bytes32 fulfillerConduitKey; uint256 totalOriginalAdditionalRecipients; AdditionalRecipient[] additionalRecipients; bytes signature; }", "struct ConsiderationItem { uint8 itemType; address token; uint256 identifierOrCriteria; uint256 startAmount; uint256 endAmount; address recipient; }", "struct CriteriaResolver { uint256 orderIndex; uint8 side; uint256 index; uint256 identifier; bytes32[] criteriaProof; }", "struct Execution { ReceivedItem item; address offerer; bytes32 conduitKey; }", "struct Fulfillment { FulfillmentComponent[] offerComponents; FulfillmentComponent[] considerationComponents; }", "struct FulfillmentComponent { uint256 orderIndex; uint256 itemIndex; }", "struct OfferItem { uint8 itemType; address token; uint256 identifierOrCriteria; uint256 startAmount; uint256 endAmount; }", "struct Order { OrderParameters parameters; bytes signature; }", "struct OrderComponents { address offerer; address zone; OfferItem[] offer; ConsiderationItem[] consideration; uint8 orderType; uint256 startTime; uint256 endTime; bytes32 zoneHash; uint256 salt; bytes32 conduitKey; uint256 counter; }", "struct OrderParameters { address offerer; address zone; OfferItem[] offer; ConsiderationItem[] consideration; uint8 orderType; uint256 startTime; uint256 endTime; bytes32 zoneHash; uint256 salt; bytes32 conduitKey; uint256 totalOriginalConsiderationItems; }", "struct OrderStatus { bool isValidated; bool isCancelled; uint120 numerator; uint120 denominator; }", "struct ReceivedItem { uint8 itemType; address token; uint256 identifier; uint256 amount; address recipient; }", "struct SpentItem { uint8 itemType; address token; uint256 identifier; uint256 amount; }", "function cancel(OrderComponents[] orders) external returns (bool cancelled)", "function fulfillBasicOrder(BasicOrderParameters parameters) external payable returns (bool fulfilled)", "function fulfillBasicOrder_efficient_6GL6yc(BasicOrderParameters parameters) external payable returns (bool fulfilled)", "function fulfillOrder(Order order, bytes32 fulfillerConduitKey) external payable returns (bool fulfilled)", "function fulfillAdvancedOrder(AdvancedOrder advancedOrder, CriteriaResolver[] criteriaResolvers, bytes32 fulfillerConduitKey, address recipient) external payable returns (bool fulfilled)", "function fulfillAvailableOrders(Order[] orders, FulfillmentComponent[][] offerFulfillments, FulfillmentComponent[][] considerationFulfillments, bytes32 fulfillerConduitKey, uint256 maximumFulfilled) external payable returns (bool[] availableOrders, Execution[] executions)", "function fulfillAvailableAdvancedOrders(AdvancedOrder[] advancedOrders, CriteriaResolver[] criteriaResolvers, FulfillmentComponent[][] offerFulfillments, FulfillmentComponent[][] considerationFulfillments, bytes32 fulfillerConduitKey, address recipient, uint256 maximumFulfilled) external payable returns (bool[] availableOrders, Execution[] executions)", "function getContractOffererNonce(address contractOfferer) external view returns (uint256 nonce)", "function getOrderHash(OrderComponents order) external view returns (bytes32 orderHash)", "function getOrderStatus(bytes32 orderHash) external view returns (bool isValidated, bool isCancelled, uint256 totalFilled, uint256 totalSize)", "function getCounter(address offerer) external view returns (uint256 counter)", "function incrementCounter() external returns (uint256 newCounter)", "function information() external view returns (string version, bytes32 domainSeparator, address conduitController)", "function name() external view returns (string contractName)", "function matchAdvancedOrders(AdvancedOrder[] orders, CriteriaResolver[] criteriaResolvers, Fulfillment[] fulfillments) external payable returns (Execution[] executions)", "function matchOrders(Order[] orders, Fulfillment[] fulfillments) external payable returns (Execution[] executions)", "function validate(Order[] orders) external returns (bool validated)", "event CounterIncremented(uint256 newCounter, address offerer)", "event OrderCancelled(bytes32 orderHash, address offerer, address zone)", "event OrderFulfilled(bytes32 orderHash, address offerer, address zone, address recipient, SpentItem[] offer, ReceivedItem[] consideration)", "event OrdersMatched(bytes32[] orderHashes)", "event OrderValidated(bytes32 orderHash, address offerer, address zone)", "error BadContractSignature()", "error BadFraction()", "error BadReturnValueFromERC20OnTransfer(address token, address from, address to, uint amount)", "error BadSignatureV(uint8 v)", "error CannotCancelOrder()", "error ConsiderationCriteriaResolverOutOfRange()", "error ConsiderationLengthNotEqualToTotalOriginal()", "error ConsiderationNotMet(uint orderIndex, uint considerationAmount, uint shortfallAmount)", "error CriteriaNotEnabledForItem()", "error ERC1155BatchTransferGenericFailure(address token, address from, address to, uint[] identifiers, uint[] amounts)", "error InexactFraction()", "error InsufficientNativeTokensSupplied()", "error Invalid1155BatchTransferEncoding()", "error InvalidBasicOrderParameterEncoding()", "error InvalidCallToConduit(address conduit)", "error InvalidConduit(bytes32 conduitKey, address conduit)", "error InvalidContractOrder(bytes32 orderHash)", "error InvalidERC721TransferAmount(uint256 amount)", "error InvalidFulfillmentComponentData()", "error InvalidMsgValue(uint256 value)", "error InvalidNativeOfferItem()", "error InvalidProof()", "error InvalidRestrictedOrder(bytes32 orderHash)", "error InvalidSignature()", "error InvalidSigner()", "error InvalidTime(uint256 startTime, uint256 endTime)", "error MismatchedFulfillmentOfferAndConsiderationComponents(uint256 fulfillmentIndex)", "error MissingFulfillmentComponentOnAggregation(uint8 side)", "error MissingItemAmount()", "error MissingOriginalConsiderationItems()", "error NativeTokenTransferGenericFailure(address account, uint256 amount)", "error NoContract(address account)", "error NoReentrantCalls()", "error NoSpecifiedOrdersAvailable()", "error OfferAndConsiderationRequiredOnFulfillment()", "error OfferCriteriaResolverOutOfRange()", "error OrderAlreadyFilled(bytes32 orderHash)", "error OrderCriteriaResolverOutOfRange(uint8 side)", "error OrderIsCancelled(bytes32 orderHash)", "error OrderPartiallyFilled(bytes32 orderHash)", "error PartialFillsNotEnabledForOrder()", "error TokenTransferGenericFailure(address token, address from, address to, uint identifier, uint amount)", "error UnresolvedConsiderationCriteria(uint orderIndex, uint considerationIndex)", "error UnresolvedOfferCriteria(uint256 orderIndex, uint256 offerIndex)", "error UnusedItemParameters()"];
/**
 * WagmiMintExample
 * https://etherscan.io/address/0xaf0326d92b97df1221759476b072abfd8084f9be
 */
declare const wagmiMintExampleHumanReadableAbi: readonly ["constructor()", "event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId)", "event ApprovalForAll(address indexed owner, address indexed operator, bool approved)", "event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)", "function approve(address to, uint256 tokenId)", "function balanceOf(address owner) view returns (uint256)", "function getApproved(uint256 tokenId) view returns (address)", "function isApprovedForAll(address owner, address operator) view returns (bool)", "function mint()", "function name() view returns (string)", "function ownerOf(uint256 tokenId) view returns (address)", "function safeTransferFrom(address from, address to, uint256 tokenId)", "function safeTransferFrom(address from, address to, uint256 tokenId, bytes _data)", "function setApprovalForAll(address operator, bool approved)", "function supportsInterface(bytes4 interfaceId) view returns (bool)", "function symbol() view returns (string)", "function tokenURI(uint256 tokenId) pure returns (string)", "function totalSupply() view returns (uint256)", "function transferFrom(address from, address to, uint256 tokenId)"];
/**
 * WETH
 * https://etherscan.io/token/0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2
 */
declare const wethHumanReadableAbi: readonly ["function name() view returns (string)", "function approve(address guy, uint wad) returns (bool)", "function totalSupply() view returns (uint)", "function transferFrom(address src, address dst, uint wad) returns (bool)", "function withdraw(uint wad)", "function decimals() view returns (uint8)", "function symbol() view returns (string)", "function balanceOf(address guy) view returns (uint256)", "function symbol() view returns (string)", "function transfer(address dst, uint wad) returns (bool)", "function deposit() payable", "function allowance(address src, address guy) view returns (uint256)", "event Approval(address indexed src, address indexed guy, uint wad)", "event Transfer(address indexed src, address indexed dst, uint wad)", "event Deposit(address indexed dst, uint wad)", "event Withdrawal(address indexed src, uint wad)", "fallback()"];
/**
 * WritingEditionsFactory
 * https://optimistic.etherscan.io/address/0x302f746eE2fDC10DDff63188f71639094717a766
 */
declare const writingEditionsFactoryHumanReadableAbi: readonly ["constructor(address _owner, address _treasuryConfiguration, uint256 _maxLimit, bool _guardOn)", "event BaseDescriptionURISet(address indexed clone, string oldBaseDescriptionURI, string newBaseDescriptionURI)", "event CloneDeployed(address indexed factory, address indexed owner, address indexed clone)", "event EditionsDeployed(address indexed owner, address indexed clone, address indexed implementation)", "event FactoryGuardSet(bool guard)", "event FactoryImplementationSet(address indexed factory, address indexed oldImplementation, address indexed newImplementation)", "event FactoryLimitSet(address indexed factory, uint256 oldLimit, uint256 newLimit)", "event FundingRecipientSet(address indexed clone, address indexed oldFundingRecipient, address indexed newFundingRecipient)", "event NewImplementation(address indexed oldImplementation, address indexed newImplementation)", "event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)", "event PriceSet(address indexed clone, uint256 oldLimit, uint256 newLimit)", "event RendererSet(address indexed clone, address indexed renderer)", "event RoyaltyChange(address indexed clone, address indexed oldRoyaltyRecipient, uint256 oldRoyaltyBPS, address indexed newRoyaltyRecipient, uint256 newRoyaltyBPS)", "event Transfer(address indexed clone, address indexed from, address indexed to, uint256 indexed tokenId)", "event TributarySet(address indexed factory, address indexed clone, address indexed oldTributary, address indexed newTributary)", "event WritingEditionLimitSet(address indexed clone, uint256 oldLimit, uint256 newLimit)", "event WritingEditionPurchased(address indexed clone, uint256 indexed tokenId, address indexed recipient, uint256 price, string message)", "function CREATE_TYPEHASH() view returns (bytes32)", "function DOMAIN_SEPARATOR() view returns (bytes32)", "function VERSION() view returns (uint8)", "function acceptOwnership()", "function baseDescriptionURI() view returns (string)", "function cancelOwnershipTransfer()", "struct WritingEdition { string name; string symbol; string description; string imageURI; string contentURI; uint256 price; uint256 limit; address fundingRecipient; address renderer; uint256 nonce; uint16 fee; }", "function create(WritingEdition edition) returns (address clone)", "function createWithSignature(address owner, WritingEdition edition, uint8 v, bytes32 r, bytes32 s, address tokenRecipient, string message) payable returns (address clone)", "function getSalt(address owner, WritingEdition edition) view returns (bytes32)", "function guardOn() view returns (bool)", "function implementation() view returns (address)", "function isNextOwner() view returns (bool)", "function isOwner() view returns (bool)", "function isValid(address owner, bytes32 salt, uint8 v, bytes32 r, bytes32 s) view returns (bool)", "function maxLimit() view returns (uint256)", "function o11y() view returns (address)", "function owner() view returns (address)", "function predictDeterministicAddress(address _implementation, bytes32 salt) view returns (address)", "function purchaseThroughFactory(address clone, address tokenRecipient, string message) payable returns (uint256 tokenId)", "function renounceOwnership()", "function salts(bytes32) view returns (bool)", "function setGuard(bool _guardOn)", "function setImplementation(address _implementation)", "function setLimit(uint256 _maxLimit)", "function setTributary(address clone, address _tributary)", "function transferOwnership(address nextOwner_)", "function treasuryConfiguration() view returns (address)"];

declare const address: "0x0000000000000000000000000000000000000000";

export { address, customSolidityErrorsAbi, customSolidityErrorsHumanReadableAbi, ensAbi, ensHumanReadableAbi, ensRegistryWithFallbackAbi, ensRegistryWithFallbackHumanReadableAbi, erc20Abi, erc20HumanReadableAbi, nestedTupleArrayAbi, nestedTupleArrayHumanReadableAbi, nounsAuctionHouseAbi, nounsAuctionHouseHumanReadableAbi, seaportAbi, seaportHumanReadableAbi, wagmiMintExampleAbi, wagmiMintExampleHumanReadableAbi, wethAbi, wethHumanReadableAbi, writingEditionsFactoryAbi, writingEditionsFactoryHumanReadableAbi };
