import { ContractExecutionError, TransactionRevertedWithoutReasonError, TransactionRevertInstructionError, TransactionRevertWithCustomError, InvalidResponseError, TransactionPollingTimeoutError } from 'web3-errors';
import { FormatType, ETH_DATA_FORMAT, DataFormat, Bytes, ContractAbi, HexString, Numbers, Transaction, TransactionReceipt, TransactionWithFromAndToLocalWalletIndex, TransactionWithFromLocalWalletIndex, TransactionWithToLocalWalletIndex } from 'web3-types';
import { Schema } from 'web3-validator';
export type InternalTransaction = FormatType<Transaction, typeof ETH_DATA_FORMAT>;
export type SendTransactionEventsBase<ReturnFormat extends DataFormat, TxType> = {
    sending: FormatType<TxType, typeof ETH_DATA_FORMAT>;
    sent: FormatType<TxType, typeof ETH_DATA_FORMAT>;
    transactionHash: FormatType<Bytes, ReturnFormat>;
    receipt: FormatType<TransactionReceipt, ReturnFormat>;
    confirmation: {
        confirmations: FormatType<Numbers, ReturnFormat>;
        receipt: FormatType<TransactionReceipt, ReturnFormat>;
        latestBlockHash: FormatType<Bytes, ReturnFormat>;
    };
    error: TransactionRevertedWithoutReasonError<FormatType<TransactionReceipt, ReturnFormat>> | TransactionRevertInstructionError<FormatType<TransactionReceipt, ReturnFormat>> | TransactionRevertWithCustomError<FormatType<TransactionReceipt, ReturnFormat>> | TransactionPollingTimeoutError | InvalidResponseError | ContractExecutionError;
};
export type SendTransactionEvents<ReturnFormat extends DataFormat> = SendTransactionEventsBase<ReturnFormat, Transaction>;
export type SendSignedTransactionEvents<ReturnFormat extends DataFormat> = SendTransactionEventsBase<ReturnFormat, Bytes>;
export interface SendTransactionOptions<ResolveType = TransactionReceipt> {
    ignoreGasPricing?: boolean;
    transactionResolver?: (receipt: TransactionReceipt) => ResolveType;
    contractAbi?: ContractAbi;
    checkRevertBeforeSending?: boolean;
    ignoreFillingGasLimit?: boolean;
}
export interface SendSignedTransactionOptions<ResolveType = TransactionReceipt> {
    transactionResolver?: (receipt: TransactionReceipt) => ResolveType;
    contractAbi?: ContractAbi;
    checkRevertBeforeSending?: boolean;
}
export interface RevertReason {
    reason: string;
    signature?: HexString;
    data?: HexString;
}
export interface RevertReasonWithCustomError extends RevertReason {
    customErrorName: string;
    customErrorDecodedSignature: string;
    customErrorArguments: Record<string, unknown>;
}
export type TransactionMiddlewareData = Transaction | TransactionWithFromLocalWalletIndex | TransactionWithToLocalWalletIndex | TransactionWithFromAndToLocalWalletIndex;
export interface TransactionMiddleware {
    processTransaction(transaction: TransactionMiddlewareData, options?: {
        [key: string]: unknown;
    }): Promise<TransactionMiddlewareData>;
}
export type CustomTransactionSchema = {
    type: string;
    properties: Record<string, Schema>;
};
