declare const ABI: readonly [{
    readonly inputs: readonly [{
        readonly internalType: "address";
        readonly name: "_factory";
        readonly type: "address";
    }, {
        readonly internalType: "address";
        readonly name: "_WETH";
        readonly type: "address";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "constructor";
}, {
    readonly inputs: readonly [];
    readonly name: "WETH";
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
        readonly name: "tokenA";
        readonly type: "address";
    }, {
        readonly internalType: "address";
        readonly name: "tokenB";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountADesired";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountBDesired";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountAMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountBMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "addLiquidity";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountA";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountB";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "liquidity";
        readonly type: "uint256";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "address";
        readonly name: "token";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountTokenDesired";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountTokenMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountETHMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "addLiquidityETH";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountToken";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountETH";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "liquidity";
        readonly type: "uint256";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [];
    readonly name: "factory";
    readonly outputs: readonly [{
        readonly internalType: "address";
        readonly name: "";
        readonly type: "address";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountOut";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "reserveIn";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "reserveOut";
        readonly type: "uint256";
    }];
    readonly name: "getAmountIn";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountIn";
        readonly type: "uint256";
    }];
    readonly stateMutability: "pure";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountIn";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "reserveIn";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "reserveOut";
        readonly type: "uint256";
    }];
    readonly name: "getAmountOut";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountOut";
        readonly type: "uint256";
    }];
    readonly stateMutability: "pure";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountOut";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }];
    readonly name: "getAmountsIn";
    readonly outputs: readonly [{
        readonly internalType: "uint256[]";
        readonly name: "amounts";
        readonly type: "uint256[]";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountIn";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }];
    readonly name: "getAmountsOut";
    readonly outputs: readonly [{
        readonly internalType: "uint256[]";
        readonly name: "amounts";
        readonly type: "uint256[]";
    }];
    readonly stateMutability: "view";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountA";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "reserveA";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "reserveB";
        readonly type: "uint256";
    }];
    readonly name: "quote";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountB";
        readonly type: "uint256";
    }];
    readonly stateMutability: "pure";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "address";
        readonly name: "tokenA";
        readonly type: "address";
    }, {
        readonly internalType: "address";
        readonly name: "tokenB";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "liquidity";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountAMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountBMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "removeLiquidity";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountA";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountB";
        readonly type: "uint256";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "address";
        readonly name: "token";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "liquidity";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountTokenMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountETHMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "removeLiquidityETH";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountToken";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountETH";
        readonly type: "uint256";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "address";
        readonly name: "token";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "liquidity";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountTokenMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountETHMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "removeLiquidityETHSupportingFeeOnTransferTokens";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountETH";
        readonly type: "uint256";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "address";
        readonly name: "token";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "liquidity";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountTokenMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountETHMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }, {
        readonly internalType: "bool";
        readonly name: "approveMax";
        readonly type: "bool";
    }, {
        readonly internalType: "uint8";
        readonly name: "v";
        readonly type: "uint8";
    }, {
        readonly internalType: "bytes32";
        readonly name: "r";
        readonly type: "bytes32";
    }, {
        readonly internalType: "bytes32";
        readonly name: "s";
        readonly type: "bytes32";
    }];
    readonly name: "removeLiquidityETHWithPermit";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountToken";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountETH";
        readonly type: "uint256";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "address";
        readonly name: "token";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "liquidity";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountTokenMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountETHMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }, {
        readonly internalType: "bool";
        readonly name: "approveMax";
        readonly type: "bool";
    }, {
        readonly internalType: "uint8";
        readonly name: "v";
        readonly type: "uint8";
    }, {
        readonly internalType: "bytes32";
        readonly name: "r";
        readonly type: "bytes32";
    }, {
        readonly internalType: "bytes32";
        readonly name: "s";
        readonly type: "bytes32";
    }];
    readonly name: "removeLiquidityETHWithPermitSupportingFeeOnTransferTokens";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountETH";
        readonly type: "uint256";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "address";
        readonly name: "tokenA";
        readonly type: "address";
    }, {
        readonly internalType: "address";
        readonly name: "tokenB";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "liquidity";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountAMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountBMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }, {
        readonly internalType: "bool";
        readonly name: "approveMax";
        readonly type: "bool";
    }, {
        readonly internalType: "uint8";
        readonly name: "v";
        readonly type: "uint8";
    }, {
        readonly internalType: "bytes32";
        readonly name: "r";
        readonly type: "bytes32";
    }, {
        readonly internalType: "bytes32";
        readonly name: "s";
        readonly type: "bytes32";
    }];
    readonly name: "removeLiquidityWithPermit";
    readonly outputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountA";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountB";
        readonly type: "uint256";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountOut";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "swapETHForExactTokens";
    readonly outputs: readonly [{
        readonly internalType: "uint256[]";
        readonly name: "amounts";
        readonly type: "uint256[]";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountOutMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "swapExactETHForTokens";
    readonly outputs: readonly [{
        readonly internalType: "uint256[]";
        readonly name: "amounts";
        readonly type: "uint256[]";
    }];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountOutMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "swapExactETHForTokensSupportingFeeOnTransferTokens";
    readonly outputs: readonly [];
    readonly stateMutability: "payable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountIn";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountOutMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "swapExactTokensForETH";
    readonly outputs: readonly [{
        readonly internalType: "uint256[]";
        readonly name: "amounts";
        readonly type: "uint256[]";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountIn";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountOutMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "swapExactTokensForETHSupportingFeeOnTransferTokens";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountIn";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountOutMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "swapExactTokensForTokens";
    readonly outputs: readonly [{
        readonly internalType: "uint256[]";
        readonly name: "amounts";
        readonly type: "uint256[]";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountIn";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountOutMin";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "swapExactTokensForTokensSupportingFeeOnTransferTokens";
    readonly outputs: readonly [];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountOut";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountInMax";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "swapTokensForExactETH";
    readonly outputs: readonly [{
        readonly internalType: "uint256[]";
        readonly name: "amounts";
        readonly type: "uint256[]";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly inputs: readonly [{
        readonly internalType: "uint256";
        readonly name: "amountOut";
        readonly type: "uint256";
    }, {
        readonly internalType: "uint256";
        readonly name: "amountInMax";
        readonly type: "uint256";
    }, {
        readonly internalType: "address[]";
        readonly name: "path";
        readonly type: "address[]";
    }, {
        readonly internalType: "address";
        readonly name: "to";
        readonly type: "address";
    }, {
        readonly internalType: "uint256";
        readonly name: "deadline";
        readonly type: "uint256";
    }];
    readonly name: "swapTokensForExactTokens";
    readonly outputs: readonly [{
        readonly internalType: "uint256[]";
        readonly name: "amounts";
        readonly type: "uint256[]";
    }];
    readonly stateMutability: "nonpayable";
    readonly type: "function";
}, {
    readonly stateMutability: "payable";
    readonly type: "receive";
}];
export default ABI;
export declare const UNISWAP_V2_ROUTER_CONTRACT = "0x7a250d5630b4cf539739df2c5dacb4c659f2488d";
//# sourceMappingURL=uniswap-v2.d.ts.map