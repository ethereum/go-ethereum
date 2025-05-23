import { Bytes, HexString, Numbers } from './primitives_types.js';
export type ValueTypes = 'address' | 'bool' | 'string' | 'int256' | 'uint256' | 'bytes' | 'bigint';
export type HexString32Bytes = HexString;
export type HexString16Bytes = HexString;
export type HexString8Bytes = HexString;
export type HexStringSingleByte = HexString;
export type HexStringBytes = HexString;
export type HexString256Bytes = HexString;
export type Uint = HexString;
export type Uint256 = HexString;
export type Address = HexString;
export type Topic = HexString32Bytes;
export type TransactionHash = HexString;
export type Uncles = HexString32Bytes[];
export declare enum BlockTags {
    EARLIEST = "earliest",
    LATEST = "latest",
    PENDING = "pending",
    SAFE = "safe",
    FINALIZED = "finalized",
    COMMITTED = "committed"
}
export type BlockTag = `${BlockTags}`;
export type BlockNumberOrTag = Numbers | BlockTag;
export interface Proof {
    readonly address: HexString;
    readonly nonce: string;
    readonly balance: string;
}
export interface TransactionInput {
    readonly [key: string]: unknown;
    readonly to?: HexString;
    readonly from?: HexString;
    readonly data?: string;
    readonly input?: string;
    readonly gas: HexString;
    readonly gasLimit?: string;
    readonly gasPrice?: string;
    readonly maxPriorityFeePerGas?: string;
    readonly maxFeePerGas?: string;
    readonly nonce: string;
    readonly value: string;
    readonly blockNumber?: HexString;
    readonly transactionIndex?: HexString;
    readonly type?: HexString;
    readonly chainId?: HexString;
}
export type TransactionOutput = {
    readonly [key: string]: unknown;
    readonly to?: HexString;
    readonly from?: HexString;
    readonly input: string;
    readonly gas?: Numbers;
    readonly gasLimit?: string;
    readonly nonce: Numbers;
    readonly value: Numbers;
    readonly blockNumber?: Numbers;
    readonly transactionIndex?: Numbers;
} & ({
    maxPriorityFeePerGas: Numbers;
    maxFeePerGas: Numbers;
    gasPrice?: never;
} | {
    maxPriorityFeePerGas?: never;
    maxFeePerGas?: never;
    gasPrice: Numbers;
});
export interface LogsInput {
    readonly blockHash?: HexString;
    readonly transactionHash?: HexString;
    readonly logIndex?: HexString;
    readonly id?: string;
    readonly blockNumber?: HexString;
    readonly transactionIndex?: HexString;
    readonly address: HexString;
    readonly topics: HexString[];
    readonly data: HexString;
}
export interface LogsOutput {
    readonly id?: string;
    readonly removed: boolean;
    readonly logIndex?: Numbers;
    readonly transactionIndex?: Numbers;
    readonly transactionHash?: HexString32Bytes;
    readonly blockHash?: HexString32Bytes;
    readonly blockNumber?: Numbers;
    readonly address: string;
    readonly topics: HexString[];
    readonly data: HexString;
}
export interface BlockInput {
    readonly gasLimit: HexString;
    readonly gasUsed: HexString;
    readonly size: HexString;
    readonly timestamp: HexString;
    readonly number?: HexString;
    readonly difficulty?: HexString;
    readonly totalDifficulty?: HexString;
    readonly transactions?: TransactionInput[];
    readonly miner?: HexString;
    readonly baseFeePerGas?: HexString;
}
export interface Withdrawals {
    readonly index: Numbers;
    readonly validatorIndex: Numbers;
    readonly address: Address;
    readonly amount: Numbers;
}
export interface BlockOutput {
    readonly gasLimit: bigint | number;
    readonly gasUsed: bigint | number;
    readonly size: bigint | number;
    readonly timestamp: bigint | number;
    readonly number?: bigint | number;
    readonly difficulty?: bigint | number;
    readonly totalDifficulty?: bigint | number;
    readonly transactions?: TransactionOutput[];
    readonly miner?: HexString;
    readonly baseFeePerGas?: bigint | number;
    readonly parentHash?: HexString32Bytes;
    readonly blobGasUsed?: bigint | number;
    readonly excessBlobGas?: bigint | number;
    readonly extraData?: Bytes;
    readonly hash?: HexString32Bytes;
    readonly logsBloom?: Bytes;
    readonly nonce?: bigint | number;
    readonly parentBeaconBlockRoot?: HexString32Bytes;
    readonly receiptsRoot?: HexString32Bytes;
    readonly sha3Uncles: HexString32Bytes[];
    readonly stateRoot?: HexString32Bytes;
    readonly transactionsRoot?: HexString32Bytes;
    readonly withdrawalsRoot?: HexString32Bytes;
    readonly mixHash?: HexString32Bytes;
    readonly uncles?: Uncles;
    readonly withdrawals?: Withdrawals[];
}
export interface BlockHeaderOutput {
    readonly baseFeePerGas?: Numbers;
    readonly blobGasUsed?: Numbers;
    readonly difficulty?: Numbers;
    readonly excessBlobGas?: Numbers;
    readonly extraData?: Bytes;
    readonly gasLimit: Numbers;
    readonly gasUsed: Numbers;
    readonly hash?: HexString32Bytes;
    readonly logsBloom?: Bytes;
    readonly miner?: HexString;
    readonly nonce?: Numbers;
    readonly number?: Numbers;
    readonly parentBeaconBlockRoot?: HexString32Bytes;
    readonly parentHash?: HexString32Bytes;
    readonly receiptsRoot?: HexString32Bytes;
    readonly sha3Uncles: HexString32Bytes[];
    readonly stateRoot?: HexString32Bytes;
    readonly timestamp: Numbers;
    readonly transactionsRoot?: HexString32Bytes;
    readonly withdrawalsRoot?: HexString32Bytes;
    readonly author?: Address;
    readonly totalDifficulty?: Numbers;
    readonly size?: Numbers;
    readonly excessDataGas?: Numbers;
    readonly mixHash?: HexString32Bytes;
    readonly transactions?: TransactionOutput[];
    readonly uncles?: Uncles;
    readonly withdrawals?: Withdrawals[];
}
export interface ReceiptInput {
    readonly [x: string]: unknown;
    readonly blockNumber?: HexString;
    readonly transactionIndex?: HexString;
    readonly cumulativeGasUsed: HexString;
    readonly gasUsed: HexString;
    readonly logs?: LogsInput[];
    readonly contractAddress?: HexString;
    readonly status?: string;
    readonly effectiveGasPrice?: HexString;
}
export interface ReceiptOutput {
    readonly blockNumber?: bigint | number;
    readonly transactionIndex?: bigint | number;
    readonly cumulativeGasUsed: bigint | number;
    readonly gasUsed: bigint | number;
    readonly logs?: LogsOutput[];
    readonly contractAddress?: HexString;
    readonly status: boolean;
    readonly effectiveGasPrice?: bigint | number;
}
export interface PostInput {
    readonly ttl?: HexString;
    readonly workToProve?: HexString;
    readonly priority?: HexString;
    readonly expiry?: HexString;
    readonly sent?: HexString;
    readonly workProved?: HexString;
    readonly topics?: HexString[];
}
export interface PostOutput {
    readonly ttl?: bigint | number;
    readonly workToProve?: bigint | number;
    readonly priority?: bigint | number;
    readonly expiry?: bigint | number;
    readonly sent?: bigint | number;
    readonly workProved?: bigint | number;
    readonly topics?: string[];
}
export interface SyncInput {
    readonly startingBlock: HexString;
    readonly currentBlock: HexString;
    readonly highestBlock: HexString;
    readonly knownStates?: HexString;
    readonly pulledStates?: HexString;
}
export interface SyncOutput {
    readonly startingBlock: Numbers;
    readonly currentBlock: Numbers;
    readonly highestBlock: Numbers;
    readonly knownStates?: Numbers;
    readonly pulledStates?: Numbers;
}
export type Receipt = Record<string, unknown>;
type FilterOption = Record<string, Numbers | Numbers[] | boolean | boolean[]>;
export interface Filter {
    readonly fromBlock?: BlockNumberOrTag;
    readonly toBlock?: BlockNumberOrTag;
    readonly address?: Address | Address[];
    readonly blockHash?: Address;
    readonly topics?: (null | Topic | Topic[])[];
    readonly filter?: FilterOption;
}
export type FilterParams = Omit<Filter, 'blockHash' | 'filter'>;
export interface AccessListEntry {
    readonly address?: Address;
    readonly storageKeys?: HexString32Bytes[];
}
export type AccessList = AccessListEntry[];
export type AccessListResult = {
    readonly accessList?: AccessList;
    readonly gasUsed?: Numbers;
};
export type ValidChains = 'goerli' | 'kovan' | 'mainnet' | 'rinkeby' | 'ropsten' | 'sepolia';
export declare enum HardforksOrdered {
    chainstart = "chainstart",
    frontier = "frontier",
    homestead = "homestead",
    dao = "dao",
    tangerineWhistle = "tangerineWhistle",
    spuriousDragon = "spuriousDragon",
    byzantium = "byzantium",
    constantinople = "constantinople",
    petersburg = "petersburg",
    istanbul = "istanbul",
    muirGlacier = "muirGlacier",
    berlin = "berlin",
    london = "london",
    altair = "altair",
    arrowGlacier = "arrowGlacier",
    grayGlacier = "grayGlacier",
    bellatrix = "bellatrix",
    merge = "merge",
    capella = "capella",
    shanghai = "shanghai"
}
export type Hardfork = `${HardforksOrdered}`;
export interface LogBase<NumberType, ByteType> {
    readonly removed?: boolean;
    readonly logIndex?: NumberType;
    readonly transactionIndex?: NumberType;
    readonly transactionHash?: ByteType;
    readonly blockHash?: ByteType;
    readonly blockNumber?: NumberType;
    readonly address?: Address;
    readonly data?: ByteType;
    readonly topics?: ByteType[];
    readonly id?: string;
}
export interface Log extends LogBase<Numbers, Bytes> {
    readonly id?: string;
}
export interface EventLog {
    readonly event: string;
    readonly id?: string;
    readonly logIndex?: bigint | number | string;
    readonly transactionIndex?: bigint | number | string;
    readonly transactionHash?: HexString32Bytes;
    readonly blockHash?: HexString32Bytes;
    readonly blockNumber?: bigint | number | string;
    readonly address: string;
    readonly topics: HexString[];
    readonly data: HexString;
    readonly raw?: {
        data: string;
        topics: unknown[];
    };
    readonly returnValues: Record<string, unknown>;
    readonly signature?: HexString;
}
export interface TransactionReceiptBase<numberType, hashByteType, logsBloomByteType, logsType> {
    readonly transactionHash: hashByteType;
    readonly transactionIndex: numberType;
    readonly blockHash: hashByteType;
    readonly blockNumber: numberType;
    readonly from: Address;
    readonly to: Address;
    readonly cumulativeGasUsed: numberType;
    readonly gasUsed: numberType;
    readonly effectiveGasPrice?: numberType;
    readonly contractAddress?: Address;
    readonly logs: logsType[];
    readonly logsBloom: logsBloomByteType;
    readonly root: hashByteType;
    readonly status: numberType;
    readonly type?: numberType;
    events?: {
        [key: string]: EventLog;
    };
}
export type TransactionReceipt = TransactionReceiptBase<Numbers, Bytes, Bytes, Log>;
export interface CustomChain {
    name?: string;
    networkId: Numbers;
    chainId: Numbers;
}
export interface Common {
    customChain: CustomChain;
    baseChain?: ValidChains;
    hardfork?: Hardfork;
}
interface TransactionBase {
    value?: Numbers;
    accessList?: AccessList;
    common?: Common;
    gas?: Numbers;
    gasPrice?: Numbers;
    type?: Numbers;
    maxFeePerGas?: Numbers;
    maxPriorityFeePerGas?: Numbers;
    data?: Bytes;
    input?: Bytes;
    nonce?: Numbers;
    chain?: ValidChains;
    hardfork?: Hardfork;
    chainId?: Numbers;
    networkId?: Numbers;
    gasLimit?: Numbers;
    yParity?: Uint;
    v?: Numbers;
    r?: Bytes;
    s?: Bytes;
}
export interface Transaction extends TransactionBase {
    from?: Address;
    to?: Address | null;
}
export interface TransactionForAccessList extends Transaction {
    from: Address;
}
export interface TransactionCall extends Transaction {
    to: Address;
}
export interface TransactionWithFromLocalWalletIndex extends Omit<Transaction, 'from'> {
    from: Numbers;
}
export interface TransactionWithToLocalWalletIndex extends Omit<Transaction, 'to'> {
    to: Numbers;
}
export interface TransactionWithFromAndToLocalWalletIndex extends Omit<Transaction, 'from' | 'to'> {
    from: Numbers;
    to: Numbers;
}
export interface TransactionInfo extends Transaction {
    readonly blockHash?: Bytes;
    readonly blockNumber?: Numbers;
    readonly from: Address;
    readonly hash: Bytes;
    readonly transactionIndex?: Numbers;
}
export interface PopulatedUnsignedBaseTransaction {
    from: Address;
    to?: Address;
    value: Numbers;
    gas?: Numbers;
    gasPrice: Numbers;
    type: Numbers;
    input?: Bytes;
    data?: Bytes;
    nonce: Numbers;
    networkId: Numbers;
    chain: ValidChains;
    hardfork: Hardfork;
    chainId: Numbers;
    common: Common;
    gasLimit: Numbers;
}
export interface PopulatedUnsignedEip2930Transaction extends PopulatedUnsignedBaseTransaction {
    accessList: AccessList;
}
export interface PopulatedUnsignedEip1559Transaction extends PopulatedUnsignedEip2930Transaction {
    gasPrice: never;
    maxFeePerGas: Numbers;
    maxPriorityFeePerGas: Numbers;
}
export type PopulatedUnsignedTransaction = PopulatedUnsignedBaseTransaction | PopulatedUnsignedEip2930Transaction | PopulatedUnsignedEip1559Transaction;
export interface BlockBase<ByteType, HexStringType, NumberType, extraDataType, TransactionTypes, logsBloomType> {
    readonly parentHash: ByteType;
    readonly sha3Uncles: ByteType;
    readonly miner: HexStringType;
    readonly stateRoot: ByteType;
    readonly transactionsRoot: ByteType;
    readonly receiptsRoot: ByteType;
    readonly logsBloom?: logsBloomType;
    readonly difficulty?: NumberType;
    readonly number: NumberType;
    readonly gasLimit: NumberType;
    readonly gasUsed: NumberType;
    readonly timestamp: NumberType;
    readonly extraData: extraDataType;
    readonly mixHash: ByteType;
    readonly nonce: NumberType;
    readonly totalDifficulty: NumberType;
    readonly baseFeePerGas?: NumberType;
    readonly size: NumberType;
    readonly transactions: TransactionTypes;
    readonly uncles: Uncles;
    readonly hash?: ByteType;
}
export type Block = BlockBase<Bytes, Bytes, Numbers, Bytes, TransactionHash[] | TransactionInfo[], Bytes>;
export interface FeeHistoryBase<NumberType> {
    readonly oldestBlock: NumberType;
    readonly baseFeePerGas: NumberType[];
    readonly reward: NumberType[][];
    readonly gasUsedRatio: NumberType[];
}
export type FeeHistory = FeeHistoryBase<Numbers>;
export interface StorageProof {
    readonly key: Bytes;
    readonly value: Numbers;
    readonly proof: Bytes[];
}
export interface AccountObject {
    readonly balance: Numbers;
    readonly codeHash: Bytes;
    readonly nonce: Numbers;
    readonly storageHash: Bytes;
    readonly accountProof: Bytes[];
    readonly storageProof: StorageProof[];
}
export interface Eip712TypeDetails {
    name: string;
    type: string;
}
export interface Eip712TypedData {
    readonly types: {
        EIP712Domain: Eip712TypeDetails[];
        [key: string]: Eip712TypeDetails[];
    };
    readonly primaryType: string;
    readonly domain: Record<string, string | number>;
    readonly message: Record<string, unknown>;
}
/**
 * To contain the gas Fee Data to be used with EIP-1559 transactions.
 * EIP-1559 was applied to Ethereum after London hardfork.
 *
 * Typically you will only need `maxFeePerGas` and `maxPriorityFeePerGas` for a transaction following EIP-1559.
 * However, if you want to get informed about the fees of last block, you can use `baseFeePerGas` too.
 *
 *
 * 	@see https://eips.ethereum.org/EIPS/eip-1559
 *
 */
export interface FeeData {
    /**
     * This filed is used for legacy networks that does not support EIP-1559.
     */
    readonly gasPrice?: Numbers;
    /**
     * The baseFeePerGas returned from the last available block.
     *
     * If EIP-1559 is not supported, this will be `undefined`
     *
     * However, the user will only pay (the future baseFeePerGas + the maxPriorityFeePerGas).
     * And this value is just for getting informed about the fees of last block.
     */
    readonly baseFeePerGas?: Numbers;
    /**
     * The maximum fee that the user would be willing to pay per-gas.
     *
     * However, the user will only pay (the future baseFeePerGas + the maxPriorityFeePerGas).
     * And the `maxFeePerGas` could be used to prevent paying more than it, if `baseFeePerGas` went too high.
     *
     * If EIP-1559 is not supported, this will be `undefined`
     */
    readonly maxFeePerGas?: Numbers;
    /**
     * The validator's tip for including a transaction in a block.
     *
     * If EIP-1559 is not supported, this will be `undefined`
     */
    readonly maxPriorityFeePerGas?: Numbers;
}
export {};
//# sourceMappingURL=eth_types.d.ts.map