import { keccak_256 } from '@noble/hashes/sha3';
import { concatBytes, hexToBytes } from '@noble/hashes/utils';
import { createContract } from "../abi/decoder.js";
import { default as UNISWAP_V2_ROUTER, UNISWAP_V2_ROUTER_CONTRACT } from "../abi/uniswap-v2.js";
import { ethHex } from "../utils.js";
import * as uni from "./uniswap-common.js";
const FACTORY_ADDRESS = '0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f';
const INIT_CODE_HASH = hexToBytes('96e8ac4277198ff8b6f785478aa9a39f403cb768dd02cbee326c3e7da348845f');
const PAIR_CONTRACT = [
    {
        type: 'function',
        name: 'getReserves',
        outputs: [
            { name: 'reserve0', type: 'uint112' },
            { name: 'reserve1', type: 'uint112' },
            { name: 'blockTimestampLast', type: 'uint32' },
        ],
    },
];
export function create2(from, salt, initCodeHash) {
    const cat = concatBytes(new Uint8Array([255]), from, salt, initCodeHash);
    return ethHex.encode(keccak_256(cat).slice(12));
}
export function pairAddress(a, b, factory = FACTORY_ADDRESS) {
    // This is completely broken: '0x11' '0x11' will return '0x1111'. But this is how it works in sdk.
    const data = concatBytes(...uni.sortTokens(a, b).map((i) => ethHex.decode(i)));
    return create2(ethHex.decode(factory), keccak_256(data), INIT_CODE_HASH);
}
async function reserves(net, a, b) {
    a = uni.wrapContract(a);
    b = uni.wrapContract(b);
    const contract = createContract(PAIR_CONTRACT, net, pairAddress(a, b));
    const res = await contract.getReserves.call();
    return a < b ? [res.reserve0, res.reserve1] : [res.reserve1, res.reserve0];
}
// amountIn set: returns amountOut, how many tokenB user gets for amountIn of tokenA
// amountOut set: returns amountIn, how many tokenA user should send to get exact
// amountOut of tokenB
export function amount(reserveIn, reserveOut, amountIn, amountOut) {
    if (amountIn && amountOut)
        throw new Error('uniswap.amount: provide only one amount');
    if (!reserveIn || !reserveOut || (amountOut && amountOut >= reserveOut))
        throw new Error('Uniswap: Insufficient reserves');
    if (amountIn) {
        const amountInWithFee = amountIn * BigInt(997);
        const amountOut = (amountInWithFee * reserveOut) / (reserveIn * BigInt(1000) + amountInWithFee);
        if (amountOut === BigInt(0) || amountOut >= reserveOut)
            throw new Error('Uniswap: Insufficient reserves');
        return amountOut;
    }
    else if (amountOut)
        return ((reserveIn * amountOut * BigInt(1000)) / ((reserveOut - amountOut) * BigInt(997)) + BigInt(1));
    else
        throw new Error('uniswap.amount: provide only one amount');
}
async function bestPath(net, tokenA, tokenB, amountIn, amountOut) {
    if ((amountIn && amountOut) || (!amountIn && !amountOut))
        throw new Error('uniswap.bestPath: provide only one amount');
    const wA = uni.wrapContract(tokenA);
    const wB = uni.wrapContract(tokenB);
    let resP = [];
    // Direct pair
    resP.push((async () => {
        const pairAmount = amount(...(await reserves(net, tokenA, tokenB)), amountIn, amountOut);
        return {
            path: [wA, wB],
            amountIn: amountIn ? amountIn : pairAmount,
            amountOut: amountOut ? amountOut : pairAmount,
        };
    })());
    const BASES = uni.COMMON_BASES.filter((c) => c && c.contract && c.contract !== wA && c.contract !== wB);
    for (let c of BASES) {
        resP.push((async () => {
            const [rAC, rCB] = await Promise.all([
                reserves(net, wA, c.contract),
                reserves(net, c.contract, wB),
            ]);
            const path = [wA, c.contract, wB];
            if (amountIn)
                return { path, amountIn, amountOut: amount(...rCB, amount(...rAC, amountIn)) };
            else if (amountOut) {
                return {
                    path,
                    amountOut,
                    amountIn: amount(...rAC, undefined, amount(...rCB, undefined, amountOut)),
                };
            }
            else
                throw new Error('Impossible invariant');
        })());
    }
    let res = (await uni.awaitDeep(resP, true)).filter((i) => !!i);
    // biggest output or smallest input
    res.sort((a, b) => Number(amountIn ? b.amountOut - a.amountOut : a.amountIn - b.amountIn));
    if (!res.length)
        throw new Error('uniswap: cannot find path');
    return res[0];
}
const ROUTER_CONTRACT = createContract(UNISWAP_V2_ROUTER, undefined, UNISWAP_V2_ROUTER_CONTRACT);
const TX_DEFAULT_OPT = {
    ...uni.DEFAULT_SWAP_OPT,
    feeOnTransfer: false, // have no idea what it is
};
export function txData(to, input, output, path, amountIn, amountOut, opt = TX_DEFAULT_OPT) {
    opt = { ...TX_DEFAULT_OPT, ...opt };
    if (!uni.isValidUniAddr(input) || !uni.isValidUniAddr(output) || !uni.isValidEthAddr(to))
        throw new Error('Invalid address');
    if (input === 'eth' && output === 'eth')
        throw new Error('Both input and output is ETH!');
    if (input === 'eth' && path.path[0] !== uni.WETH)
        throw new Error('Input is ETH but path starts with different contract');
    if (output === 'eth' && path.path[path.path.length - 1] !== uni.WETH)
        throw new Error('Output is ETH but path ends with different contract');
    if ((amountIn && amountOut) || (!amountIn && !amountOut))
        throw new Error('uniswap.txData: provide only one amount');
    if (amountOut && opt.feeOnTransfer)
        throw new Error('Exact output + feeOnTransfer is impossible');
    const method = ('swap' +
        (amountIn ? 'Exact' : '') +
        (input === 'eth' ? 'ETH' : 'Tokens') +
        'For' +
        (amountOut ? 'Exact' : '') +
        (output === 'eth' ? 'ETH' : 'Tokens') +
        (opt.feeOnTransfer ? 'SupportingFeeOnTransferTokens' : ''));
    if (!(method in ROUTER_CONTRACT))
        throw new Error('Invalid method');
    const deadline = opt.deadline ? opt.deadline : Math.floor(Date.now() / 1000) + opt.ttl;
    const amountInMax = uni.addPercent(path.amountIn, opt.slippagePercent);
    const amountOutMin = uni.addPercent(path.amountOut, -opt.slippagePercent);
    // TODO: remove any
    const data = ROUTER_CONTRACT[method].encodeInput({
        amountInMax,
        amountOutMin,
        amountIn,
        amountOut,
        to,
        deadline,
        path: path.path,
    });
    const amount = amountIn ? amountIn : amountInMax;
    const value = input === 'eth' ? amount : BigInt(0);
    const allowance = input === 'eth' ? undefined : { token: input, amount };
    return { to: UNISWAP_V2_ROUTER_CONTRACT, value, data, allowance };
}
// Here goes Exchange API. Everything above is SDK. Supports almost everything from official sdk except liquidity stuff.
export default class UniswapV2 extends uni.UniswapAbstract {
    constructor() {
        super(...arguments);
        this.name = 'Uniswap V2';
        this.contract = UNISWAP_V2_ROUTER_CONTRACT;
    }
    bestPath(fromCoin, toCoin, inputAmount) {
        return bestPath(this.net, fromCoin, toCoin, inputAmount);
    }
    txData(toAddress, fromCoin, toCoin, path, inputAmount, outputAmount, opt = uni.DEFAULT_SWAP_OPT) {
        return txData(toAddress, fromCoin, toCoin, path, inputAmount, outputAmount, {
            ...TX_DEFAULT_OPT,
            ...opt,
        });
    }
}
//# sourceMappingURL=uniswap-v2.js.map