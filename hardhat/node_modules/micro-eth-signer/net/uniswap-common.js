"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.UniswapAbstract = exports.WETH = exports.COMMON_BASES = exports.DEFAULT_SWAP_OPT = void 0;
exports.addPercent = addPercent;
exports.isPromise = isPromise;
exports.awaitDeep = awaitDeep;
exports.wrapContract = wrapContract;
exports.sortTokens = sortTokens;
exports.isValidEthAddr = isValidEthAddr;
exports.isValidUniAddr = isValidUniAddr;
const index_ts_1 = require("../abi/index.js");
const index_ts_2 = require("../index.js");
const utils_ts_1 = require("../utils.js");
exports.DEFAULT_SWAP_OPT = { slippagePercent: 0.5, ttl: 30 * 60 };
function addPercent(n, _perc) {
    const perc = BigInt((_perc * 10000) | 0);
    const p100 = BigInt(100) * BigInt(10000);
    return ((p100 + perc) * n) / p100;
}
function isPromise(o) {
    if (!o || !['object', 'function'].includes(typeof o))
        return false;
    return typeof o.then === 'function';
}
async function awaitDeep(o, ignore_errors) {
    let promises = [];
    const traverse = (o) => {
        if (Array.isArray(o))
            return o.map((i) => traverse(i));
        if ((0, utils_ts_1.isBytes)(o))
            return o;
        if (isPromise(o))
            return { awaitDeep: promises.push(o) };
        if (typeof o === 'object') {
            let ret = {};
            for (let k in o)
                ret[k] = traverse(o[k]);
            return ret;
        }
        return o;
    };
    let out = traverse(o);
    let values;
    if (!ignore_errors)
        values = await Promise.all(promises);
    else {
        values = (await Promise.allSettled(promises)).map((i) => i.status === 'fulfilled' ? i.value : undefined);
    }
    const trBack = (o) => {
        if (Array.isArray(o))
            return o.map((i) => trBack(i));
        if ((0, utils_ts_1.isBytes)(o))
            return o;
        if (typeof o === 'object') {
            if (typeof o === 'object' && o.awaitDeep)
                return values[o.awaitDeep - 1];
            let ret = {};
            for (let k in o)
                ret[k] = trBack(o[k]);
            return ret;
        }
        return o;
    };
    return trBack(out);
}
exports.COMMON_BASES = [
    'WETH',
    'DAI',
    'USDC',
    'USDT',
    'COMP',
    'MKR',
    'WBTC',
    'AMPL',
]
    .map((i) => (0, index_ts_1.tokenFromSymbol)(i))
    .filter((i) => !!i);
exports.WETH = (0, index_ts_1.tokenFromSymbol)('WETH').contract;
if (!exports.WETH)
    throw new Error('WETH is undefined!');
function wrapContract(contract) {
    contract = contract.toLowerCase();
    return contract === 'eth' ? exports.WETH : contract;
}
function sortTokens(a, b) {
    a = wrapContract(a);
    b = wrapContract(b);
    if (a === b)
        throw new Error('uniswap.sortTokens: same token!');
    return a < b ? [a, b] : [b, a];
}
function isValidEthAddr(address) {
    return index_ts_2.addr.isValid(address);
}
function isValidUniAddr(address) {
    return address === 'eth' || isValidEthAddr(address);
}
function getToken(token) {
    if (typeof token === 'string' && token.toLowerCase() === 'eth')
        return { symbol: 'ETH', decimals: 18, contract: 'eth' };
    return token;
}
class UniswapAbstract {
    constructor(net) {
        this.net = net;
    }
    // private async coinInfo(netName: string) {
    //   if (!validateAddr(netName)) return;
    //   if (netName === 'eth') return { symbol: 'ETH', decimals: 18 };
    //   //return await this.mgr.tokenInfo('eth', netName);
    // }
    async swap(fromCoin, toCoin, amount, opt = exports.DEFAULT_SWAP_OPT) {
        const fromInfo = getToken(fromCoin);
        const toInfo = getToken(toCoin);
        if (!fromInfo || !toInfo)
            return;
        const fromContract = fromInfo.contract.toLowerCase();
        const toContract = toInfo.contract.toLowerCase();
        if (!fromContract || !toContract)
            return;
        const fromDecimal = (0, utils_ts_1.createDecimal)(fromInfo.decimals);
        const toDecimal = (0, utils_ts_1.createDecimal)(toInfo.decimals);
        const inputAmount = fromDecimal.decode(amount);
        try {
            const path = await this.bestPath(fromContract, toContract, inputAmount);
            const expectedAmount = toDecimal.encode(path.amountOut);
            return {
                name: this.name,
                expectedAmount,
                tx: async (_fromAddress, toAddress) => {
                    const txUni = this.txData(toAddress, fromContract, toContract, path, inputAmount, undefined, opt);
                    return {
                        amount: utils_ts_1.weieth.encode(txUni.value),
                        address: txUni.to,
                        expectedAmount,
                        data: utils_ts_1.ethHex.encode(txUni.data),
                        allowance: txUni.allowance && {
                            token: txUni.allowance.token,
                            contract: this.contract,
                            amount: fromDecimal.encode(txUni.allowance.amount),
                        },
                    };
                },
            };
        }
        catch (e) {
            // @ts-ignore
            console.log('E', e);
            return;
        }
    }
}
exports.UniswapAbstract = UniswapAbstract;
//# sourceMappingURL=uniswap-common.js.map