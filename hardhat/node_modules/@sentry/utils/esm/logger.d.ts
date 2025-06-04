/** JSDoc */
declare class Logger {
    /** JSDoc */
    private _enabled;
    /** JSDoc */
    constructor();
    /** JSDoc */
    disable(): void;
    /** JSDoc */
    enable(): void;
    /** JSDoc */
    log(...args: any[]): void;
    /** JSDoc */
    warn(...args: any[]): void;
    /** JSDoc */
    error(...args: any[]): void;
}
declare const logger: Logger;
export { logger };
//# sourceMappingURL=logger.d.ts.map