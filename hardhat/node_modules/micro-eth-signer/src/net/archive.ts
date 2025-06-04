import { ERC1155, ERC20, ERC721, WETH, createContract, events } from '../abi/index.ts';
import { Transaction } from '../index.ts';
import { TxVersions, legacySig, type AccessList } from '../tx.ts';
import { amounts, ethHex, hexToNumber, type IWeb3Provider, type Web3CallArgs } from '../utils.ts';

/*
Methods to fetch list of transactions from any ETH node RPC.
It should be easy. However, this is sparta^W ethereum, so, prepare to suffer.

The network is not directly called: `ArchiveNodeProvider#rpc` calls `Web3Provider`.

- There is no simple & fast API inside nodes, all external API create their own namespace for this
- API is different between nodes: erigon uses streaming, other nodes use pagination
- Recently, Erigon have been also adding pagination
- For token transactions: download block headers, look at bloom filter, download affected blocks
- There is a good `getLogs` API for contracts, but nothing for ETH transfers
- `trace_filter` is slow: it not only finds the transaction, but also executes them
- It's good that it allows to get internal transactions
- The whole thing could be 10x simpler if there was an event in logs for ETH transfer
- For most cases, we only need to see last transactions and know blocks of last txs, which is 20x faster
- This creates a lot of requests to node (2 per tx, 1 per block, and some more depends on block range limits)

Recommended software:

- eth-nodes-for-rent are bad, because of their limits and timeouts
- erigon nodes are fast, taking ~15 seconds per batch
- reth has 100-block limit for trace_filter, requiring 190k requests just get transactions
*/

// Utils
const ethNum = (n: number | bigint | undefined) =>
  `0x${!n ? '0' : n.toString(16).replace(/^0+/, '')}`;

const ERC_TRANSFER = events(ERC20).Transfer;
const WETH_DEPOSIT = events(WETH).Deposit;
const WETH_WITHDRAW = events(WETH).Withdrawal;
const ERC721_TRANSFER = events(ERC721).Transfer;
const ERC1155_SINGLE = events(ERC1155).TransferSingle;
const ERC1155_BATCH = events(ERC1155).TransferBatch;

const ERC165 = [
  //     function supportsInterface(bytes4 interfaceID) external view returns (bool);
  {
    type: 'function',
    name: 'supportsInterface',
    inputs: [{ name: 'interfaceID', type: 'bytes4' }],
    outputs: [{ type: 'bool' }],
  },
] as const;

const CONTRACT_CAPABILITIES: Record<string, string> = {
  erc165: '0x01ffc9a7',
  erc165_check: '0xffffffff',
  erc20: '0x36372b07',
  erc721: '0x80ac58cd',
  erc721_metadata: '0x5b5e139f',
  erc721_enumerable: '0x780e9d63',
  erc1155: '0xd9b67a26',
  erc1155_tokenreceiver: '0x4e2312e0',
  erc1155_metadata: '0x0e89341c',
};

function group<T>(items: T[], s: string | ((i: T) => string)): Record<string, T[]> {
  let res: Record<string, T[]> = {};
  for (let i of items) {
    const key = typeof s === 'function' ? s(i) : (i as any)[s];
    if (!res[key]) res[key] = [];
    res[key].push(i);
  }
  return res;
}
// Output types
export type BlockInfo = {
  baseFeePerGas: bigint;
  difficulty: bigint;
  extraData: string;
  gasLimit: bigint;
  gasUsed: bigint;
  hash: string;
  logsBloom: string;
  miner: string;
  mixHash: string;
  nonce: string;
  number: number;
  parentHash: string;
  receiptsRoot: string;
  sha3Uncles: string;
  size: number;
  stateRoot: string;
  timestamp: number;
  totalDifficulty?: bigint;
  transactions: string[]; // transaction hashes (if false)
  transactionsRoot: string;
  uncles: string[];
};

export type Action = {
  action: {
    from: string;
    callType: string;
    gas: bigint;
    input: string;
    to: string;
    value: bigint;
  };
  blockHash: string;
  blockNumber: number;
  result: { gasUsed: bigint; output: string };
  subtraces: number;
  traceAddress: string[];
  transactionHash: string;
  transactionPosition: number;
  type: string;
};

export type Log = {
  address: string;
  topics: string[];
  data: string;
  blockNumber: number;
  transactionHash: string;
  transactionIndex: number;
  blockHash: string;
  logIndex: number;
  removed: boolean;
};

export type TxInfo = {
  blockHash: string;
  blockNumber: number;
  hash: string;
  accessList?: AccessList;
  transactionIndex: number;
  type: number;
  nonce: bigint;
  input: string;
  r: bigint;
  s: bigint;
  chainId: bigint;
  v: bigint;
  yParity?: string;
  gas: bigint;
  maxPriorityFeePerGas?: bigint;
  from: string;
  to: string;
  maxFeePerGas?: bigint;
  value: bigint;
  gasPrice: bigint;
  // blobs
  maxFeePerBlobGas?: bigint;
  blobVersionedHashes?: string[];
};

export type TxReceipt = {
  transactionHash: string;
  blockHash: string;
  blockNumber: number;
  logsBloom: string;
  gasUsed: bigint;
  contractAddress: string | null;
  cumulativeGasUsed: bigint;
  transactionIndex: number;
  from: string;
  to: string;
  type: number;
  effectiveGasPrice: bigint;
  logs: Log[];
  status: number;
  blobGasPrice?: bigint;
  blobGasUsed?: bigint;
};

export type Unspent = {
  symbol: 'ETH';
  decimals: number;
  balance: bigint;
  nonce: number;
  // useful for wallets to know if there was transactions related to wallet
  // NOTE: even if nonce is zero, there can be transfers to wallet
  // can be used to check before fetching all transactions
  active: boolean;
};

type ERC20Token = {
  abi: 'ERC20';
  name?: string;
  symbol?: string;
  decimals?: number;
  totalSupply: bigint;
};
type ERC721Token = {
  abi: 'ERC721';
  name?: string;
  symbol?: string;
  totalSupply?: bigint;
  enumerable?: boolean;
  metadata?: boolean;
};
type ERC1155Token = { abi: 'ERC1155' };
export type TokenInfo = { contract: string } & (ERC20Token | ERC721Token | ERC1155Token);
// Main idea: there is broken contracts that behave strange, instead of crashing we return this error.
// Separate error allows easily to discriminate between "strange contract" and "input invalid" | "network error"
// We still want to crash on network errors, but if there is per-contract error it is better to continue batched request
// like tokenTransfers
type TokenError = { contract: string; error: string };
type TokenBalanceSingle = Map<bigint, bigint>;

// This is unified type for ERC-20 | ERC-721 | ERC-1155
// ERC20: Record<contractAddress, Map<1n, tokenValue>> - ERC-20 has only single tokenId (always!)
// ERC721: Record<contractAddress, Map<tokenId, 1n>> - ERC-721 every tokenId is 0n or 1n
// ERC-1155: Record<contractAddress, Map<tokenId, tokenValue> - ERC-1155 each tokenId can have value (merge of ERC-20 and ERC-721)
export type TokenBalances = Record<string, TokenBalanceSingle | TokenError>;
export type Topics = (string | null | (string | null)[])[];
export type Transfer = { from: string; to?: string; value: bigint };
// tokens: Map<tokenId, value> (same as balances!)
export type TokenTransfer = TokenInfo & { from: string; to: string; tokens: Map<bigint, bigint> };

export type TxTransfers = {
  // This is most interesting info about tx for wallets
  hash: string;
  timestamp?: number;
  block?: number;
  transfers: Transfer[];
  tokenTransfers: TokenTransfer[];
  reverted: boolean;
  // This contains everything about tx in raw format
  info: {
    type: keyof typeof TxVersions;
    info: TxInfo;
    receipt: TxReceipt;
    raw?: string;
    block: BlockInfo;
    actions: Action[];
  };
};

/**
 * Callbacks are needed, because we want to call getTx / getBlock / getTokenInfo
 * requests as fast as possible, to reduce amount of sequential execution.
 * If we retrieve 10 pages of transactions, we can call per tx
 * callbacks for transaction from first page before all other pages fetched.
 *
 * Ensure caching: they can be called multiple times for same tx / block.
 */
export type Callbacks = {
  txCallback?: (txHash: string) => void;
  blockCallback?: (blockNum: number) => void;
  contractCallback?: (contrct: string) => void;
};

export type Pagination = { fromBlock?: number; toBlock?: number };
export type TraceOpts = Callbacks &
  Pagination & {
    perRequest?: number;
    limitTrace?: number;
  };
export type LogOpts = Callbacks &
  (
    | Pagination
    | {
        fromBlock: number;
        toBlock: number;
        limitLogs: number; // limit block range per request
      }
  );
export type Balances = {
  balances: Record<string, bigint>;
  tokenBalances: Record<string, Record<string, bigint>>;
};
export type TxInfoOpts = Callbacks & { ignoreTxRebuildErrors?: boolean };
export type TxAllowances = Record<string, Record<string, bigint>>;

function fixBlock(block: BlockInfo) {
  block.timestamp = Number(block.timestamp) * 1000;
  block.size = Number(block.size);
  if (block.number && block.number !== null) block.number = Number(block.number);
  for (const i of [
    'baseFeePerGas',
    'difficulty',
    'gasLimit',
    'gasUsed',
    'totalDifficulty',
  ] as const) {
    if (block[i] && block[i] !== null) block[i] = BigInt(block[i]);
  }
}
function fixAction(action: Action, opts: Callbacks = {}) {
  action.action.value = BigInt(action.action.value);
  action.action.gas = BigInt(action.action.gas);
  action.result.gasUsed = BigInt(action.result.gasUsed);
  if (opts.txCallback) opts.txCallback(action.transactionHash);
  if (opts.blockCallback) opts.blockCallback(action.blockNumber);
}
// Fixes type of network response inplace
function fixLog(log: Log, opts: Callbacks = {}) {
  log.blockNumber = Number(log.blockNumber);
  log.transactionIndex = Number(log.transactionIndex);
  log.logIndex = Number(log.logIndex);
  if (opts.txCallback) opts.txCallback(log.transactionHash);
  if (opts.blockCallback) opts.blockCallback(log.blockNumber);
  if (opts.contractCallback) opts.contractCallback(log.address);
  return log;
}
function fixTxInfo(info: TxInfo) {
  for (const i of ['blockNumber', 'type', 'transactionIndex'] as const) info[i] = Number(info[i]);
  for (const i of [
    'nonce',
    'r',
    's',
    'chainId',
    'v',
    'gas',
    'maxPriorityFeePerGas',
    'maxFeePerGas',
    'value',
    'gasPrice',
    'maxFeePerBlobGas',
  ] as const) {
    if (info[i] !== undefined && info[i] !== null) info[i] = BigInt(info[i]!);
  }
  return info;
}

function fixTxReceipt(receipt: TxReceipt) {
  for (const i of ['blockNumber', 'type', 'transactionIndex', 'status'] as const)
    receipt[i] = Number(receipt[i]);
  for (const i of [
    'gasUsed',
    'cumulativeGasUsed',
    'effectiveGasPrice',
    'blobGasPrice',
    'blobGasUsed',
  ] as const) {
    if (receipt[i] !== undefined) receipt[i] = BigInt(receipt[i]!);
  }
  for (const log of receipt.logs) fixLog(log);
  return receipt;
}
function validateCallbacks(opts: Record<string, unknown>) {
  for (const i of ['txCallback', 'blockCallback', 'contractCallback']) {
    if (opts[i] !== undefined && typeof opts[i] !== 'function')
      throw new Error(`validateCallbacks: ${i} should be function`);
  }
}

function validatePagination(opts: Record<string, unknown>) {
  for (const i of ['fromBlock', 'toBlock']) {
    if (opts[i] === undefined || Number.isSafeInteger(opts[i])) continue;
    throw new Error(
      `validatePagination: wrong field ${i}=${opts[i]}. Should be integer or undefined`
    );
  }
}

function validateTraceOpts(opts: Record<string, unknown>) {
  validatePagination(opts);
  for (const i of ['perRequest', 'limitTrace']) {
    if (opts[i] === undefined || Number.isSafeInteger(opts[i])) continue;
    throw new Error(
      `validateTraceOpts: wrong field ${i}=${opts[i]}. Should be integer or undefined`
    );
  }
  if (opts.limitTrace !== undefined) {
    if (opts.fromBlock === undefined || opts.toBlock === undefined)
      throw new Error('validateTraceOpts: fromBlock/toBlock required if limitTrace is present');
  }
  validateCallbacks(opts);
}

function validateLogOpts(opts: Record<string, unknown>) {
  validatePagination(opts);
  for (const i of ['limitLogs']) {
    if (opts[i] === undefined || Number.isSafeInteger(opts[i])) continue;
    throw new Error(`validateLogOpts: wrong field ${i}=${opts[i]}. Should be integer or undefined`);
  }
  if (opts.limitLogs !== undefined) {
    if (opts.fromBlock === undefined || opts.toBlock === undefined)
      throw new Error('validateLogOpts: fromBlock/toBlock required if limitLogs is present');
  }
  validateCallbacks(opts);
}

export type JsonrpcInterface = {
  call: (method: string, ...args: any[]) => Promise<any>;
};

// Promise.all for objects, undefined if error
async function wait<T extends Record<string, Promise<any>>>(
  obj: T
): Promise<{ [K in keyof T]?: T[K] extends Promise<infer R> ? R : never }> {
  const keys = Object.keys(obj) as (keyof T)[];
  const p = await Promise.allSettled(Object.values(obj));
  const res = p.map((r, i) => [keys[i], r.status === 'fulfilled' ? r.value : undefined]);
  return Object.fromEntries(res) as { [K in keyof T]?: T[K] extends Promise<infer R> ? R : never };
}

const isReverted = (e: Error) => e instanceof Error && e.message.toLowerCase().includes('revert');

/**
 * Transaction-related code around Web3Provider.
 * High-level methods are `height`, `unspent`, `transfers`, `allowances` and `tokenBalances`.
 *
 * Low-level methods are `blockInfo`, `internalTransactions`, `ethLogs`, `tokenTransfers`, `wethTransfers`,
 * `tokenInfo` and `txInfo`.
 */
export class Web3Provider implements IWeb3Provider {
  private rpc: JsonrpcInterface;
  constructor(rpc: JsonrpcInterface) {
    this.rpc = rpc;
  }

  call(method: string, ...args: any[]): Promise<any> {
    return this.rpc.call(method, ...args);
  }
  ethCall(args: Web3CallArgs, tag = 'latest'): Promise<any> {
    return this.rpc.call('eth_call', args, tag);
  }
  async estimateGas(args: Web3CallArgs, tag = 'latest'): Promise<bigint> {
    return hexToNumber(await this.rpc.call('eth_estimateGas', args, tag));
  }

  // Timestamp is available only inside blocks
  async blockInfo(block: number): Promise<BlockInfo> {
    const res = await this.call('eth_getBlockByNumber', ethNum(block), false);
    fixBlock(res);
    return res;
  }

  async unspent(address: string): Promise<Unspent> {
    let [balance, nonce] = await Promise.all([
      this.call('eth_getBalance', address, 'latest'),
      this.call('eth_getTransactionCount', address, 'latest'),
    ]);
    balance = BigInt(balance);
    nonce = BigInt(nonce);
    return {
      symbol: 'ETH',
      decimals: amounts.ETH_PRECISION,
      balance,
      nonce,
      // Note: account can be active even if nonce!==0!
      active: balance > 0 || nonce !== 0,
    };
  }
  async height(): Promise<number> {
    return Number.parseInt(await this.call('eth_blockNumber'));
  }

  async traceFilterSingle(address: string, opts: TraceOpts = {}): Promise<any> {
    const res = await this.call('trace_filter', {
      fromBlock: ethNum(opts.fromBlock),
      toBlock: ethNum(opts.toBlock),
      toAddress: [address],
      fromAddress: [address],
    });
    for (const action of res) fixAction(action, opts);
    return res;
  }

  async internalTransactions(address: string, opts: TraceOpts = {}): Promise<any[]> {
    if (typeof address !== 'string') throw new Error('internalTransactions: wrong address');
    validateTraceOpts(opts);
    // For reth
    if (opts.limitTrace) {
      const promises = [];
      for (let i = opts.fromBlock!; i <= opts.toBlock!; i += opts.limitTrace)
        promises.push(
          this.traceFilterSingle(address, { fromBlock: i, toBlock: i + opts.limitTrace })
        );
      const out = [];
      for (const i of await Promise.all(promises)) out.push(...i);
      return out;
    }
    let lastBlock = opts.fromBlock || 0;
    const perBlock: Record<number, number> = {};
    const out: Action[] = [];
    for (;;) {
      const params: Record<string, any> = {
        fromBlock: ethNum(lastBlock),
        toAddress: [address],
        fromAddress: [address],
        after: perBlock[lastBlock] || 0, // we cannot just store after, since fromBlock changes to last block
      };
      if (opts.toBlock !== undefined) params.toBlock = ethNum(opts.toBlock);
      if (opts.perRequest !== undefined) params.count = opts.perRequest;

      const res = await this.call('trace_filter', params);
      if (!res.length) break;
      for (const action of res) {
        fixAction(action, opts);
        if (perBlock[action.blockNumber] === undefined) perBlock[action.blockNumber] = 0;
        perBlock[action.blockNumber]++;
        out.push(action);
        lastBlock = Math.max(lastBlock, action.blockNumber);
      }
    }
    return out;
  }

  async contractCapabilities(
    address: string,
    capabilities: typeof CONTRACT_CAPABILITIES = {}
  ): Promise<{
    [k: string]: boolean;
  }> {
    const all = { ...CONTRACT_CAPABILITIES, ...capabilities };
    let c = createContract(ERC165, this, address);
    const keys = Object.keys(all);
    // TODO: what about revert?
    // if reverted -> all capabilities disabled
    try {
      const promises = await Promise.all(
        Object.values(all).map((i) => c.supportsInterface.call(ethHex.decode(i)))
      );
      const res = Object.fromEntries(keys.map((k, i) => [k, promises[i]]));
      // if somehow there is same method, but it doesn't support erc165, then it is different method!
      // erc165_check if sailsafe when there is method that always returns true
      if (!res.erc165 || res.erc165_check) for (const k in res) res[k] = false;
      return res;
    } catch (e) {
      // If execution reverted: contract doesn't support ERC165
      if (isReverted(e as Error)) return Object.fromEntries(keys.map((k) => [k, false]));
      throw e;
    }
  }

  async ethLogsSingle(topics: Topics, opts: LogOpts): Promise<Log[]> {
    const req: Record<string, any> = { topics, fromBlock: ethNum(opts.fromBlock || 0) };
    if (opts.toBlock !== undefined) req.toBlock = ethNum(opts.toBlock);
    const res = await this.call('eth_getLogs', req);
    return res.map((i: any) => fixLog(i, opts));
  }

  async ethLogs(topics: Topics, opts: LogOpts = {}): Promise<Log[]> {
    validateLogOpts(opts);
    const fromBlock = opts.fromBlock || 0;
    if (!('limitLogs' in opts)) return this.ethLogsSingle(topics, opts);
    const promises = [];
    for (let i = fromBlock; i <= opts.toBlock; i += opts.limitLogs)
      promises.push(this.ethLogsSingle(topics, { fromBlock: i, toBlock: i + opts.limitLogs }));
    const out = [];
    for (const i of await Promise.all(promises)) out.push(...i);
    return out;
  }
  // NOTE: this is very low-level methods that return parts used for .transfers method,
  // you will need to decode data yourself.
  async tokenTransfers(address: string, opts: LogOpts = {}): Promise<[Log[], Log[]]> {
    if (typeof address !== 'string') throw new Error('tokenTransfers: wrong address');
    validateLogOpts(opts);
    // If we want incoming and outgoing token transfers we need to call both
    return await Promise.all([
      this.ethLogs(ERC_TRANSFER.topics({ from: address, to: null, value: null }), opts), // From
      this.ethLogs(ERC_TRANSFER.topics({ from: null, to: address, value: null }), opts), // To
    ]);
  }
  async wethTransfers(address: string, opts: LogOpts = {}): Promise<[Log[]]> {
    if (typeof address !== 'string') throw new Error('tokenTransfers: wrong address');
    validateLogOpts(opts);
    const depositTopic = WETH_DEPOSIT.topics({ dst: address, wad: null });
    const withdrawTopic = WETH_WITHDRAW.topics({ src: address, wad: null });
    // OR query
    return await Promise.all([
      this.ethLogs([[depositTopic[0], withdrawTopic[0]], depositTopic[1]], opts),
    ]);
  }
  async erc1155Transfers(
    address: string,
    opts: LogOpts = {}
  ): Promise<[Log[], Log[], Log[], Log[]]> {
    if (typeof address !== 'string') throw new Error('tokenTransfers: wrong address');
    validateLogOpts(opts);
    return await Promise.all([
      // Single
      this.ethLogs(
        ERC1155_SINGLE.topics({ operator: null, from: address, to: null, id: null, value: null }),
        opts
      ),
      this.ethLogs(
        ERC1155_SINGLE.topics({ operator: null, from: null, to: address, id: null, value: null }),
        opts
      ),
      // Batch
      this.ethLogs(
        ERC1155_BATCH.topics({ operator: null, from: address, to: null, ids: null, values: null }),
        opts
      ),
      this.ethLogs(
        ERC1155_BATCH.topics({ operator: null, from: null, to: address, ids: null, values: null }),
        opts
      ),
    ]);
  }

  async txInfo(
    txHash: string,
    opts: TxInfoOpts = {}
  ): Promise<{
    type: 'legacy' | 'eip2930' | 'eip1559' | 'eip4844' | 'eip7702';
    info: any;
    receipt: any;
    raw: string | undefined;
  }> {
    let [info, receipt] = await Promise.all([
      this.call('eth_getTransactionByHash', txHash),
      this.call('eth_getTransactionReceipt', txHash),
    ]);
    info = fixTxInfo(info);
    receipt = fixTxReceipt(receipt);
    const type = Object.keys(TxVersions)[info.type] as keyof typeof TxVersions;
    // This is not strictly neccessary, but allows to store tx info in very compact format and remove unneccessary fields
    // Also, there is additional validation that node returned actual with correct hash/sender and not corrupted stuff.
    let raw: string | undefined = undefined;
    try {
      const rawData: Record<string, any> = {
        nonce: info.nonce,
        gasLimit: info.gas,
        to: info.to === null ? '0x' : info.to,
        value: info.value,
        data: info.input,
        r: info.r,
        s: info.s,
        yParity: Number(info.v),
        chainId: info.chainId,
      };
      if (info.accessList) rawData.accessList = info.accessList;
      if (info.maxFeePerBlobGas) rawData.maxFeePerBlobGas = info.maxFeePerBlobGas;
      if (info.blobVersionedHashes) rawData.blobVersionedHashes = info.blobVersionedHashes;
      if (info.maxFeePerGas) {
        rawData.maxFeePerGas = info.maxFeePerGas;
        rawData.maxPriorityFeePerGas = info.maxPriorityFeePerGas;
      } else if (info.gasPrice) rawData.gasPrice = info.gasPrice;
      if (type === 'legacy')
        Object.assign(rawData, legacySig.encode({ v: info.v, r: info.r, s: info.s }));
      const tx = new Transaction(type, rawData as any, false, true);
      if (tx.recoverSender().address.toLowerCase() !== info.from.toLowerCase())
        throw new Error('txInfo: wrong sender');
      if (receipt.transactionHash !== `0x${tx.hash}`) throw new Error('txInfo: wrong hash');
      raw = tx.toHex();
    } catch (err) {
      // This can crash if something wrong with our parser or limits, so
      // we have option to make network code to work even if rebuilding is crashed
      if (!opts.ignoreTxRebuildErrors) throw err;
    }
    if (opts.blockCallback && info.blockNumber !== null) opts.blockCallback(info.blockNumber);
    return { type, info, receipt, raw };
  }
  async tokenInfo(contract: string): Promise<TokenInfo | TokenError> {
    const c = createContract(ERC20, this, contract);
    const t = await wait({
      code: this.call('eth_getCode', contract, 'latest'),
      capabilities: this.contractCapabilities(contract),
      // We call all stuff at same time to reduce latency (should be done in single req if batched)
      name: c.name.call(), // ERC-20 (optional), ERC-721 (metada)
      symbol: c.symbol.call(), // ERC-20 (optional), ERC-721 (metadata)
      decimals: c.decimals.call(), // ERC-20 (optional), ERC-721 (enumarable)
      totalSupply: c.totalSupply.call(), // ERC-20 (required), ERC-721
    });
    // No code, probably self-destructed
    if (t.code === '0x') return { contract, error: 'not contract or destructed' };
    if (t.capabilities && t.capabilities.erc1155) {
      // All metadata is inside URI per tokenId to outside network stuff (maybe ipfs), so nothing to do here.
      return { contract, abi: 'ERC1155' };
    }
    if (t.capabilities && t.capabilities.erc721) {
      const res = { contract, abi: 'ERC721' };
      if (t.capabilities.erc721_metadata) {
        if (t.name === undefined) return { contract, error: 'ERC721+Metadata without name' };
        if (t.symbol === undefined) return { contract, error: 'ERC721+Metadata without symbol' };
        Object.assign(res, { name: t.name, symbol: t.symbol, metadata: true });
      }
      if (t.capabilities.erc721_enumerable) {
        if (t.totalSupply === undefined)
          return { contract, error: 'ERC721+Enumerable without totalSupply' };
        Object.assign(res, { totalSupply: t.totalSupply, enumerable: true });
      }
      return res as TokenInfo;
    }
    if (t.totalSupply === undefined) return { contract, error: 'not ERC20 token' }; // If there is no totalSupply, it is not ERC20!
    return {
      contract,
      abi: 'ERC20',
      name: t.name,
      symbol: t.symbol,
      totalSupply: t.totalSupply,
      decimals: t.decimals ? Number(t.decimals) : undefined,
    };
  }

  private async tokenBalanceSingle(
    address: string,
    token: TokenInfo | TokenError,
    tokenIds?: Set<bigint>
  ): Promise<TokenBalanceSingle | TokenError> {
    if ('error' in token) return token;
    if (token.abi === 'ERC20') {
      const balance = await createContract(ERC20, this, token.contract).balanceOf.call(address);
      if (tokenIds && (tokenIds.size > 1 || Array.from(tokenIds)[0] !== 1n)) {
        return { contract: token.contract, error: 'unexpected tokenIds for ERC20' };
      }
      return new Map([[1n, balance]]);
    } else if (token.abi === 'ERC721') {
      const c = createContract(ERC721, this, token.contract);
      const balance = await c.balanceOf.call(address);
      if (!token.enumerable) {
        if (!tokenIds) {
          if (!balance) return new Map(); // no tokens owned by user
          return {
            contract: token.contract,
            error: 'erc721 contract not enumerable, but owner has ' + balance + ' tokens',
          };
        }
        // if we cannot enumerate, but has tokenIds, we can check if tokenIds still owned by account
        const ids = Array.from(tokenIds);
        const owners = await Promise.all(ids.map((i) => c.ownerOf.call(i)));
        return new Map(
          ids.map((i, j) => [i, owners[j].toLowerCase() === address.toLowerCase() ? 1n : 0n])
        );
      }
      // if we can fetch tokenIds: always do this
      const p = [];
      for (let i = 0; i < balance; i++)
        p.push(c.tokenOfOwnerByIndex.call({ owner: address, index: BigInt(i) }));
      tokenIds = new Set(await Promise.all(p));
      const ids = Array.from(tokenIds!);
      return new Map(ids.map((i) => [i, 1n]));
    } else if (token.abi === 'ERC1155') {
      // This is pretty bad standard, because it doesn't allow enumeration of tokenIds for owner
      if (!tokenIds)
        return { contract: token.contract, error: 'cannot fetch erc1155 without tokenIds' };
      const c = createContract(ERC1155, this, token.contract);
      const ids = Array.from(tokenIds);
      const balances = await c.balanceOfBatch.call({ accounts: ids.map((_) => address), ids });
      const res = new Map(ids.map((i, j) => [i, balances[j]]));
      return res;
    }
    throw new Error('unknown token type');
  }

  async tokenURI(
    token: TokenInfo | TokenError | string,
    tokenId: bigint
  ): Promise<string | TokenError> {
    if (typeof token === 'string') token = await this.tokenInfo(token);
    if ('error' in token) return token;
    if (token.abi === 'ERC721') {
      const c = createContract(ERC721, this, token.contract);
      if (!token.metadata) return { contract: token.contract, error: 'erc721 without metadata' };
      return c.tokenURI.call(tokenId);
    } else if (token.abi === 'ERC1155') {
      const c = createContract(ERC1155, this, token.contract);
      return c.uri.call(tokenId);
    }
    return { contract: token.contract, error: 'not supported token type' };
  }

  async tokenBalances(
    address: string,
    tokens: string[],
    tokenIds?: Record<string, Set<bigint>>
  ): Promise<TokenBalances> {
    // New API requires data from tokenInfo (which is slow and should be cached).
    // But for compat with old API, we do tokenInfo call if contract address (as string) presented
    const _tokens = await Promise.all(
      tokens.map((i) => (typeof i === 'string' ? this.tokenInfo(i) : i))
    );
    const balances = await Promise.all(
      _tokens.map((i) => this.tokenBalanceSingle(address, i, tokenIds && tokenIds[i.contract]))
    );
    return Object.fromEntries(_tokens.map((i, j) => [i.contract, balances[j]])) as any;
  }
  private decodeTokenTransfer(token: TokenInfo, log: Log): TokenTransfer | undefined {
    if ('error' in token) return;
    if (token.abi === 'ERC20') {
      try {
        const decoded = ERC_TRANSFER.decode(log.topics, log.data);
        return {
          ...token,
          contract: log.address,
          to: decoded.to,
          from: decoded.from,
          tokens: new Map([[1n, decoded.value]]),
        };
      } catch (e) {}
      // Weth doesn't issue Transfer event on Deposit/Withdrawal
      // NOTE: we don't filter for WETH_CONTRACT here in case of other contracts with similar API or different networks
      try {
        const decoded = WETH_DEPOSIT.decode(log.topics, log.data);
        return {
          ...token,
          contract: log.address,
          from: log.address,
          to: decoded.dst,
          tokens: new Map([[1n, decoded.wad]]),
        };
      } catch (e) {}
      try {
        const decoded = WETH_WITHDRAW.decode(log.topics, log.data);
        return {
          ...token,
          contract: log.address,
          from: decoded.src,
          to: log.address,
          tokens: new Map([[1n, decoded.wad]]),
        };
      } catch (e) {}
    } else if (token.abi === 'ERC721') {
      try {
        const decoded = ERC721_TRANSFER.decode(log.topics, log.data);
        return {
          ...token,
          from: decoded.from,
          to: decoded.to,
          tokens: new Map([[decoded.tokenId, 1n]]),
        };
      } catch (e) {}
    } else if (token.abi === 'ERC1155') {
      try {
        const decoded = ERC1155_SINGLE.decode(log.topics, log.data);
        return {
          ...token,
          from: decoded.from,
          to: decoded.to,
          tokens: new Map([[decoded.id, decoded.value]]),
        };
      } catch (e) {}
      try {
        const decoded = ERC1155_BATCH.decode(log.topics, log.data);
        return {
          ...token,
          from: decoded.from,
          to: decoded.to,
          tokens: new Map(decoded.ids.map((i, j) => [i, decoded.values[j]])),
        };
      } catch (e) {}
    }
    return; // unknown token type
  }

  // We want to get all transactions related to address, that means:
  // - from or to equals address in tx
  // - any internal tx from or to equals address in tx
  // - any erc20 token transfer which hash address in src or dst
  // - erc721 is exactly same function signature as erc20 (need to detect after getting transactions)
  // - erc1155: from/to + single/batch
  // trace_filter (web3) returns information only for first two cases, most of explorers returns only first case.
  async transfers(address: string, opts: TraceOpts & LogOpts = {}): Promise<TxTransfers[]> {
    const txCache: Record<string, any> = {};
    const blockCache: Record<number, any> = {};
    const tokenCache: Record<string, any> = {};
    const _opts = {
      ...opts,
      txCallback: (txHash: string) => {
        if (txCache[txHash]) return;
        txCache[txHash] = this.txInfo(txHash, opts);
      },
      blockCallback: (blockNumber: number) => {
        if (blockCache[blockNumber]) return;
        blockCache[blockNumber] = this.blockInfo(blockNumber);
      },
      contractCallback: (address: string) => {
        if (tokenCache[address]) return;
        tokenCache[address] = this.tokenInfo(address);
      },
    };
    if (!_opts.fromBlock) _opts.fromBlock = 0;
    // This runs in parallel and executes callbacks
    // Note, we ignore logs and weth, but they will call callbacks and fetch related
    const [actions, _logs, _weth] = await Promise.all([
      this.internalTransactions(address, _opts),
      this.tokenTransfers(address, _opts),
      this.wethTransfers(address, _opts),
      this.erc1155Transfers(address, _opts),
    ]);
    const mapCache = async (cache: Record<any, any>) => {
      const keys = Object.keys(cache);
      const values = await Promise.all(Object.values(cache));
      return Object.fromEntries(values.map((v, i) => [keys[i], v]));
    };
    // it is ok to do this sequentially, since promises already started and probably resolved at this point
    const blocks = await mapCache(blockCache);
    const tx = await mapCache(txCache);
    const tokens = await mapCache(tokenCache);
    const actionPerTx = group(actions, 'transactionHash');

    // Sort transactions by [blockNumber, transactionIndex]
    const _txHashes = Object.entries(tx).map(
      ([k, v]) => [k, v.info.blockNumber, v.info.transactionIndex] as [string, number, number]
    );
    _txHashes.sort((a, b) => (a[1] !== b[1] ? a[1] - b[1] : a[2] - b[2]));
    const txHashes = _txHashes.map((i) => i[0]);
    return txHashes.map((txHash) => {
      const { info, receipt } = tx[txHash] as { info: TxInfo; receipt: TxReceipt };
      const actions = actionPerTx[txHash];
      const block = info.blockNumber !== null ? blocks[info.blockNumber] : undefined;
      const transfers: Transfer[] = [];
      if (actions) {
        for (const a of actions)
          transfers.push({ from: a.action.from, to: a.action.to, value: a.action.value });
      } else {
        // If we have action, it was call to contract and transfer from tx is already added
        transfers.push({ from: info.from, to: info.to, value: info.value });
      }
      // cumulativeGasUsed includes all transactions before that in block, so useless. gasUsed is correct even for internal transactions
      transfers.push({ from: info.from, value: receipt.gasUsed * receipt.effectiveGasPrice });
      // Tokens
      const tokenTransfers: TokenTransfer[] = [];
      for (const log of receipt.logs) {
        const tokenInfo = tokens[log.address];
        if (!tokenInfo) continue;
        const tt = this.decodeTokenTransfer(tokenInfo, log);
        if (tt) tokenTransfers.push(tt);
      }
      return {
        hash: txHash,
        timestamp: block.timestamp,
        block: info.blockNumber !== null ? info.blockNumber : undefined,
        reverted: !receipt.status,
        transfers,
        tokenTransfers,
        info: { ...tx[txHash], block, actions },
      };
    }) as TxTransfers[];
  }

  async allowances(address: string, opts: LogOpts = {}): Promise<TxAllowances> {
    const approval = events(ERC20).Approval;
    // ERC-721/ERC-1155: +ApprovalForAll
    // ERC-1761 Scoped Approval for partial with 1155/721?
    const topics = approval.topics({ owner: address, spender: null, value: null });
    const logs = await this.ethLogs(topics, opts);
    // res[tokenContract][spender] = value
    const res: TxAllowances = {};
    for (const l of logs) {
      const decoded = approval.decode(l.topics, l.data);
      if (decoded.owner.toLowerCase() !== address.toLowerCase()) continue;
      if (!res[l.address]) res[l.address] = {};
      res[l.address][decoded.spender] = decoded.value;
    }
    return res;
  }
}

/**
 * Calculates balances at specific point in time after tx.
 * Also, useful as a sanity check in case we've missed something.
 * Info from multiple addresses can be merged (sort everything first).
 */
export function calcTransfersDiff(transfers: TxTransfers[]): (TxTransfers & Balances)[] {
  // address -> balance
  const balances: Record<string, bigint> = {};
  // contract -> address -> tokenId -> balance
  const tokenBalances: Record<string, Record<string, Map<bigint, bigint>>> = {};
  let _0 = BigInt(0);
  for (const t of transfers) {
    for (const it of t.transfers) {
      if (it.from) {
        if (balances[it.from] === undefined) balances[it.from] = _0;
        balances[it.from] -= it.value;
      }
      if (it.to) {
        if (balances[it.to] === undefined) balances[it.to] = _0;
        balances[it.to] += it.value;
      }
    }
    for (const tt of t.tokenTransfers) {
      if (!tokenBalances[tt.contract]) tokenBalances[tt.contract] = {};
      const token = tokenBalances[tt.contract];
      for (const [tokenId, value] of tt.tokens) {
        if (token[tt.from] === undefined) token[tt.from] = new Map();
        if (token[tt.to] === undefined) token[tt.to] = new Map();
        const fromTokens = token[tt.from];
        const toTokens = token[tt.to];
        fromTokens.set(tokenId, (fromTokens.get(tokenId) || _0) - value);
        toTokens.set(tokenId, (toTokens.get(tokenId) || _0) + value);
      }
    }
    Object.assign(t, {
      balances: { ...balances },
      // deep copy
      tokenBalances: Object.fromEntries(
        Object.entries(tokenBalances).map(([k, v]) => [k, { ...v }])
      ),
    });
  }
  return transfers as (TxTransfers & Balances)[];
}
