export declare const PublicResolverAbi: readonly [{
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly internalType: "address";
        readonly name: "a";
        readonly type: "address";
    }];
    readonly name: "AddrChanged";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly internalType: "uint256";
        readonly name: "coinType";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly internalType: "bytes";
        readonly name: "newAddress";
        readonly type: "bytes";
    }];
    readonly name: "AddressChanged";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "address";
        readonly name: "owner";
        readonly type: "address";
    }, {
        readonly indexed: true;
        readonly internalType: "address";
        readonly name: "operator";
        readonly type: "address";
    }, {
        readonly indexed: false;
        readonly internalType: "bool";
        readonly name: "approved";
        readonly type: "bool";
    }];
    readonly name: "ApprovalForAll";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly internalType: "bytes";
        readonly name: "hash";
        readonly type: "bytes";
    }];
    readonly name: "ContenthashChanged";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly internalType: "bytes";
        readonly name: "name";
        readonly type: "bytes";
    }, {
        readonly indexed: false;
        readonly internalType: "uint16";
        readonly name: "resource";
        readonly type: "uint16";
    }, {
        readonly indexed: false;
        readonly internalType: "bytes";
        readonly name: "record";
        readonly type: "bytes";
    }];
    readonly name: "DNSRecordChanged";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly internalType: "bytes";
        readonly name: "name";
        readonly type: "bytes";
    }, {
        readonly indexed: false;
        readonly internalType: "uint16";
        readonly name: "resource";
        readonly type: "uint16";
    }];
    readonly name: "DNSRecordDeleted";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "DNSZoneCleared";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly internalType: "bytes";
        readonly name: "lastzonehash";
        readonly type: "bytes";
    }, {
        readonly indexed: false;
        readonly internalType: "bytes";
        readonly name: "zonehash";
        readonly type: "bytes";
    }];
    readonly name: "DNSZonehashChanged";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: true;
        readonly internalType: "bytes4";
        readonly name: "interfaceID";
        readonly type: "bytes4";
    }, {
        readonly indexed: false;
        readonly internalType: "address";
        readonly name: "implementer";
        readonly type: "address";
    }];
    readonly name: "InterfaceChanged";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly internalType: "string";
        readonly name: "name";
        readonly type: "string";
    }];
    readonly name: "NameChanged";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly internalType: "bytes32";
        readonly name: "x";
        readonly type: "bytes32";
    }, {
        readonly indexed: false;
        readonly internalType: "bytes32";
        readonly name: "y";
        readonly type: "bytes32";
    }];
    readonly name: "PubkeyChanged";
    readonly type: "event";
}, {
    readonly anonymous: false;
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly indexed: true;
        readonly internalType: "string";
        readonly name: "indexedKey";
        readonly type: "string";
    }, {
        readonly indexed: false;
        readonly internalType: "string";
        readonly name: "key";
        readonly type: "string";
    }];
    readonly name: "TextChanged";
    readonly type: "event";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly internalType: "uint256";
        readonly name: "contentTypes";
        readonly type: "uint256";
    }];
    readonly name: "ABI";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "";
        readonly type: "uint256";
    }, {
        readonly internalType: "bytes";
        readonly name: "";
        readonly type: "bytes";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "addr";
    readonly outputs: readonly [{
        readonly internalType: "address payable";
        readonly name: "";
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly internalType: "uint256";
        readonly name: "coinType";
        readonly type: "uint256";
    }];
    readonly name: "addr";
    readonly outputs: readonly [{
        readonly internalType: "bytes";
        readonly name: "";
        readonly type: "bytes";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "contenthash";
    readonly outputs: readonly [{
        readonly internalType: "bytes";
        readonly name: "";
        readonly type: "bytes";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly internalType: "bytes32";
        readonly name: "name";
        readonly type: "bytes32";
    }, {
        readonly internalType: "uint16";
        readonly name: "resource";
        readonly type: "uint16";
    }];
    readonly name: "dnsRecord";
    readonly outputs: readonly [{
        readonly internalType: "bytes";
        readonly name: "";
        readonly type: "bytes";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly internalType: "bytes32";
        readonly name: "name";
        readonly type: "bytes32";
    }];
    readonly name: "hasDNSRecords";
    readonly outputs: readonly [{
        readonly internalType: "bool";
        readonly name: "";
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly internalType: "bytes4";
        readonly name: "interfaceID";
        readonly type: "bytes4";
    }];
    readonly name: "interfaceImplementer";
    readonly outputs: readonly [{
        readonly internalType: "address";
        readonly name: "";
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "address";
        readonly name: "account";
        readonly type: "address";
    }, {
        readonly internalType: "address";
        readonly name: "operator";
        readonly type: "address";
    }];
    readonly name: "isApprovedForAll";
    readonly outputs: readonly [{
        readonly internalType: "bool";
        readonly name: "";
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "name";
    readonly outputs: readonly [{
        readonly internalType: "string";
        readonly name: "";
        readonly type: "string";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "pubkey";
    readonly outputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "x";
        readonly type: "bytes32";
    }, {
        readonly internalType: "bytes32";
        readonly name: "y";
        readonly type: "bytes32";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes4";
        readonly name: "interfaceID";
        readonly type: "bytes4";
    }];
    readonly name: "supportsInterface";
    readonly outputs: readonly [{
        readonly internalType: "bool";
        readonly name: "";
        readonly type: "bool";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly internalType: "string";
        readonly name: "key";
        readonly type: "string";
    }];
    readonly name: "text";
    readonly outputs: readonly [{
        readonly internalType: "string";
        readonly name: "";
        readonly type: "string";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }];
    readonly name: "zonehash";
    readonly outputs: readonly [{
        readonly internalType: "bytes";
        readonly name: "";
        readonly type: "bytes";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "bytes32";
        readonly name: "node";
        readonly type: "bytes32";
    }, {
        readonly internalType: "address";
        readonly name: "a";
        readonly type: "address";
    }];
    readonly name: "setAddr";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}];
//# sourceMappingURL=PublicResolver.d.ts.map