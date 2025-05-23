/**
 * Memo class used for decycle json objects. Uses WeakSet if available otherwise array.
 */
export declare class Memo {
    /** Determines if WeakSet is available */
    private readonly _hasWeakSet;
    /** Either WeakSet or Array */
    private readonly _inner;
    constructor();
    /**
     * Sets obj to remember.
     * @param obj Object to remember
     */
    memoize(obj: any): boolean;
    /**
     * Removes object from internal storage.
     * @param obj Object to forget
     */
    unmemoize(obj: any): void;
}
//# sourceMappingURL=memo.d.ts.map