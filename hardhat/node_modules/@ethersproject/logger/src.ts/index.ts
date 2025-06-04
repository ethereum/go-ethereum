"use strict";

let _permanentCensorErrors = false;
let _censorErrors = false;

const LogLevels: { [ name: string ]: number } = { debug: 1, "default": 2, info: 2, warning: 3, error: 4, off: 5 };
let _logLevel = LogLevels["default"];

import { version } from "./_version";

let _globalLogger: Logger = null;

function _checkNormalize(): string {
    try {
        const missing: Array<string> = [ ];

        // Make sure all forms of normalization are supported
        ["NFD", "NFC", "NFKD", "NFKC"].forEach((form) => {
            try {
                if ("test".normalize(form) !== "test") {
                    throw new Error("bad normalize");
                };
            } catch(error) {
                missing.push(form);
            }
        });

        if (missing.length) {
            throw new Error("missing " + missing.join(", "));
        }

        if (String.fromCharCode(0xe9).normalize("NFD") !== String.fromCharCode(0x65, 0x0301)) {
            throw new Error("broken implementation")
        }
    } catch (error) {
        return error.message;
    }

    return null;
}

const _normalizeError = _checkNormalize();

export enum LogLevel {
    DEBUG    = "DEBUG",
    INFO     = "INFO",
    WARNING  = "WARNING",
    ERROR    = "ERROR",
    OFF      = "OFF"
}


export enum ErrorCode {

    ///////////////////
    // Generic Errors

    // Unknown Error
    UNKNOWN_ERROR = "UNKNOWN_ERROR",

    // Not Implemented
    NOT_IMPLEMENTED = "NOT_IMPLEMENTED",

    // Unsupported Operation
    //   - operation
    UNSUPPORTED_OPERATION = "UNSUPPORTED_OPERATION",

    // Network Error (i.e. Ethereum Network, such as an invalid chain ID)
    //   - event ("noNetwork" is not re-thrown in provider.ready; otherwise thrown)
    NETWORK_ERROR = "NETWORK_ERROR",

    // Some sort of bad response from the server
    SERVER_ERROR = "SERVER_ERROR",

    // Timeout
    TIMEOUT = "TIMEOUT",

    ///////////////////
    // Operational  Errors

    // Buffer Overrun
    BUFFER_OVERRUN = "BUFFER_OVERRUN",

    // Numeric Fault
    //   - operation: the operation being executed
    //   - fault: the reason this faulted
    NUMERIC_FAULT = "NUMERIC_FAULT",


    ///////////////////
    // Argument Errors

    // Missing new operator to an object
    //  - name: The name of the class
    MISSING_NEW = "MISSING_NEW",

    // Invalid argument (e.g. value is incompatible with type) to a function:
    //   - argument: The argument name that was invalid
    //   - value: The value of the argument
    INVALID_ARGUMENT = "INVALID_ARGUMENT",

    // Missing argument to a function:
    //   - count: The number of arguments received
    //   - expectedCount: The number of arguments expected
    MISSING_ARGUMENT = "MISSING_ARGUMENT",

    // Too many arguments
    //   - count: The number of arguments received
    //   - expectedCount: The number of arguments expected
    UNEXPECTED_ARGUMENT = "UNEXPECTED_ARGUMENT",


    ///////////////////
    // Blockchain Errors

    // Call exception
    //  - transaction: the transaction
    //  - address?: the contract address
    //  - args?: The arguments passed into the function
    //  - method?: The Solidity method signature
    //  - errorSignature?: The EIP848 error signature
    //  - errorArgs?: The EIP848 error parameters
    //  - reason: The reason (only for EIP848 "Error(string)")
    CALL_EXCEPTION = "CALL_EXCEPTION",

    // Insufficient funds (< value + gasLimit * gasPrice)
    //   - transaction: the transaction attempted
    INSUFFICIENT_FUNDS = "INSUFFICIENT_FUNDS",

    // Nonce has already been used
    //   - transaction: the transaction attempted
    NONCE_EXPIRED = "NONCE_EXPIRED",

    // The replacement fee for the transaction is too low
    //   - transaction: the transaction attempted
    REPLACEMENT_UNDERPRICED = "REPLACEMENT_UNDERPRICED",

    // The gas limit could not be estimated
    //   - transaction: the transaction passed to estimateGas
    UNPREDICTABLE_GAS_LIMIT = "UNPREDICTABLE_GAS_LIMIT",

    // The transaction was replaced by one with a higher gas price
    //   - reason: "cancelled", "replaced" or "repriced"
    //   - cancelled: true if reason == "cancelled" or reason == "replaced")
    //   - hash: original transaction hash
    //   - replacement: the full TransactionsResponse for the replacement
    //   - receipt: the receipt of the replacement
    TRANSACTION_REPLACED = "TRANSACTION_REPLACED",


    ///////////////////
    // Interaction Errors

    // The user rejected the action, such as signing a message or sending
    // a transaction
    ACTION_REJECTED = "ACTION_REJECTED",
};

const HEX = "0123456789abcdef";

export class Logger {
    readonly version: string;

    static errors = ErrorCode;

    static levels = LogLevel;

    constructor(version: string) {
        Object.defineProperty(this, "version", {
            enumerable: true,
            value: version,
            writable: false
        });
    }

    _log(logLevel: LogLevel, args: Array<any>): void {
        const level = logLevel.toLowerCase();
        if (LogLevels[level] == null) {
            this.throwArgumentError("invalid log level name", "logLevel", logLevel);
        }
        if (_logLevel > LogLevels[level]) { return; }
        console.log.apply(console, args);
    }

    debug(...args: Array<any>): void {
        this._log(Logger.levels.DEBUG, args);
    }

    info(...args: Array<any>): void {
        this._log(Logger.levels.INFO, args);
    }

    warn(...args: Array<any>): void {
        this._log(Logger.levels.WARNING, args);
    }

    makeError(message: string, code?: ErrorCode, params?: any): Error {
        // Errors are being censored
        if (_censorErrors) {
            return this.makeError("censored error", code, { });
        }

        if (!code) { code = Logger.errors.UNKNOWN_ERROR; }
        if (!params) { params = {}; }

        const messageDetails: Array<string> = [];
        Object.keys(params).forEach((key) => {
            const value = params[key];
            try {
                if (value instanceof Uint8Array) {
                    let hex = "";
                    for (let i = 0; i < value.length; i++) {
                      hex += HEX[value[i] >> 4];
                      hex += HEX[value[i] & 0x0f];
                    }
                    messageDetails.push(key + "=Uint8Array(0x" + hex + ")");
                } else {
                    messageDetails.push(key + "=" + JSON.stringify(value));
                }
            } catch (error) {
                messageDetails.push(key + "=" + JSON.stringify(params[key].toString()));
            }
        });
        messageDetails.push(`code=${ code }`);
        messageDetails.push(`version=${ this.version }`);

        const reason = message;

        let url = "";

        switch (code) {
            case ErrorCode.NUMERIC_FAULT: {
                url = "NUMERIC_FAULT";
                const fault = message;

                switch (fault) {
                    case "overflow": case "underflow": case "division-by-zero":
                        url += "-" + fault;
                        break;
                    case "negative-power": case "negative-width":
                        url += "-unsupported";
                        break;
                    case "unbound-bitwise-result":
                        url += "-unbound-result";
                        break;
                }
                break;
            }
            case ErrorCode.CALL_EXCEPTION:
            case ErrorCode.INSUFFICIENT_FUNDS:
            case ErrorCode.MISSING_NEW:
            case ErrorCode.NONCE_EXPIRED:
            case ErrorCode.REPLACEMENT_UNDERPRICED:
            case ErrorCode.TRANSACTION_REPLACED:
            case ErrorCode.UNPREDICTABLE_GAS_LIMIT:
                url = code;
                break;
        }

        if (url) {
            message += " [ See: https:/\/links.ethers.org/v5-errors-" + url + " ]";
        }

        if (messageDetails.length) {
            message += " (" + messageDetails.join(", ") + ")";
        }

        // @TODO: Any??
        const error: any = new Error(message);
        error.reason = reason;
        error.code = code

        Object.keys(params).forEach(function(key) {
            error[key] = params[key];
        });

        return error;
    }

    throwError(message: string, code?: ErrorCode, params?: any): never {
        throw this.makeError(message, code, params);
    }

    throwArgumentError(message: string, name: string, value: any): never {
        return this.throwError(message, Logger.errors.INVALID_ARGUMENT, {
            argument: name,
            value: value
        });
    }

    assert(condition: any, message: string, code?: ErrorCode, params?: any): void {
        if (!!condition) { return; }
        this.throwError(message, code, params);
    }

    assertArgument(condition: any, message: string, name: string, value: any): void {
        if (!!condition) { return; }
        this.throwArgumentError(message, name, value);
    }

    checkNormalize(message?: string): void {
        if (message == null) { message = "platform missing String.prototype.normalize"; }
        if (_normalizeError) {
            this.throwError("platform missing String.prototype.normalize", Logger.errors.UNSUPPORTED_OPERATION, {
                operation: "String.prototype.normalize", form: _normalizeError
            });
        }
    }

    checkSafeUint53(value: number, message?: string): void {
        if (typeof(value) !== "number") { return; }

        if (message == null) { message = "value not safe"; }

        if (value < 0 || value >= 0x1fffffffffffff) {
            this.throwError(message, Logger.errors.NUMERIC_FAULT, {
                operation: "checkSafeInteger",
                fault: "out-of-safe-range",
                value: value
            });
        }

        if (value % 1) {
            this.throwError(message, Logger.errors.NUMERIC_FAULT, {
                operation: "checkSafeInteger",
                fault: "non-integer",
                value: value
            });
        }
    }

    checkArgumentCount(count: number, expectedCount: number, message?: string): void {
        if (message) {
            message = ": " + message;
        } else {
            message = "";
        }

        if (count < expectedCount) {
            this.throwError("missing argument" + message, Logger.errors.MISSING_ARGUMENT, {
                count: count,
                expectedCount: expectedCount
            });
        }

        if (count > expectedCount) {
            this.throwError("too many arguments" + message, Logger.errors.UNEXPECTED_ARGUMENT, {
                count: count,
                expectedCount: expectedCount
            });
        }
    }

    checkNew(target: any, kind: any): void {
        if (target === Object || target == null) {
            this.throwError("missing new", Logger.errors.MISSING_NEW, { name: kind.name });
        }
    }

    checkAbstract(target: any, kind: any): void {
        if (target === kind) {
            this.throwError(
                "cannot instantiate abstract class " + JSON.stringify(kind.name) + " directly; use a sub-class",
                Logger.errors.UNSUPPORTED_OPERATION,
                { name: target.name, operation: "new" }
            );
        } else if (target === Object || target == null) {
            this.throwError("missing new", Logger.errors.MISSING_NEW, { name: kind.name });
        }
    }

    static globalLogger(): Logger {
        if (!_globalLogger) { _globalLogger = new Logger(version); }
        return _globalLogger;
    }

    static setCensorship(censorship: boolean, permanent?: boolean): void {
        if (!censorship && permanent) {
            this.globalLogger().throwError("cannot permanently disable censorship", Logger.errors.UNSUPPORTED_OPERATION, {
                operation: "setCensorship"
            });
        }

        if (_permanentCensorErrors) {
            if (!censorship) { return; }
            this.globalLogger().throwError("error censorship permanent", Logger.errors.UNSUPPORTED_OPERATION, {
                operation: "setCensorship"
            });
        }

        _censorErrors = !!censorship;
        _permanentCensorErrors = !!permanent;
    }

    static setLogLevel(logLevel: LogLevel): void {
        const level = LogLevels[logLevel.toLowerCase()];
        if (level == null) {
            Logger.globalLogger().warn("invalid log level - " + logLevel);
            return;
        }
        _logLevel = level;
    }

    static from(version: string): Logger {
        return new Logger(version);
    }
}
