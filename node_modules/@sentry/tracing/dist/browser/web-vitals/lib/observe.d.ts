export interface PerformanceEntryHandler {
    (entry: PerformanceEntry): void;
}
/**
 * Takes a performance entry type and a callback function, and creates a
 * `PerformanceObserver` instance that will observe the specified entry type
 * with buffering enabled and call the callback _for each entry_.
 *
 * This function also feature-detects entry support and wraps the logic in a
 * try/catch to avoid errors in unsupporting browsers.
 */
export declare const observe: (type: string, callback: PerformanceEntryHandler) => PerformanceObserver | undefined;
//# sourceMappingURL=observe.d.ts.map