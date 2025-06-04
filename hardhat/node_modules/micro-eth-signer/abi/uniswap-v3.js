"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.UNISWAP_V3_ROUTER_CONTRACT = void 0;
const utils_1 = require("@noble/hashes/utils");
const utils_ts_1 = require("../utils.js");
const common_ts_1 = require("./common.js");
// prettier-ignore
const _ABI = [
    { inputs: [{ internalType: "address", name: "_factory", type: "address" }, { internalType: "address", name: "_WETH9", type: "address" }], stateMutability: "nonpayable", type: "constructor" }, { inputs: [], name: "WETH9", outputs: [{ internalType: "address", name: "", type: "address" }], stateMutability: "view", type: "function" }, { inputs: [{ components: [{ internalType: "bytes", name: "path", type: "bytes" }, { internalType: "address", name: "recipient", type: "address" }, { internalType: "uint256", name: "deadline", type: "uint256" }, { internalType: "uint256", name: "amountIn", type: "uint256" }, { internalType: "uint256", name: "amountOutMinimum", type: "uint256" }], internalType: "struct ISwapRouter.ExactInputParams", name: "params", type: "tuple" }], name: "exactInput", outputs: [{ internalType: "uint256", name: "amountOut", type: "uint256" }], stateMutability: "payable", type: "function" }, { inputs: [{ components: [{ internalType: "address", name: "tokenIn", type: "address" }, { internalType: "address", name: "tokenOut", type: "address" }, { internalType: "uint24", name: "fee", type: "uint24" }, { internalType: "address", name: "recipient", type: "address" }, { internalType: "uint256", name: "deadline", type: "uint256" }, { internalType: "uint256", name: "amountIn", type: "uint256" }, { internalType: "uint256", name: "amountOutMinimum", type: "uint256" }, { internalType: "uint160", name: "sqrtPriceLimitX96", type: "uint160" }], internalType: "struct ISwapRouter.ExactInputSingleParams", name: "params", type: "tuple" }], name: "exactInputSingle", outputs: [{ internalType: "uint256", name: "amountOut", type: "uint256" }], stateMutability: "payable", type: "function" }, { inputs: [{ components: [{ internalType: "bytes", name: "path", type: "bytes" }, { internalType: "address", name: "recipient", type: "address" }, { internalType: "uint256", name: "deadline", type: "uint256" }, { internalType: "uint256", name: "amountOut", type: "uint256" }, { internalType: "uint256", name: "amountInMaximum", type: "uint256" }], internalType: "struct ISwapRouter.ExactOutputParams", name: "params", type: "tuple" }], name: "exactOutput", outputs: [{ internalType: "uint256", name: "amountIn", type: "uint256" }], stateMutability: "payable", type: "function" }, { inputs: [{ components: [{ internalType: "address", name: "tokenIn", type: "address" }, { internalType: "address", name: "tokenOut", type: "address" }, { internalType: "uint24", name: "fee", type: "uint24" }, { internalType: "address", name: "recipient", type: "address" }, { internalType: "uint256", name: "deadline", type: "uint256" }, { internalType: "uint256", name: "amountOut", type: "uint256" }, { internalType: "uint256", name: "amountInMaximum", type: "uint256" }, { internalType: "uint160", name: "sqrtPriceLimitX96", type: "uint160" }], internalType: "struct ISwapRouter.ExactOutputSingleParams", name: "params", type: "tuple" }], name: "exactOutputSingle", outputs: [{ internalType: "uint256", name: "amountIn", type: "uint256" }], stateMutability: "payable", type: "function" }, { inputs: [], name: "factory", outputs: [{ internalType: "address", name: "", type: "address" }], stateMutability: "view", type: "function" }, { inputs: [{ internalType: "bytes[]", name: "data", type: "bytes[]" }], name: "multicall", outputs: [{ internalType: "bytes[]", name: "results", type: "bytes[]" }], stateMutability: "payable", type: "function" }, { inputs: [], name: "refundETH", outputs: [], stateMutability: "payable", type: "function" }, { inputs: [{ internalType: "address", name: "token", type: "address" }, { internalType: "uint256", name: "value", type: "uint256" }, { internalType: "uint256", name: "deadline", type: "uint256" }, { internalType: "uint8", name: "v", type: "uint8" }, { internalType: "bytes32", name: "r", type: "bytes32" }, { internalType: "bytes32", name: "s", type: "bytes32" }], name: "selfPermit", outputs: [], stateMutability: "payable", type: "function" }, { inputs: [{ internalType: "address", name: "token", type: "address" }, { internalType: "uint256", name: "nonce", type: "uint256" }, { internalType: "uint256", name: "expiry", type: "uint256" }, { internalType: "uint8", name: "v", type: "uint8" }, { internalType: "bytes32", name: "r", type: "bytes32" }, { internalType: "bytes32", name: "s", type: "bytes32" }], name: "selfPermitAllowed", outputs: [], stateMutability: "payable", type: "function" }, { inputs: [{ internalType: "address", name: "token", type: "address" }, { internalType: "uint256", name: "nonce", type: "uint256" }, { internalType: "uint256", name: "expiry", type: "uint256" }, { internalType: "uint8", name: "v", type: "uint8" }, { internalType: "bytes32", name: "r", type: "bytes32" }, { internalType: "bytes32", name: "s", type: "bytes32" }], name: "selfPermitAllowedIfNecessary", outputs: [], stateMutability: "payable", type: "function" }, { inputs: [{ internalType: "address", name: "token", type: "address" }, { internalType: "uint256", name: "value", type: "uint256" }, { internalType: "uint256", name: "deadline", type: "uint256" }, { internalType: "uint8", name: "v", type: "uint8" }, { internalType: "bytes32", name: "r", type: "bytes32" }, { internalType: "bytes32", name: "s", type: "bytes32" }], name: "selfPermitIfNecessary", outputs: [], stateMutability: "payable", type: "function" }, { inputs: [{ internalType: "address", name: "token", type: "address" }, { internalType: "uint256", name: "amountMinimum", type: "uint256" }, { internalType: "address", name: "recipient", type: "address" }], name: "sweepToken", outputs: [], stateMutability: "payable", type: "function" }, { inputs: [{ internalType: "address", name: "token", type: "address" }, { internalType: "uint256", name: "amountMinimum", type: "uint256" }, { internalType: "address", name: "recipient", type: "address" }, { internalType: "uint256", name: "feeBips", type: "uint256" }, { internalType: "address", name: "feeRecipient", type: "address" }], name: "sweepTokenWithFee", outputs: [], stateMutability: "payable", type: "function" }, { inputs: [{ internalType: "int256", name: "amount0Delta", type: "int256" }, { internalType: "int256", name: "amount1Delta", type: "int256" }, { internalType: "bytes", name: "_data", type: "bytes" }], name: "uniswapV3SwapCallback", outputs: [], stateMutability: "nonpayable", type: "function" }, { inputs: [{ internalType: "uint256", name: "amountMinimum", type: "uint256" }, { internalType: "address", name: "recipient", type: "address" }], name: "unwrapWETH9", outputs: [], stateMutability: "payable", type: "function" }, { inputs: [{ internalType: "uint256", name: "amountMinimum", type: "uint256" }, { internalType: "address", name: "recipient", type: "address" }, { internalType: "uint256", name: "feeBips", type: "uint256" }, { internalType: "address", name: "feeRecipient", type: "address" }], name: "unwrapWETH9WithFee", outputs: [], stateMutability: "payable", type: "function" }, { stateMutability: "payable", type: "receive" }
];
// Generic multicall hook, maybe move to common?
const ABI_MULTICALL = /* @__PURE__ */ (0, common_ts_1.addHook)(_ABI, 'multicall', (d, contract, info, opt) => {
    const decoded = info.value.map((i) => d.decode(contract, i, opt));
    info.name = `multicall(${decoded.map((i) => i.name).join(', ')})`;
    info.signature = `multicall(${decoded.map((i) => i.signature).join(', ')})`;
    info.value = decoded.map((i) => i.value);
    let hasHint = false;
    for (let i of decoded)
        if (i.hint)
            hasHint = true;
    if (hasHint)
        info.hint = decoded
            .filter((i) => i.hint)
            .map((i) => i.hint)
            .join(' ');
    return info;
});
const decodePath = (b) => [b.slice(0, 20), b.slice(-20)].map((i) => (0, utils_ts_1.add0x)((0, utils_1.bytesToHex)(i)));
function uniToken(contract, amount, opt) {
    if (!contract || !opt.contracts || !opt.contracts[contract])
        return;
    const info = opt.contracts[contract];
    if (!info.decimals || !info.symbol)
        return;
    return `${(0, utils_ts_1.createDecimal)(info.decimals).encode(amount)} ${info.symbol}`;
}
const uniTs = (ts) => `Expires at ${new Date(Number(ts) * 1000).toUTCString()}`;
const hints = {
    exactInputSingle(v, opt) {
        const [from, to] = [
            uniToken(v.tokenIn, v.amountIn, opt),
            uniToken(v.tokenOut, v.amountOutMinimum, opt),
        ];
        if (!from || !to)
            throw new Error('Not enough info');
        return `Swap exact ${from} for at least ${to}. ${uniTs(v.deadline)}`;
    },
    exactOutputSingle(v, opt) {
        const [from, to] = [
            uniToken(v.tokenIn, v.amountInMaximum, opt),
            uniToken(v.tokenOut, v.amountOut, opt),
        ];
        if (!from || !to)
            throw new Error('Not enough info');
        return `Swap up to ${from} for exact ${to}. ${uniTs(v.deadline)}`;
    },
    exactInput(v, opt) {
        const [tokenIn, tokenOut] = decodePath(v.path);
        const [from, to] = [
            uniToken(tokenIn, v.amountIn, opt),
            uniToken(tokenOut, v.amountOutMinimum, opt),
        ];
        if (!from || !to)
            throw new Error('Not enough info');
        return `Swap exact ${from} for at least ${to}. ${uniTs(v.deadline)}`;
    },
    exactOutput(v, opt) {
        const [tokenIn, tokenOut] = decodePath(v.path).reverse();
        const [from, to] = [
            uniToken(tokenIn, v.amountInMaximum, opt),
            uniToken(tokenOut, v.amountOut, opt),
        ];
        if (!from || !to)
            throw new Error('Not enough info');
        return `Swap up to ${from} for exact ${to}. ${uniTs(v.deadline)}`;
    },
};
const ABI = /* @__PURE__ */ (0, common_ts_1.addHints)(ABI_MULTICALL, hints);
exports.default = ABI;
exports.UNISWAP_V3_ROUTER_CONTRACT = '0xe592427a0aece92de3edee1f18e0157c05861564';
//# sourceMappingURL=uniswap-v3.js.map