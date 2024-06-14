"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.consoleLogToString = exports.ConsoleLogger = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const util_1 = __importDefault(require("util"));
const logger_1 = require("./logger");
const message_trace_1 = require("./message-trace");
const CONSOLE_ADDRESS = "0x000000000000000000636F6e736F6c652e6c6f67"; // toHex("console.log")
const REGISTER_SIZE = 32;
class ConsoleLogger {
    constructor() {
        this._consoleLogs = {};
        this._consoleLogs = logger_1.ConsoleLogs;
    }
    getLogMessages(maybeDecodedMessageTrace) {
        return this.getExecutionLogs(maybeDecodedMessageTrace).map(consoleLogToString);
    }
    getExecutionLogs(maybeDecodedMessageTrace) {
        if ((0, message_trace_1.isPrecompileTrace)(maybeDecodedMessageTrace)) {
            return [];
        }
        const logs = [];
        this._collectExecutionLogs(maybeDecodedMessageTrace, logs);
        return logs;
    }
    _collectExecutionLogs(trace, logs) {
        for (const messageTrace of trace.steps) {
            if ((0, message_trace_1.isEvmStep)(messageTrace) || (0, message_trace_1.isPrecompileTrace)(messageTrace)) {
                continue;
            }
            if ((0, message_trace_1.isCallTrace)(messageTrace) &&
                (0, ethereumjs_util_1.bytesToHex)(messageTrace.address) === CONSOLE_ADDRESS.toLowerCase()) {
                const log = this._maybeConsoleLog(Buffer.from(messageTrace.calldata));
                if (log !== undefined) {
                    logs.push(log);
                }
                continue;
            }
            this._collectExecutionLogs(messageTrace, logs);
        }
    }
    /**
     * Temporary code to print console.sol messages that come from EDR
     */
    getDecodedLogs(messages) {
        const logs = [];
        for (const message of messages) {
            const log = this._maybeConsoleLog(message);
            if (log !== undefined) {
                logs.push(consoleLogToString(log));
            }
        }
        return logs;
    }
    _maybeConsoleLog(calldata) {
        const sig = (0, ethereumjs_util_1.bytesToInt)(calldata.slice(0, 4));
        const parameters = calldata.slice(4);
        const types = this._consoleLogs[sig];
        if (types === undefined) {
            return;
        }
        const consoleLogs = this._decode(parameters, types);
        this._replaceNumberFormatSpecifiers(consoleLogs);
        return consoleLogs;
    }
    _replaceNumberFormatSpecifiers(consoleLogs) {
        /**
         * Replace the occurrences of %d and %i with %s. This is necessary because if the arguments passed are numbers,
         * they could be too large to be formatted as a Number or an Integer, so it is safer to use a String.
         * %d and %i are replaced only if there is an odd number of % before the d or i.
         * If there is an even number of % then it is assumed that the % is escaped and should not be replaced.
         * The regex matches a '%d' or an '%i' that has an even number of
         * '%' behind it (including 0). This group of pairs of '%' is captured
         * and preserved, while the '%[di]' is replaced with '%s'.
         * Naively doing (%%)* is not enough; we also have to use the
         * (?<!%) negative look-behind to make this work.
         * The (?:) is just to avoid capturing that inner group.
         */
        if (consoleLogs.length > 0 && typeof consoleLogs[0] === "string") {
            consoleLogs[0] = consoleLogs[0].replace(/((?<!%)(?:%%)*)(%[di])/g, "$1%s");
        }
    }
    _decode(data, types) {
        return types.map((type, i) => {
            const position = i * 32;
            switch (types[i]) {
                case logger_1.Uint256Ty:
                    return (0, ethereumjs_util_1.bytesToBigInt)(data.slice(position, position + REGISTER_SIZE)).toString(10);
                case logger_1.Int256Ty:
                    return (0, ethereumjs_util_1.fromSigned)(data.slice(position, position + REGISTER_SIZE)).toString();
                case logger_1.BoolTy:
                    if (data[position + 31] !== 0) {
                        return "true";
                    }
                    return "false";
                case logger_1.StringTy:
                    const sStart = (0, ethereumjs_util_1.bytesToInt)(data.slice(position, position + REGISTER_SIZE));
                    const sLen = (0, ethereumjs_util_1.bytesToInt)(data.slice(sStart, sStart + REGISTER_SIZE));
                    return data
                        .slice(sStart + REGISTER_SIZE, sStart + REGISTER_SIZE + sLen)
                        .toString();
                case logger_1.AddressTy:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position + 12, position + REGISTER_SIZE));
                case logger_1.BytesTy:
                    const bStart = (0, ethereumjs_util_1.bytesToInt)(data.slice(position, position + REGISTER_SIZE));
                    const bLen = (0, ethereumjs_util_1.bytesToInt)(data.slice(bStart, bStart + REGISTER_SIZE));
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(bStart + REGISTER_SIZE, bStart + REGISTER_SIZE + bLen));
                case logger_1.Bytes1Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 1));
                case logger_1.Bytes2Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 2));
                case logger_1.Bytes3Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 3));
                case logger_1.Bytes4Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 4));
                case logger_1.Bytes5Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 5));
                case logger_1.Bytes6Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 6));
                case logger_1.Bytes7Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 7));
                case logger_1.Bytes8Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 8));
                case logger_1.Bytes9Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 9));
                case logger_1.Bytes10Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 10));
                case logger_1.Bytes11Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 11));
                case logger_1.Bytes12Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 12));
                case logger_1.Bytes13Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 13));
                case logger_1.Bytes14Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 14));
                case logger_1.Bytes15Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 15));
                case logger_1.Bytes16Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 16));
                case logger_1.Bytes17Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 17));
                case logger_1.Bytes18Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 18));
                case logger_1.Bytes19Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 19));
                case logger_1.Bytes20Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 20));
                case logger_1.Bytes21Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 21));
                case logger_1.Bytes22Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 22));
                case logger_1.Bytes23Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 23));
                case logger_1.Bytes24Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 24));
                case logger_1.Bytes25Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 25));
                case logger_1.Bytes26Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 26));
                case logger_1.Bytes27Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 27));
                case logger_1.Bytes28Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 28));
                case logger_1.Bytes29Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 29));
                case logger_1.Bytes30Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 30));
                case logger_1.Bytes31Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 31));
                case logger_1.Bytes32Ty:
                    return (0, ethereumjs_util_1.bytesToHex)(data.slice(position, position + 32));
                default:
                    return "";
            }
        });
    }
}
exports.ConsoleLogger = ConsoleLogger;
function consoleLogToString(log) {
    if (log === undefined) {
        return "";
    }
    // special case for console.log()
    if (log.length === 0) {
        return "";
    }
    return util_1.default.format(log[0], ...log.slice(1));
}
exports.consoleLogToString = consoleLogToString;
//# sourceMappingURL=consoleLogger.js.map