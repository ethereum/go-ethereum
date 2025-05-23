import { AccessList, AccessListEntry, BaseTransactionAPI, Transaction1559UnsignedAPI, Transaction2930UnsignedAPI, TransactionCall, TransactionLegacyUnsignedAPI, TransactionWithSenderAPI } from 'web3-types';
import { CustomTransactionSchema, InternalTransaction } from './types.js';
export declare function isBaseTransaction(value: BaseTransactionAPI): boolean;
export declare function isAccessListEntry(value: AccessListEntry): boolean;
export declare function isAccessList(value: AccessList): boolean;
export declare function isTransaction1559Unsigned(value: Transaction1559UnsignedAPI): boolean;
export declare function isTransaction2930Unsigned(value: Transaction2930UnsignedAPI): boolean;
export declare function isTransactionLegacyUnsigned(value: TransactionLegacyUnsignedAPI): boolean;
export declare function isTransactionWithSender(value: TransactionWithSenderAPI): boolean;
export declare function validateTransactionWithSender(value: TransactionWithSenderAPI): void;
export declare function isTransactionCall(value: TransactionCall): boolean;
export declare function validateTransactionCall(value: TransactionCall): void;
export declare const validateCustomChainInfo: (transaction: InternalTransaction) => void;
export declare const validateChainInfo: (transaction: InternalTransaction) => void;
export declare const validateBaseChain: (transaction: InternalTransaction) => void;
export declare const validateHardfork: (transaction: InternalTransaction) => void;
export declare const validateLegacyGas: (transaction: InternalTransaction) => void;
export declare const validateFeeMarketGas: (transaction: InternalTransaction) => void;
/**
 * This method checks if all required gas properties are present for either
 * legacy gas (type 0x0 and 0x1) OR fee market transactions (0x2)
 */
export declare const validateGas: (transaction: InternalTransaction) => void;
export declare const validateTransactionForSigning: (transaction: InternalTransaction, overrideMethod?: (transaction: InternalTransaction) => void, options?: {
    transactionSchema?: CustomTransactionSchema;
}) => void;
