import { Filter, Numbers, Topic, BlockInput, BlockOutput, LogsInput, LogsOutput, Mutable, PostInput, PostOutput, Proof, ReceiptInput, ReceiptOutput, SyncInput, SyncOutput, TransactionInput, TransactionOutput } from 'web3-types';
/**
 * @deprecated Use format function from web3-utils package instead
 * Will format the given storage key array values to hex strings.
 */
export declare const inputStorageKeysFormatter: (keys: Array<string>) => string[];
/**
 * @deprecated Use format function from web3-utils package instead
 * Will format the given proof response from the node.
 */
export declare const outputProofFormatter: (proof: Proof) => Proof;
/**
 * @deprecated Use format function from web3-utils package instead
 * Should the format output to a big number
 */
export declare const outputBigIntegerFormatter: (number: Numbers) => number | bigint;
/**
 * @deprecated Use format function from web3-utils package instead
 * Returns the given block number as hex string or the predefined block number 'latest', 'pending', 'earliest', 'genesis'
 */
export declare const inputBlockNumberFormatter: (blockNumber: Numbers | undefined) => string | undefined;
/**
 * @deprecated Use format function from web3-utils package instead
 * Returns the given block number as hex string or does return the defaultBlock property of the current module
 */
export declare const inputDefaultBlockNumberFormatter: (blockNumber: Numbers | undefined, defaultBlock: Numbers) => string | undefined;
/**
 * @deprecated Use format function from web3-utils package instead
 * @param address
 */
export declare const inputAddressFormatter: (address: string) => string | never;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a transaction and converts all values to HEX
 */
export declare const txInputOptionsFormatter: (options: TransactionInput) => Mutable<TransactionOutput>;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a transaction and converts all values to HEX
 */
export declare const inputCallFormatter: (options: TransactionInput, defaultAccount?: string) => Mutable<TransactionOutput>;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a transaction and converts all values to HEX
 */
export declare const inputTransactionFormatter: (options: TransactionInput, defaultAccount?: string) => Mutable<TransactionOutput>;
/**
 * @deprecated Use format function from web3-utils package instead
 * Hex encodes the data passed to eth_sign and personal_sign
 */
export declare const inputSignFormatter: (data: string) => string;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a transaction to its proper values
 * @function outputTransactionFormatter
 */
export declare const outputTransactionFormatter: (tx: TransactionInput) => TransactionOutput;
/**
 * @deprecated Use format function from web3-utils package instead
 * @param topic
 */
export declare const inputTopicFormatter: (topic: Topic) => Topic | null;
/**
 * @deprecated Use format function from web3-utils package instead
 * @param filter
 */
export declare const inputLogFormatter: (filter: Filter) => Filter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a log
 * @function outputLogFormatter
 */
export declare const outputLogFormatter: (log: Partial<LogsInput>) => LogsOutput;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a transaction receipt to its proper values
 */
export declare const outputTransactionReceiptFormatter: (receipt: ReceiptInput) => ReceiptOutput;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a block to its proper values
 * @function outputBlockFormatter
 */
export declare const outputBlockFormatter: (block: BlockInput) => BlockOutput;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a whisper post and converts all values to HEX
 */
export declare const inputPostFormatter: (post: PostOutput) => PostInput;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a received post message
 * @function outputPostFormatter
 */
export declare const outputPostFormatter: (post: PostInput) => PostOutput;
/**
 * @deprecated Use format function from web3-utils package instead
 */
export declare const outputSyncingFormatter: (result: SyncInput) => SyncOutput;
