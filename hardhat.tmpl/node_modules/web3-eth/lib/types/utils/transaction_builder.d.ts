import { EthExecutionAPI, Address, HexString, Transaction, TransactionWithFromLocalWalletIndex, TransactionWithToLocalWalletIndex, TransactionWithFromAndToLocalWalletIndex, Web3NetAPI, DataFormat, FormatType, ETH_DATA_FORMAT } from 'web3-types';
import { Web3Context } from 'web3-core';
export declare const getTransactionFromOrToAttr: (attr: "from" | "to", web3Context: Web3Context<EthExecutionAPI>, transaction?: Transaction | TransactionWithFromLocalWalletIndex | TransactionWithToLocalWalletIndex | TransactionWithFromAndToLocalWalletIndex, privateKey?: HexString | Uint8Array) => Address | undefined;
export declare const getTransactionNonce: <ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, address?: Address, returnFormat?: ReturnFormat) => Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
export declare const getTransactionType: (transaction: FormatType<Transaction, typeof ETH_DATA_FORMAT>, web3Context: Web3Context<EthExecutionAPI>) => string | undefined;
export declare function defaultTransactionBuilder<ReturnType = Transaction>(options: {
    transaction: Transaction;
    web3Context: Web3Context<EthExecutionAPI & Web3NetAPI>;
    privateKey?: HexString | Uint8Array;
    fillGasPrice?: boolean;
    fillGasLimit?: boolean;
}): Promise<ReturnType>;
export declare const transactionBuilder: <ReturnType = Transaction>(options: {
    transaction: Transaction;
    web3Context: Web3Context<EthExecutionAPI>;
    privateKey?: HexString | Uint8Array;
    fillGasPrice?: boolean;
    fillGasLimit?: boolean;
}) => Promise<ReturnType>;
//# sourceMappingURL=transaction_builder.d.ts.map