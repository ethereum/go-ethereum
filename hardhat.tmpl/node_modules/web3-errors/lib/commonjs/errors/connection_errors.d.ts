import { ConnectionEvent } from 'web3-types';
import { BaseWeb3Error } from '../web3_error_base.js';
export declare class ConnectionError extends BaseWeb3Error {
    code: number;
    errorCode?: number;
    errorReason?: string;
    constructor(message: string, event?: ConnectionEvent);
    toJSON(): {
        errorCode: number | undefined;
        errorReason: string | undefined;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class InvalidConnectionError extends ConnectionError {
    host: string;
    constructor(host: string, event?: ConnectionEvent);
    toJSON(): {
        host: string;
        errorCode: number | undefined;
        errorReason: string | undefined;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class ConnectionTimeoutError extends ConnectionError {
    duration: number;
    constructor(duration: number);
    toJSON(): {
        duration: number;
        errorCode: number | undefined;
        errorReason: string | undefined;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class ConnectionNotOpenError extends ConnectionError {
    constructor(event?: ConnectionEvent);
}
export declare class ConnectionCloseError extends ConnectionError {
    constructor(event?: ConnectionEvent);
}
export declare class MaxAttemptsReachedOnReconnectingError extends ConnectionError {
    constructor(numberOfAttempts: number);
}
export declare class PendingRequestsOnReconnectingError extends ConnectionError {
    constructor();
}
export declare class RequestAlreadySentError extends ConnectionError {
    constructor(id: number | string);
}
