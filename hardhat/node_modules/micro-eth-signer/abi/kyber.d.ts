declare const ABI: readonly [{
    readonly type: "function";
    readonly name: "getExpectedRate";
    readonly inputs: readonly [{
        readonly name: "src";
        readonly type: "address";
    }, {
        readonly name: "dest";
        readonly type: "address";
    }, {
        readonly name: "srcQty";
        readonly type: "uint256";
    }];
    readonly outputs: readonly [{
        readonly name: "expectedRate";
        readonly type: "uint256";
    }, {
        readonly name: "worstRate";
        readonly type: "uint256";
    }];
}, {
    readonly type: "function";
    readonly name: "getExpectedRateAfterFee";
    readonly inputs: readonly [{
        readonly name: "src";
        readonly type: "address";
    }, {
        readonly name: "dest";
        readonly type: "address";
    }, {
        readonly name: "srcQty";
        readonly type: "uint256";
    }, {
        readonly name: "platformFeeBps";
        readonly type: "uint256";
    }, {
        readonly name: "hint";
        readonly type: "bytes";
    }];
    readonly outputs: readonly [{
        readonly name: "expectedRate";
        readonly type: "uint256";
    }];
}, {
    readonly type: "function";
    readonly name: "trade";
    readonly inputs: readonly [{
        readonly name: "src";
        readonly type: "address";
    }, {
        readonly name: "srcAmount";
        readonly type: "uint256";
    }, {
        readonly name: "dest";
        readonly type: "address";
    }, {
        readonly name: "destAddress";
        readonly type: "address";
    }, {
        readonly name: "maxDestAmount";
        readonly type: "uint256";
    }, {
        readonly name: "minConversionRate";
        readonly type: "uint256";
    }, {
        readonly name: "platformWallet";
        readonly type: "address";
    }];
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
}, {
    readonly type: "function";
    readonly name: "tradeWithHint";
    readonly inputs: readonly [{
        readonly name: "src";
        readonly type: "address";
    }, {
        readonly name: "srcAmount";
        readonly type: "uint256";
    }, {
        readonly name: "dest";
        readonly type: "address";
    }, {
        readonly name: "destAddress";
        readonly type: "address";
    }, {
        readonly name: "maxDestAmount";
        readonly type: "uint256";
    }, {
        readonly name: "minConversionRate";
        readonly type: "uint256";
    }, {
        readonly name: "walletId";
        readonly type: "address";
    }, {
        readonly name: "hint";
        readonly type: "bytes";
    }];
    readonly outputs: readonly [{
        readonly type: "uint256";
    }];
}, {
    readonly type: "function";
    readonly name: "tradeWithHintAndFee";
    readonly inputs: readonly [{
        readonly name: "src";
        readonly type: "address";
    }, {
        readonly name: "srcAmount";
        readonly type: "uint256";
    }, {
        readonly name: "dest";
        readonly type: "address";
    }, {
        readonly name: "destAddress";
        readonly type: "address";
    }, {
        readonly name: "maxDestAmount";
        readonly type: "uint256";
    }, {
        readonly name: "minConversionRate";
        readonly type: "uint256";
    }, {
        readonly name: "platformWallet";
        readonly type: "address";
    }, {
        readonly name: "platformFeeBps";
        readonly type: "uint256";
    }, {
        readonly name: "hint";
        readonly type: "bytes";
    }];
    readonly outputs: readonly [{
        readonly name: "destAmount";
        readonly type: "uint256";
    }];
}];
export default ABI;
export declare const KYBER_NETWORK_PROXY_CONTRACT = "0x9aab3f75489902f3a48495025729a0af77d4b11e";
//# sourceMappingURL=kyber.d.ts.map