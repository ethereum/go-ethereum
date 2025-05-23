declare const ABI: readonly [{
    readonly name: "ApprovalForAll";
    readonly type: "event";
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "account";
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
}, {
    readonly name: "TransferBatch";
    readonly type: "event";
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "operator";
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
        readonly name: "ids";
        readonly type: "uint256[]";
    }, {
        readonly indexed: false;
        readonly name: "values";
        readonly type: "uint256[]";
    }];
}, {
    readonly name: "TransferSingle";
    readonly type: "event";
    readonly inputs: readonly [{
        readonly indexed: true;
        readonly name: "operator";
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
        readonly name: "id";
        readonly type: "uint256";
    }, {
        readonly indexed: false;
        readonly name: "value";
        readonly type: "uint256";
    }];
}, {
    readonly name: "URI";
    readonly type: "event";
    readonly inputs: readonly [{
        readonly indexed: false;
        readonly name: "value";
        readonly type: "string";
    }, {
        readonly indexed: true;
        readonly name: "id";
        readonly type: "uint256";
    }];
}, {
    readonly name: "balanceOf";
    readonly type: "function";
    readonly inputs: readonly [{
        readonly name: "account";
        readonly type: "address";
    }, {
        readonly name: "id";
        readonly type: "uint256";
    }];
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
}, {
    readonly name: "balanceOfBatch";
    readonly type: "function";
    readonly inputs: readonly [{
        readonly name: "accounts";
        readonly type: "address[]";
    }, {
        readonly name: "ids";
        readonly type: "uint256[]";
    }];
    readonly outputs: readonly [{
        readonly type: "uint256[]";
    }];
}, {
    readonly name: "isApprovedForAll";
    readonly type: "function";
    readonly inputs: readonly [{
        readonly name: "account";
        readonly type: "address";
    }, {
        readonly name: "operator";
        readonly type: "address";
    }];
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
}, {
    readonly name: "safeBatchTransferFrom";
    readonly type: "function";
    readonly inputs: readonly [{
        readonly name: "from";
        readonly type: "address";
    }, {
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly name: "ids";
        readonly type: "uint256[]";
    }, {
        readonly name: "amounts";
        readonly type: "uint256[]";
    }, {
        readonly name: "data";
        readonly type: "bytes";
    }];
    readonly outputs: readonly [];
}, {
    readonly name: "safeTransferFrom";
    readonly type: "function";
    readonly inputs: readonly [{
        readonly name: "from";
        readonly type: "address";
    }, {
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly name: "id";
        readonly type: "uint256";
    }, {
        readonly name: "amount";
        readonly type: "uint256";
    }, {
        readonly name: "data";
        readonly type: "bytes";
    }];
    readonly outputs: readonly [];
}, {
    readonly name: "setApprovalForAll";
    readonly type: "function";
    readonly inputs: readonly [{
        readonly name: "operator";
        readonly type: "address";
    }, {
        readonly name: "approved";
        readonly type: "bool";
    }];
    readonly outputs: readonly [];
}, {
    readonly name: "supportsInterface";
    readonly type: "function";
    readonly inputs: readonly [{
        readonly name: "interfaceId";
        readonly type: "bytes4";
    }];
    readonly outputs: readonly [{
        readonly type: "bool";
    }];
}, {
    readonly name: "uri";
    readonly type: "function";
    readonly inputs: readonly [{
        readonly name: "id";
        readonly type: "uint256";
    }];
    readonly outputs: readonly [{
        readonly type: "string";
    }];
}];
export default ABI;
//# sourceMappingURL=erc1155.d.ts.map