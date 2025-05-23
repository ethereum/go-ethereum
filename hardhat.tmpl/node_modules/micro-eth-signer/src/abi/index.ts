import { addr } from '../address.ts';
import { Transaction } from '../index.ts';
import { ethHex } from '../utils.ts';
import {
  type ContractABI,
  type ContractInfo,
  Decoder,
  createContract,
  deployContract,
  events,
} from './decoder.ts';
import { default as ERC1155 } from './erc1155.ts';
import { default as ERC20 } from './erc20.ts';
import { default as ERC721 } from './erc721.ts';
import { default as KYBER_NETWORK_PROXY, KYBER_NETWORK_PROXY_CONTRACT } from './kyber.ts';
import { default as UNISWAP_V2_ROUTER, UNISWAP_V2_ROUTER_CONTRACT } from './uniswap-v2.ts';
import { default as UNISWAP_V3_ROUTER, UNISWAP_V3_ROUTER_CONTRACT } from './uniswap-v3.ts';
import { default as WETH, WETH_CONTRACT } from './weth.ts';

// We need to export raw contracts, because 'CONTRACTS' object requires to know address it is not static type
// so it cannot be re-used in createContract with nice types.
export {
  ERC1155,
  ERC20,
  ERC721,
  KYBER_NETWORK_PROXY_CONTRACT,
  UNISWAP_V2_ROUTER_CONTRACT,
  UNISWAP_V3_ROUTER_CONTRACT,
  WETH,
};

export { Decoder, createContract, deployContract, events };
// Export decoder related types
export type { ContractABI, ContractInfo };

export const TOKENS: Record<string, ContractInfo> = /* @__PURE__ */ (() =>
  Object.freeze(
    Object.fromEntries(
      (
        [
          ['UNI', '0x1f9840a85d5af5bf1d1762f925bdaddc4201f984'],
          ['BAT', '0x0d8775f648430679a709e98d2b0cb6250d2887ef'],
          // Required for Uniswap multi-hop routing
          ['USDT', '0xdac17f958d2ee523a2206206994597c13d831ec7', 6, 1],
          ['USDC', '0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48', 6, 1],
          ['WETH', '0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2'],
          ['WBTC', '0x2260fac5e5542a773aa44fbcfedf7c193bc2c599', 8],
          ['DAI', '0x6b175474e89094c44da98b954eedeac495271d0f', 18, 1],
          ['COMP', '0xc00e94cb662c3520282e6f5717214004a7f26888'],
          ['MKR', '0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2'],
          ['AMPL', '0xd46ba6d942050d489dbd938a2c909a5d5039a161', 9],
        ] as [string, string, number?, number?][]
      ).map(([symbol, addr, decimals, price]) => [
        addr as string,
        { abi: 'ERC20' as const, symbol, decimals: decimals || 18, price },
      ])
    )
  ))();
// <address, contractInfo>
export const CONTRACTS: Record<string, ContractInfo> = /* @__PURE__ */ (() =>
  Object.freeze({
    [UNISWAP_V2_ROUTER_CONTRACT]: { abi: UNISWAP_V2_ROUTER, name: 'UNISWAP V2 ROUTER' },
    [KYBER_NETWORK_PROXY_CONTRACT]: { abi: KYBER_NETWORK_PROXY, name: 'KYBER NETWORK PROXY' },
    [UNISWAP_V3_ROUTER_CONTRACT]: { abi: UNISWAP_V3_ROUTER, name: 'UNISWAP V3 ROUTER' },
    ...TOKENS,
    [WETH_CONTRACT]: { abi: WETH, name: 'WETH Token', decimals: 18, symbol: 'WETH' },
  }))();

export const tokenFromSymbol = (
  symbol: string
): {
  contract: string;
} & ContractInfo => {
  for (let c in TOKENS) {
    if (TOKENS[c].symbol === symbol) return Object.assign({ contract: c }, TOKENS[c]);
  }
  throw new Error('unknown token');
};

const getABI = (info: ContractInfo) => {
  if (typeof info.abi === 'string') {
    if (info.abi === 'ERC20') return ERC20;
    else if (info.abi === 'ERC721') return ERC721;
    else throw new Error(`getABI: unknown abi type=${info.abi}`);
  }
  return info.abi;
};

export type DecoderOpt = {
  customContracts?: Record<string, ContractInfo>;
  noDefault?: boolean; // don't add default contracts
};

// TODO: export? Seems useful enough
// We cannot have this inside decoder itself,
// since it will create dependencies on all default contracts
const getDecoder = (opt: DecoderOpt = {}) => {
  const decoder = new Decoder();
  const contracts: Record<string, ContractInfo> = {};
  // Add contracts
  if (!opt.noDefault) Object.assign(contracts, CONTRACTS);
  if (opt.customContracts) {
    for (const k in opt.customContracts) contracts[k.toLowerCase()] = opt.customContracts[k];
  }
  // Contract info validation
  for (const k in contracts) {
    if (!addr.isValid(k)) throw new Error(`getDecoder: invalid contract address=${k}`);
    const c = contracts[k];
    if (c.symbol !== undefined && typeof c.symbol !== 'string')
      throw new Error(`getDecoder: wrong symbol type=${c.symbol}`);
    if (c.decimals !== undefined && !Number.isSafeInteger(c.decimals))
      throw new Error(`getDecoder: wrong decimals type=${c.decimals}`);
    if (c.name !== undefined && typeof c.name !== 'string')
      throw new Error(`getDecoder: wrong name type=${c.name}`);
    if (c.price !== undefined && typeof c.price !== 'number')
      throw new Error(`getDecoder: wrong price type=${c.price}`);
    decoder.add(k, getABI(c)); // validates c.abi
  }
  return { decoder, contracts };
};

// These methods are for case when user wants to inspect tx/logs/receipt,
// but doesn't know anything about which contract is used. If you work with
// specific contract it is better to use 'createContract' which will return nice types.
// 'to' can point to specific known contract, but also can point to any address (it is part of tx)
// 'to' should be part of real tx you want to parse, not hardcoded contract!
// Even if contract is unknown, we still try to process by known function signatures
// from other contracts.

// Can be used to parse tx or 'eth_getTransactionReceipt' output
export const decodeData = (to: string, data: string, amount?: bigint, opt: DecoderOpt = {}) => {
  if (!addr.isValid(to)) throw new Error(`decodeData: wrong to=${to}`);
  if (amount !== undefined && typeof amount !== 'bigint')
    throw new Error(`decodeData: wrong amount=${amount}`);
  const { decoder, contracts } = getDecoder(opt);
  return decoder.decode(to, ethHex.decode(data), {
    contract: to,
    contracts, // NOTE: we need whole contracts list here, since exchanges can use info about other contracts (tokens)
    contractInfo: contracts[to.toLowerCase()], // current contract info (for tokens)
    amount, // Amount is not neccesary, but some hints won't work without it (exchange eth to some tokens)
  });
};

// Requires deps on tx, but nicer API.
// Doesn't cover all use cases of decodeData, since it can't parse 'eth_getTransactionReceipt'
export const decodeTx = (transaction: string, opt: DecoderOpt = {}) => {
  const tx = Transaction.fromHex(transaction);
  return decodeData(tx.raw.to, tx.raw.data, tx.raw.value, opt);
};

// Parses output of eth_getLogs/eth_getTransactionReceipt
export const decodeEvent = (to: string, topics: string[], data: string, opt: DecoderOpt = {}) => {
  if (!addr.isValid(to)) throw new Error(`decodeEvent: wrong to=${to}`);
  const { decoder, contracts } = getDecoder(opt);
  return decoder.decodeEvent(to, topics, data, {
    contract: to,
    contracts,
    contractInfo: contracts[to.toLowerCase()],
    // amount here is not used by our hooks. Should we ask it for consistency?
  });
};
