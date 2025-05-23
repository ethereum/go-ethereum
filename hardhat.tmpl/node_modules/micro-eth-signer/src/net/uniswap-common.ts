import { tokenFromSymbol } from '../abi/index.ts';
import { addr } from '../index.ts';
import { type IWeb3Provider, createDecimal, ethHex, isBytes, weieth } from '../utils.ts';

export type SwapOpt = { slippagePercent: number; ttl: number };
export const DEFAULT_SWAP_OPT: SwapOpt = { slippagePercent: 0.5, ttl: 30 * 60 };

// [res?.id, res?.payinAddress, res?.amountExpectedTo]
export type ExchangeTx = {
  address: string;
  amount: string;
  currency: string;
  expectedAmount: string;
  data?: string;
  allowance?: { token: string; contract: string; amount: string };
  txId?: string;
};

export type SwapElm = {
  name: string; // Human readable exchange name
  expectedAmount: string;
  tx: (fromAddress: string, toAddress: string) => Promise<ExchangeTx>;
};

export function addPercent(n: bigint, _perc: number): bigint {
  const perc = BigInt((_perc * 10000) | 0);
  const p100 = BigInt(100) * BigInt(10000);
  return ((p100 + perc) * n) / p100;
}

export function isPromise(o: unknown): boolean {
  if (!o || !['object', 'function'].includes(typeof o)) return false;
  return typeof (o as any).then === 'function';
}

// Promise.all(), but allows to wait for nested objects with promises and to ignore errors.
// It's hard to make ignore_errors argument optional in current TS.
export type UnPromise<T> = T extends Promise<infer U> ? U : T;
type NestedUnPromise<T> = { [K in keyof T]: NestedUnPromise<UnPromise<T[K]>> };
type UnPromiseIgnore<T> = T extends Promise<infer U> ? U | undefined : T;
type NestedUnPromiseIgnore<T> = { [K in keyof T]: NestedUnPromiseIgnore<UnPromiseIgnore<T[K]>> };
export async function awaitDeep<T, E extends boolean | undefined>(
  o: T,
  ignore_errors: E
): Promise<E extends true ? NestedUnPromiseIgnore<T> : NestedUnPromise<T>> {
  let promises: Promise<any>[] = [];
  const traverse = (o: any): any => {
    if (Array.isArray(o)) return o.map((i) => traverse(i));
    if (isBytes(o)) return o;
    if (isPromise(o)) return { awaitDeep: promises.push(o) };
    if (typeof o === 'object') {
      let ret: Record<string, any> = {};
      for (let k in o) ret[k] = traverse(o[k]);
      return ret;
    }
    return o;
  };
  let out = traverse(o);
  let values: any[];
  if (!ignore_errors) values = await Promise.all(promises);
  else {
    values = (await Promise.allSettled(promises)).map((i) =>
      i.status === 'fulfilled' ? i.value : undefined
    );
  }
  const trBack = (o: any): any => {
    if (Array.isArray(o)) return o.map((i) => trBack(i));
    if (isBytes(o)) return o;
    if (typeof o === 'object') {
      if (typeof o === 'object' && o.awaitDeep) return values[o.awaitDeep - 1];
      let ret: Record<string, any> = {};
      for (let k in o) ret[k] = trBack(o[k]);
      return ret;
    }
    return o;
  };
  return trBack(out);
}

export type CommonBase = {
  contract: string;
} & import('../abi/decoder.js').ContractInfo;
export const COMMON_BASES: CommonBase[] = [
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
export const WETH: string = tokenFromSymbol('WETH')!.contract;
if (!WETH) throw new Error('WETH is undefined!');

export function wrapContract(contract: string): string {
  contract = contract.toLowerCase();
  return contract === 'eth' ? WETH : contract;
}

export function sortTokens(a: string, b: string): [string, string] {
  a = wrapContract(a);
  b = wrapContract(b);
  if (a === b) throw new Error('uniswap.sortTokens: same token!');
  return a < b ? [a, b] : [b, a];
}

export function isValidEthAddr(address: string): boolean {
  return addr.isValid(address);
}

export function isValidUniAddr(address: string): boolean {
  return address === 'eth' || isValidEthAddr(address);
}

export type Token = { decimals: number; contract: string; symbol: string };

function getToken(token: 'eth' | Token): Token {
  if (typeof token === 'string' && token.toLowerCase() === 'eth')
    return { symbol: 'ETH', decimals: 18, contract: 'eth' };
  return token as Token;
}

export abstract class UniswapAbstract {
  abstract name: string;
  abstract contract: string;
  abstract bestPath(fromCoin: string, toCoin: string, inputAmount: bigint): any;
  abstract txData(
    toAddress: string,
    fromCoin: string,
    toCoin: string,
    path: any,
    inputAmount?: bigint,
    outputAmount?: bigint,
    opt?: { slippagePercent: number }
  ): any;
  readonly net: IWeb3Provider;
  constructor(net: IWeb3Provider) {
    this.net = net;
  }
  // private async coinInfo(netName: string) {
  //   if (!validateAddr(netName)) return;
  //   if (netName === 'eth') return { symbol: 'ETH', decimals: 18 };
  //   //return await this.mgr.tokenInfo('eth', netName);
  // }
  async swap(
    fromCoin: 'eth' | Token,
    toCoin: 'eth' | Token,
    amount: string,
    opt: SwapOpt = DEFAULT_SWAP_OPT
  ): Promise<
    | {
        name: string;
        expectedAmount: string;
        tx: (
          _fromAddress: string,
          toAddress: string
        ) => Promise<{
          amount: string;
          address: any;
          expectedAmount: string;
          data: string;
          allowance: any;
        }>;
      }
    | undefined
  > {
    const fromInfo = getToken(fromCoin);
    const toInfo = getToken(toCoin);
    if (!fromInfo || !toInfo) return;
    const fromContract = fromInfo.contract.toLowerCase();
    const toContract = toInfo.contract.toLowerCase();
    if (!fromContract || !toContract) return;
    const fromDecimal = createDecimal(fromInfo.decimals);
    const toDecimal = createDecimal(toInfo.decimals);
    const inputAmount = fromDecimal.decode(amount);
    try {
      const path = await this.bestPath(fromContract, toContract, inputAmount);
      const expectedAmount = toDecimal.encode(path.amountOut as bigint);
      return {
        name: this.name,
        expectedAmount,
        tx: async (_fromAddress: string, toAddress: string) => {
          const txUni = this.txData(
            toAddress,
            fromContract,
            toContract,
            path,
            inputAmount,
            undefined,
            opt
          );
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
    } catch (e) {
      // @ts-ignore
      console.log('E', e);
      return;
    }
  }
}
