import { JsonRpcError, TransactionReceipt, HexString } from 'web3-types';
import { BaseWeb3Error, InvalidValueError } from '../web3_error_base.js';
export declare class Web3ContractError extends BaseWeb3Error {
    code: number;
    receipt?: TransactionReceipt;
    constructor(message: string, receipt?: TransactionReceipt);
}
export declare class ResolverMethodMissingError extends BaseWeb3Error {
    address: string;
    name: string;
    code: number;
    constructor(address: string, name: string);
    toJSON(): {
        address: string;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class ContractMissingABIError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class ContractOnceRequiresCallbackError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class ContractEventDoesNotExistError extends BaseWeb3Error {
    eventName: string;
    code: number;
    constructor(eventName: string);
    toJSON(): {
        eventName: string;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class ContractReservedEventError extends BaseWeb3Error {
    type: string;
    code: number;
    constructor(type: string);
    toJSON(): {
        type: string;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class ContractMissingDeployDataError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class ContractNoAddressDefinedError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class ContractNoFromAddressDefinedError extends BaseWeb3Error {
    code: number;
    constructor();
}
export declare class ContractInstantiationError extends BaseWeb3Error {
    code: number;
}
export type ProviderErrorData = HexString | {
    data: HexString;
} | {
    originalError: {
        data: HexString;
    };
};
/**
 * This class is expected to be set as an `cause` inside ContractExecutionError
 * The properties would be typically decoded from the `data` if it was encoded according to EIP-838
 */
export declare class Eip838ExecutionError extends Web3ContractError {
    readonly name: string;
    code: number;
    data?: HexString;
    errorName?: string;
    errorSignature?: string;
    errorArgs?: {
        [K in string]: unknown;
    };
    cause: Eip838ExecutionError | undefined;
    constructor(error: JsonRpcError<ProviderErrorData> | Eip838ExecutionError);
    setDecodedProperties(errorName: string, errorSignature?: string, errorArgs?: {
        [K in string]: unknown;
    }): void;
    toJSON(): {
        name: string;
        code: number;
        message: string;
        innerError: Eip838ExecutionError | undefined;
        cause: Eip838ExecutionError | undefined;
        data: string;
        errorName?: string;
        errorSignature?: string;
        errorArgs?: { [K in string]: unknown; };
    };
}
/**
 * Used when an error is raised while executing a function inside a smart contract.
 * The data is expected to be encoded according to EIP-848.
 */
export declare class ContractExecutionError extends Web3ContractError {
    cause: Eip838ExecutionError;
    constructor(rpcError: JsonRpcError);
}
export declare class ContractTransactionDataAndInputError extends InvalidValueError {
    code: number;
    constructor(value: {
        data: HexString | undefined;
        input: HexString | undefined;
    });
}
