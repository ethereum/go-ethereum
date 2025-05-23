/// <reference types="node" />
/// <reference types="node" />
/** The decoded string representation of the arguments supplied to console.log */
export type ConsoleLogArgs = string[];
export type ConsoleLogs = ConsoleLogArgs[];
export declare class ConsoleLogger {
    /**
     * Temporary code to print console.sol messages that come from EDR
     */
    static getDecodedLogs(messages: Buffer[]): string[];
    /**
     * Returns a formatted string using the first argument as a `printf`-like
     * format string which can contain zero or more format specifiers.
     *
     * If there are more arguments passed than the number of specifiers, the
     * extra arguments are concatenated to the returned string, separated by spaces.
     */
    static format(args?: ConsoleLogArgs): string;
    /** Decodes a calldata buffer into string arguments for a console log. */
    private static _maybeConsoleLog;
    /** Decodes calldata parameters from `data` according to `types` into their string representation. */
    private static _decode;
}
//# sourceMappingURL=consoleLogger.d.ts.map