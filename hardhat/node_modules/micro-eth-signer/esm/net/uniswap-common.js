import { tokenFromSymbol } from "../abi/index.js";
import { addr } from "../index.js";
import { createDecimal, ethHex, isBytes, weieth } from "../utils.js";
export const DEFAULT_SWAP_OPT = { slippagePercent: 0.5, ttl: 30 * 60 };
export function addPercent(n, _perc) {
    const perc = BigInt((_perc * 10000) | 0);
    const p100 = BigInt(100) * BigInt(10000);
    return ((p100 + perc) * n) / p100;
}
export function isPromise(o) {
    if (!o || !['object', 'function'].includes(typeof o))
        return false;
    return typeof o.then === 'function';
}
export async function awaitDeep(o, ignore_errors) {
    let promises = [];
    const traverse = (o) => {
        if (Array.isArray(o))
            return o.map((i) => traverse(i));
        if (isBytes(o))
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
        if (isBytes(o))
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
export const COMMON_BASES = [
    'WETH',
    'DAI',
    'USDC',
    'USDT',
    'COMP',
    'MKR',
    'WBTC',
    'AMPL',
]
    .map((i) => tokenFromSymbol(i))
    .filter((i) => !!i);
export const WETH = tokenFromSymbol('WETH').contract;
if (!WETH)
    throw new Error('WETH is undefined!');
export function wrapContract(contract) {
    contract = contract.toLowerCase();
    return contract === 'eth' ? WETH : contract;
}
export function sortTokens(a, b) {
    a = wrapContract(a);
    b = wrapContract(b);
    if (a === b)
        throw new Error('uniswap.sortTokens: same token!');
    return a < b ? [a, b] : [b, a];
}
export function isValidEthAddr(address) {
    return addr.isValid(address);
}
export function isValidUniAddr(address) {
    return address === 'eth' || isValidEthAddr(address);
}
function getToken(token) {
    if (typeof token === 'string' && token.toLowerCase() === 'eth')
        return { symbol: 'ETH', decimals: 18, contract: 'eth' };
    return token;
}
export class UniswapAbstract {
    constructor(net) {
        this.net = net;
    }
    // private async coinInfo(netName: string) {
    //   if (!validateAddr(netName)) return;
    //   if (netName === 'eth') return { symbol: 'ETH', decimals: 18 };
    //   //return await this.mgr.tokenInfo('eth', netName);
    // }
    async swap(fromCoin, toCoin, amount, opt = DEFAULT_SWAP_OPT) {
        const fromInfo = getToken(fromCoin);
        const toInfo = getToken(toCoin);
        if (!fromInfo || !toInfo)
            return;
        const fromContract = fromInfo.contract.toLowerCase();
        const toContract = toInfo.contract.toLowerCase();
        if (!fromContract || !toContract)
            return;
        const fromDecimal = createDecimal(fromInfo.decimals);
        const toDecimal = createDecimal(toInfo.decimals);
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
                        amount: weieth.encode(txUni.value),
                        address: txUni.to,
                        expectedAmount,
                        data: ethHex.encode(txUni.data),
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
//# sourceMappingURL=uniswap-common.js.map