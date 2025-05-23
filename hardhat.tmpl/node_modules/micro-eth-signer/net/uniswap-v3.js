"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Fee = void 0;
exports.txData = txData;
const utils_1 = require("@noble/hashes/utils");
const decoder_ts_1 = require("../abi/decoder.js");
const uniswap_v3_ts_1 = require("../abi/uniswap-v3.js");
const utils_ts_1 = require("../utils.js");
const uni = require("./uniswap-common.js");
const ADDRESS_ZERO = '0x0000000000000000000000000000000000000000';
const QUOTER_ADDRESS = '0xb27308f9F90D607463bb33eA1BeBb41C27CE5AB6';
const QUOTER_ABI = [
    {
        type: 'function',
        name: 'quoteExactInput',
        inputs: [
            { name: 'path', type: 'bytes' },
            { name: 'amountIn', type: 'uint256' },
        ],
        outputs: [{ name: 'amountOut', type: 'uint256' }],
    },
    {
        type: 'function',
        name: 'quoteExactInputSingle',
        inputs: [
            { name: 'tokenIn', type: 'address' },
            { name: 'tokenOut', type: 'address' },
            { name: 'fee', type: 'uint24' },
            { name: 'amountIn', type: 'uint256' },
            { name: 'sqrtPriceLimitX96', type: 'uint160' },
        ],
        outputs: [{ name: 'amountOut', type: 'uint256' }],
    },
    {
        type: 'function',
        name: 'quoteExactOutput',
        inputs: [
            { name: 'path', type: 'bytes' },
            { name: 'amountOut', type: 'uint256' },
        ],
        outputs: [{ name: 'amountIn', type: 'uint256' }],
    },
    {
        type: 'function',
        name: 'quoteExactOutputSingle',
        inputs: [
            { name: 'tokenIn', type: 'address' },
            { name: 'tokenOut', type: 'address' },
            { name: 'fee', type: 'uint24' },
            { name: 'amountOut', type: 'uint256' },
            { name: 'sqrtPriceLimitX96', type: 'uint160' },
        ],
        outputs: [{ name: 'amountIn', type: 'uint256' }],
    },
];
exports.Fee = {
    LOW: 500,
    MEDIUM: 3000,
    HIGH: 10000,
};
function basePaths(a, b, exactOutput = false) {
    let res = [];
    for (let fee in exports.Fee)
        res.push({ fee: exports.Fee[fee], p: [a, b] });
    const wA = uni.wrapContract(a);
    const wB = uni.wrapContract(b);
    const BASES = uni.COMMON_BASES.filter((c) => c && c.contract && c.contract !== wA && c.contract !== wB);
    const packFee = (n) => exports.Fee[n].toString(16).padStart(6, '0');
    for (let c of BASES) {
        for (let fee1 in exports.Fee) {
            for (let fee2 in exports.Fee) {
                let path = [wA, packFee(fee1), c.contract, packFee(fee2), wB].map((i) => utils_ts_1.ethHex.decode(i));
                if (exactOutput)
                    path = path.reverse();
                res.push({ path: (0, utils_1.concatBytes)(...path) });
            }
        }
    }
    return res;
}
async function bestPath(net, a, b, amountIn, amountOut) {
    if ((amountIn && amountOut) || (!amountIn && !amountOut))
        throw new Error('uniswapV3.bestPath: provide only one amount');
    const quoter = (0, decoder_ts_1.createContract)(QUOTER_ABI, net, QUOTER_ADDRESS);
    let paths = basePaths(a, b, !!amountOut);
    for (let i of paths) {
        if (!i.path && !i.fee)
            continue;
        const opt = { ...i, tokenIn: a, tokenOut: b, amountIn, amountOut, sqrtPriceLimitX96: 0 };
        const method = 'quoteExact' + (amountIn ? 'Input' : 'Output') + (i.path ? '' : 'Single');
        // TODO: remove any
        i[amountIn ? 'amountOut' : 'amountIn'] = quoter[method].call(opt);
    }
    paths = (await uni.awaitDeep(paths, true));
    paths = paths.filter((i) => i.amountIn || i.amountOut);
    paths.sort((a, b) => Number(amountIn ? b.amountOut - a.amountOut : a.amountIn - b.amountIn));
    if (!paths.length)
        throw new Error('uniswap: cannot find path');
    return paths[0];
}
const ROUTER_CONTRACT = (0, decoder_ts_1.createContract)(uniswap_v3_ts_1.default, undefined, uniswap_v3_ts_1.UNISWAP_V3_ROUTER_CONTRACT);
function txData(to, input, output, route, amountIn, amountOut, opt = uni.DEFAULT_SWAP_OPT) {
    opt = { ...uni.DEFAULT_SWAP_OPT, ...opt };
    const err = 'Uniswap v3: ';
    if (!uni.isValidUniAddr(input))
        throw new Error(err + 'invalid input address');
    if (!uni.isValidUniAddr(output))
        throw new Error(err + 'invalid output address');
    if (!uni.isValidEthAddr(to))
        throw new Error(err + 'invalid to address');
    if (opt.fee && !uni.isValidUniAddr(opt.fee.to))
        throw new Error(err + 'invalid fee recepient addresss');
    if (input === 'eth' && output === 'eth')
        throw new Error(err + 'both input and output cannot be eth');
    if ((amountIn && amountOut) || (!amountIn && !amountOut))
        throw new Error(err + 'specify either amountIn or amountOut, but not both');
    if ((amountIn && !route.amountOut) ||
        (amountOut && !route.amountIn) ||
        (!route.fee && !route.path))
        throw new Error(err + 'invalid route');
    if (route.path && opt.sqrtPriceLimitX96)
        throw new Error(err + 'sqrtPriceLimitX96 on multi-hop trade');
    const deadline = opt.deadline || Math.floor(Date.now() / 1000);
    // flags for whether funds should be send first to the router
    const routerMustCustody = output === 'eth' || !!opt.fee;
    // TODO: remove "as bigint"
    let args = {
        ...route,
        tokenIn: uni.wrapContract(input),
        tokenOut: uni.wrapContract(output),
        recipient: routerMustCustody ? ADDRESS_ZERO : to,
        deadline,
        amountIn: (amountIn || route.amountIn),
        amountOut: (amountOut || route.amountOut),
        sqrtPriceLimitX96: opt.sqrtPriceLimitX96 || BigInt(0),
        amountInMaximum: undefined,
        amountOutMinimum: undefined,
    };
    args.amountInMaximum = uni.addPercent(args.amountIn, opt.slippagePercent);
    args.amountOutMinimum = uni.addPercent(args.amountOut, -opt.slippagePercent);
    const method = ('exact' + (amountIn ? 'Input' : 'Output') + (!args.path ? 'Single' : ''));
    // TODO: remove unknown
    const calldatas = [ROUTER_CONTRACT[method].encodeInput(args)];
    if (input === 'eth' && amountOut)
        calldatas.push(ROUTER_CONTRACT['refundETH'].encodeInput());
    // unwrap
    if (routerMustCustody) {
        calldatas.push(ROUTER_CONTRACT[(output === 'eth' ? 'unwrapWETH9' : 'sweepToken') + (opt.fee ? 'WithFee' : '')].encodeInput({
            token: uni.wrapContract(output),
            amountMinimum: args.amountOutMinimum,
            recipient: to,
            feeBips: opt.fee && opt.fee.fee * 10000,
            feeRecipient: opt.fee && opt.fee.to,
        }));
    }
    const data = calldatas.length === 1 ? calldatas[0] : ROUTER_CONTRACT['multicall'].encodeInput(calldatas);
    const value = input === 'eth' ? (amountIn ? amountIn : args.amountInMaximum) : BigInt(0);
    const allowance = input !== 'eth'
        ? { token: input, amount: amountIn ? amountIn : args.amountInMaximum }
        : undefined;
    return { to: uniswap_v3_ts_1.UNISWAP_V3_ROUTER_CONTRACT, value, data, allowance };
}
// Here goes Exchange API. Everything above is SDK.
class UniswapV3 extends uni.UniswapAbstract {
    constructor() {
        super(...arguments);
        this.name = 'Uniswap V3';
        this.contract = uniswap_v3_ts_1.UNISWAP_V3_ROUTER_CONTRACT;
    }
    bestPath(fromCoin, toCoin, inputAmount) {
        return bestPath(this.net, fromCoin, toCoin, inputAmount);
    }
    txData(toAddress, fromCoin, toCoin, path, inputAmount, outputAmount, opt = uni.DEFAULT_SWAP_OPT) {
        return txData(toAddress, fromCoin, toCoin, path, inputAmount, outputAmount, {
            ...uni.DEFAULT_SWAP_OPT,
            ...opt,
        });
    }
}
exports.default = UniswapV3;
//# sourceMappingURL=uniswap-v3.js.map