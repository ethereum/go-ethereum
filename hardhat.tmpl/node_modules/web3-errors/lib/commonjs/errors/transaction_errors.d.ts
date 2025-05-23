import { Bytes, HexString, Numbers, TransactionReceipt, Web3ValidationErrorObject } from 'web3-types';
import { InvalidValueError, BaseWeb3Error } from '../web3_error_base.js';
export declare class TransactionError<ReceiptType = TransactionReceipt> extends BaseWeb3Error {
    receipt?: ReceiptType | undefined;
    code: number;
    constructor(message: string, receipt?: ReceiptType | undefined);
    toJSON(): {
        receipt: ReceiptType | undefined;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class RevertInstructionError extends BaseWeb3Error {
    reason: string;
    signature: string;
    code: number;
    constructor(reason: string, signature: string);
    toJSON(): {
        reason: string;
        signature: string;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class TransactionRevertInstructionError<ReceiptType = TransactionReceipt> extends BaseWeb3Error {
    reason: string;
    signature?: string | undefined;
    receipt?: ReceiptType | undefined;
    data?: string | undefined;
    code: number;
    constructor(reason: string, signature?: string | undefined, receipt?: ReceiptType | undefined, data?: string | undefined);
    toJSON(): {
        reason: string;
        signature: string | undefined;
        receipt: ReceiptType | undefined;
        data: string | undefined;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
/**
 * This error is used when a transaction to a smart contract fails and
 * a custom user error (https://blog.soliditylang.org/2021/04/21/custom-errors/)
 * is able to be parsed from the revert reason
 */
export declare class TransactionRevertWithCustomError<ReceiptType = TransactionReceipt> extends TransactionRevertInstructionError<ReceiptType> {
    reason: string;
    customErrorName: string;
    customErrorDecodedSignature: string;
    customErrorArguments: Record<string, unknown>;
    signature?: string | undefined;
    receipt?: ReceiptType | undefined;
    data?: string | undefined;
    code: number;
    constructor(reason: string, customErrorName: string, customErrorDecodedSignature: string, customErrorArguments: Record<string, unknown>, signature?: string | undefined, receipt?: ReceiptType | undefined, data?: string | undefined);
    toJSON(): {
        reason: string;
        customErrorName: string;
        customErrorDecodedSignature: string;
        customErrorArguments: Record<string, unknown>;
        signature: string | undefined;
        receipt: ReceiptType | undefined;
        data: string | undefined;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class NoContractAddressFoundError extends TransactionError {
    constructor(receipt: TransactionReceipt);
    toJSON(): {
        receipt: TransactionReceipt | undefined;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class ContractCodeNotStoredError extends TransactionError {
    constructor(receipt: TransactionReceipt);
}
export declare class TransactionRevertedWithoutReasonError<ReceiptType = TransactionReceipt> extends TransactionError<ReceiptType> {
    constructor(receipt?: ReceiptType);
}
export declare class TransactionOutOfGasError extends TransactionError {
    constructor(receipt: TransactionReceipt);
}
export declare class UndefinedRawTransactionError extends TransactionError {
    constructor();
}
export declare class TransactionNotFound extends TransactionError {
    constructor();
}
export declare class InvalidTransactionWithSender extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidTransactionWithReceiver extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidTransactionCall extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class MissingCustomChainError extends InvalidValueError {
    code: number;
    constructor();
}
export declare class MissingCustomChainIdError extends InvalidValueError {
    code: number;
    constructor();
}
export declare class ChainIdMismatchError extends InvalidValueError {
    code: number;
    constructor(value: {
        txChainId: unknown;
        customChainId: unknown;
    });
}
export declare class ChainMismatchError extends InvalidValueError {
    code: number;
    constructor(value: {
        txChain: unknown;
        baseChain: unknown;
    });
}
export declare class HardforkMismatchError extends InvalidValueError {
    code: number;
    constructor(value: {
        txHardfork: unknown;
        commonHardfork: unknown;
    });
}
export declare class CommonOrChainAndHardforkError extends InvalidValueError {
    code: number;
    constructor();
}
export declare class MissingChainOrHardforkError extends InvalidValueError {
    code: number;
    constructor(value: {
        chain: string | undefined;
        hardfork: string | undefined;
    });
}
export declare class MissingGasInnerError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class MissingGasError extends InvalidValueError {
    code: number;
    constructor(value: {
        gas: Numbers | undefined;
        gasPrice: Numbers | undefined;
        maxPriorityFeePerGas: Numbers | undefined;
        maxFeePerGas: Numbers | undefined;
    });
}
export declare class TransactionGasMismatchInnerError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class TransactionGasMismatchError extends InvalidValueError {
    code: number;
    constructor(value: {
        gas: Numbers | undefined;
        gasPrice: Numbers | undefined;
        maxPriorityFeePerGas: Numbers | undefined;
        maxFeePerGas: Numbers | undefined;
    });
}
export declare class InvalidGasOrGasPrice extends InvalidValueError {
    code: number;
    constructor(value: {
        gas: Numbers | undefined;
        gasPrice: Numbers | undefined;
    });
}
export declare class InvalidMaxPriorityFeePerGasOrMaxFeePerGas extends InvalidValueError {
    code: number;
    constructor(value: {
        maxPriorityFeePerGas: Numbers | undefined;
        maxFeePerGas: Numbers | undefined;
    });
}
export declare class Eip1559GasPriceError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class UnsupportedFeeMarketError extends InvalidValueError {
    code: number;
    constructor(value: {
        maxPriorityFeePerGas: Numbers | undefined;
        maxFeePerGas: Numbers | undefined;
    });
}
export declare class InvalidTransactionObjectError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class InvalidNonceOrChainIdError extends InvalidValueError {
    code: number;
    constructor(value: {
        nonce: Numbers | undefined;
        chainId: Numbers | undefined;
    });
}
export declare class UnableToPopulateNonceError extends InvalidValueError {
    code: number;
    constructor();
}
export declare class Eip1559NotSupportedError extends InvalidValueError {
    code: number;
    constructor();
}
export declare class UnsupportedTransactionTypeError extends InvalidValueError {
    code: number;
    constructor(value: unknown);
}
export declare class TransactionDataAndInputError extends InvalidValueError {
    code: number;
    constructor(value: {
        data: HexString | undefined;
        input: HexString | undefined;
    });
}
export declare class TransactionSendTimeoutError extends BaseWeb3Error {
    code: number;
    constructor(value: {
        numberOfSeconds: number;
        transactionHash?: Bytes;
    });
}
export declare class TransactionPollingTimeoutError extends BaseWeb3Error {
    code: number;
    constructor(value: {
        numberOfSeconds: number;
        transactionHash: Bytes;
    });
}
export declare class TransactionBlockTimeoutError extends BaseWeb3Error {
    code: number;
    constructor(value: {
        starterBlockNumber: number;
        numberOfBlocks: number;
        transactionHash?: Bytes;
    });
}
export declare class TransactionMissingReceiptOrBlockHashError extends InvalidValueError {
    code: number;
    constructor(value: {
        receipt: TransactionReceipt;
        blockHash: Bytes;
        transactionHash: Bytes;
    });
}
export declare class TransactionReceiptMissingBlockNumberError extends InvalidValueError {
    code: number;
    constructor(value: {
        receipt: TransactionReceipt;
    });
}
export declare class TransactionSigningError extends BaseWeb3Error {
    code: number;
    constructor(errorDetails: string);
}
export declare class LocalWalletNotAvailableError extends InvalidValueError {
    code: number;
    constructor();
}
export declare class InvalidPropertiesForTransactionTypeError extends BaseWeb3Error {
    code: number;
    constructor(validationError: Web3ValidationErrorObject[], txType: '0x0' | '0x1' | '0x2');
}
