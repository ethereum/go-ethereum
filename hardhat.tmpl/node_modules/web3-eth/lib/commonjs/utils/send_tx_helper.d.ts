import { FormatType, DataFormat, EthExecutionAPI, Web3BaseWalletAccount, HexString, TransactionReceipt, Transaction, TransactionCall, TransactionWithFromLocalWalletIndex, TransactionWithToLocalWalletIndex, TransactionWithFromAndToLocalWalletIndex, TransactionHash } from 'web3-types';
import { Web3Context, Web3PromiEvent } from 'web3-core';
import { JsonSchema } from 'web3-validator';
import { SendSignedTransactionEvents, SendTransactionEvents, SendTransactionOptions } from '../types.js';
export declare class SendTxHelper<ReturnFormat extends DataFormat, ResolveType = FormatType<TransactionReceipt, ReturnFormat>, TxType = Transaction | TransactionWithFromLocalWalletIndex | TransactionWithToLocalWalletIndex | TransactionWithFromAndToLocalWalletIndex> {
    private readonly web3Context;
    private readonly promiEvent;
    private readonly options;
    private readonly returnFormat;
    constructor({ options, web3Context, promiEvent, returnFormat, }: {
        web3Context: Web3Context<EthExecutionAPI>;
        options: SendTransactionOptions<ResolveType>;
        promiEvent: Web3PromiEvent<ResolveType, SendSignedTransactionEvents<ReturnFormat> | SendTransactionEvents<ReturnFormat>>;
        returnFormat: ReturnFormat;
    });
    getReceiptWithEvents(data: TransactionReceipt): ResolveType;
    checkRevertBeforeSending(tx: TransactionCall): Promise<void>;
    emitSending(tx: TxType | HexString): void;
    populateGasPrice({ transactionFormatted, transaction, }: {
        transactionFormatted: TxType;
        transaction: TxType;
    }): Promise<TxType>;
    signAndSend({ wallet, tx, }: {
        wallet: Web3BaseWalletAccount | undefined;
        tx: TxType;
    }): Promise<string>;
    emitSent(tx: TxType | HexString): void;
    emitTransactionHash(hash: string & Uint8Array): void;
    emitReceipt(receipt: ResolveType): void;
    handleError({ error, tx }: {
        error: unknown;
        tx: TransactionCall;
    }): Promise<unknown>;
    emitConfirmation({ receipt, transactionHash, customTransactionReceiptSchema, }: {
        receipt: ResolveType;
        transactionHash: TransactionHash;
        customTransactionReceiptSchema?: JsonSchema;
    }): void;
    handleResolve({ receipt, tx }: {
        receipt: ResolveType;
        tx: TransactionCall;
    }): Promise<ResolveType>;
}
