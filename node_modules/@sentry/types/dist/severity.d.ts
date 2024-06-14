/** JSDoc */
export declare enum Severity {
    /** JSDoc */
    Fatal = "fatal",
    /** JSDoc */
    Error = "error",
    /** JSDoc */
    Warning = "warning",
    /** JSDoc */
    Log = "log",
    /** JSDoc */
    Info = "info",
    /** JSDoc */
    Debug = "debug",
    /** JSDoc */
    Critical = "critical"
}
export declare namespace Severity {
    /**
     * Converts a string-based level into a {@link Severity}.
     *
     * @param level string representation of Severity
     * @returns Severity
     */
    function fromString(level: string): Severity;
}
//# sourceMappingURL=severity.d.ts.map