/// <reference types="node" />
import { MessageTrace } from "./message-trace";
interface ConsoleLogArray extends Array<ConsoleLogEntry> {
}
export type ConsoleLogEntry = string | ConsoleLogArray;
export type ConsoleLogs = ConsoleLogEntry[];
export declare class ConsoleLogger {
    private readonly _consoleLogs;
    constructor();
    getLogMessages(maybeDecodedMessageTrace: MessageTrace): string[];
    getExecutionLogs(maybeDecodedMessageTrace: MessageTrace): ConsoleLogs[];
    private _collectExecutionLogs;
    /**
     * Temporary code to print console.sol messages that come from EDR
     */
    getDecodedLogs(messages: Buffer[]): string[];
    private _maybeConsoleLog;
    private _replaceNumberFormatSpecifiers;
    private _decode;
}
export declare function consoleLogToString(log: ConsoleLogs): string;
export {};
//# sourceMappingURL=consoleLogger.d.ts.map