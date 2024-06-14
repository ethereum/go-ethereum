/**
 * Returns a timestamp in seconds since the UNIX epoch using the Date API.
 */
export declare const dateTimestampInSeconds: any;
/**
 * Returns a timestamp in seconds since the UNIX epoch using either the Performance or Date APIs, depending on the
 * availability of the Performance API.
 *
 * See `usingPerformanceAPI` to test whether the Performance API is used.
 *
 * BUG: Note that because of how browsers implement the Performance API, the clock might stop when the computer is
 * asleep. This creates a skew between `dateTimestampInSeconds` and `timestampInSeconds`. The
 * skew can grow to arbitrary amounts like days, weeks or months.
 * See https://github.com/getsentry/sentry-javascript/issues/2590.
 */
export declare const timestampInSeconds: any;
export declare const timestampWithMs: any;
/**
 * A boolean that is true when timestampInSeconds uses the Performance API to produce monotonic timestamps.
 */
export declare const usingPerformanceAPI: boolean;
/**
 * The number of milliseconds since the UNIX epoch. This value is only usable in a browser, and only when the
 * performance API is available.
 */
export declare const browserPerformanceTimeOrigin: number | undefined;
//# sourceMappingURL=time.d.ts.map