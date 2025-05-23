import { concatBytes } from '@noble/hashes/utils';
import { type ContractInfo, createContract } from '../abi/decoder.ts';
import { default as UNISWAP_V3_ROUTER, UNISWAP_V3_ROUTER_CONTRACT } from '../abi/uniswap-v3.ts';
import { type IWeb3Provider, ethHex } from '../utils.ts';
import * as uni from './uniswap-common.ts';

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
] as const;

export const Fee: Record<string, number> = {
  LOW: 500,
  MEDIUM: 3000,
  HIGH: 10000,
};

type Route = { path?: Uint8Array; fee?: number; amountIn?: bigint; amountOut?: bigint; p?: any };

function basePaths(a: string, b: string, exactOutput: boolean = false) {
  let res: Route[] = [];
  for (let fee in Fee) res.push({ fee: Fee[fee], p: [a, b] });
  const wA = uni.wrapContract(a);
  const wB = uni.wrapContract(b);
  const BASES: (ContractInfo & { contract: string })[] = uni.COMMON_BASES.filter(
    (c) => c && c.contract && c.contract !== wA && c.contract !== wB
  ) as (ContractInfo & { contract: string })[];
  const packFee = (n: string) => Fee[n].toString(16).padStart(6, '0');
  for (let c of BASES) {
    for (let fee1 in Fee) {
      for (let fee2 in Fee) {
        let path = [wA, packFee(fee1), c.contract, packFee(fee2), wB].map((i) => ethHex.decode(i));
        if (exactOutput) path = path.reverse();
        res.push({ path: concatBytes(...path) });
      }
    }
  }
  return res;
}

async function bestPath(
  net: IWeb3Provider,
  a: string,
  b: string,
  amountIn?: bigint,
  amountOut?: bigint
) {
  if ((amountIn && amountOut) || (!amountIn && !amountOut))
    throw new Error('uniswapV3.bestPath: provide only one amount');
  const quoter = createContract(QUOTER_ABI, net, QUOTER_ADDRESS);
  let paths = basePaths(a, b, !!amountOut);
  for (let i of paths) {
    if (!i.path && !i.fee) continue;
    const opt = { ...i, tokenIn: a, tokenOut: b, amountIn, amountOut, sqrtPriceLimitX96: 0 };
    const method = 'quoteExact' + (amountIn ? 'Input' : 'Output') + (i.path ? '' : 'Single');
    // TODO: remove any
    i[amountIn ? 'amountOut' : 'amountIn'] = (quoter as any)[method].call(opt);
  }
  paths = (await uni.awaitDeep(paths, true)) as any;
  paths = paths.filter((i) => i.amountIn || i.amountOut);
  paths.sort((a: any, b: any) =>
    Number(amountIn ? b.amountOut - a.amountOut : a.amountIn - b.amountIn)
  );
  if (!paths.length) throw new Error('uniswap: cannot find path');
  return paths[0];
}

const ROUTER_CONTRACT = createContract(UNISWAP_V3_ROUTER, undefined, UNISWAP_V3_ROUTER_CONTRACT);

export type TxOpt = {
  slippagePercent: number;
  ttl: number;
  sqrtPriceLimitX96?: bigint;
  deadline?: number;
  fee?: { fee: number; to: string };
};

export function txData(
  to: string,
  input: string,
  output: string,
  route: Route,
  amountIn?: bigint,
  amountOut?: bigint,
  opt: TxOpt = uni.DEFAULT_SWAP_OPT
): {
  to: string;
  value: bigint;
  data: Uint8Array;
  allowance:
    | {
        token: string;
        amount: bigint;
      }
    | undefined;
} {
  opt = { ...uni.DEFAULT_SWAP_OPT, ...opt };
  const err = 'Uniswap v3: ';
  if (!uni.isValidUniAddr(input)) throw new Error(err + 'invalid input address');
  if (!uni.isValidUniAddr(output)) throw new Error(err + 'invalid output address');
  if (!uni.isValidEthAddr(to)) throw new Error(err + 'invalid to address');
  if (opt.fee && !uni.isValidUniAddr(opt.fee.to))
    throw new Error(err + 'invalid fee recepient addresss');
  if (input === 'eth' && output === 'eth')
    throw new Error(err + 'both input and output cannot be eth');
  if ((amountIn && amountOut) || (!amountIn && !amountOut))
    throw new Error(err + 'specify either amountIn or amountOut, but not both');
  if (
    (amountIn && !route.amountOut) ||
    (amountOut && !route.amountIn) ||
    (!route.fee && !route.path)
  )
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
    amountIn: (amountIn || route.amountIn) as bigint,
    amountOut: (amountOut || route.amountOut) as bigint,
    sqrtPriceLimitX96: opt.sqrtPriceLimitX96 || BigInt(0),
    amountInMaximum: undefined as bigint | undefined,
    amountOutMinimum: undefined as bigint | undefined,
  };
  args.amountInMaximum = uni.addPercent(args.amountIn, opt.slippagePercent);
  args.amountOutMinimum = uni.addPercent(args.amountOut, -opt.slippagePercent);
  const method = ('exact' + (amountIn ? 'Input' : 'Output') + (!args.path ? 'Single' : '')) as
    | 'exactInput'
    | 'exactOutput'
    | 'exactInputSingle'
    | 'exactOutputSingle';
  // TODO: remove unknown
  const calldatas = [(ROUTER_CONTRACT[method].encodeInput as (v: unknown) => Uint8Array)(args)];
  if (input === 'eth' && amountOut) calldatas.push(ROUTER_CONTRACT['refundETH'].encodeInput());
  // unwrap
  if (routerMustCustody) {
    calldatas.push(
      (ROUTER_CONTRACT as any)[
        (output === 'eth' ? 'unwrapWETH9' : 'sweepToken') + (opt.fee ? 'WithFee' : '')
      ].encodeInput({
        token: uni.wrapContract(output),
        amountMinimum: args.amountOutMinimum,
        recipient: to,
        feeBips: opt.fee && opt.fee.fee * 10000,
        feeRecipient: opt.fee && opt.fee.to,
      })
    );
  }
  const data =
    calldatas.length === 1 ? calldatas[0] : ROUTER_CONTRACT['multicall'].encodeInput(calldatas);
  const value = input === 'eth' ? (amountIn ? amountIn : args.amountInMaximum) : BigInt(0);
  const allowance =
    input !== 'eth'
      ? { token: input, amount: amountIn ? amountIn : args.amountInMaximum }
      : undefined;
  return { to: UNISWAP_V3_ROUTER_CONTRACT, value, data, allowance };
}

// Here goes Exchange API. Everything above is SDK.
export default class UniswapV3 extends uni.UniswapAbstract {
  name = 'Uniswap V3';
  contract: string = UNISWAP_V3_ROUTER_CONTRACT;
  bestPath(fromCoin: string, toCoin: string, inputAmount: bigint): Promise<Route> {
    return bestPath(this.net, fromCoin, toCoin, inputAmount);
  }
  txData(
    toAddress: string,
    fromCoin: string,
    toCoin: string,
    path: any,
    inputAmount?: bigint,
    outputAmount?: bigint,
    opt: uni.SwapOpt = uni.DEFAULT_SWAP_OPT
  ): any {
    return txData(toAddress, fromCoin, toCoin, path, inputAmount, outputAmount, {
      ...uni.DEFAULT_SWAP_OPT,
      ...opt,
    });
  }
}
