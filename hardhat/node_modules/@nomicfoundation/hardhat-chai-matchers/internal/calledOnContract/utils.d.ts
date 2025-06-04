interface ErrorConstructor<T extends any[]> {
    new (...args: T): Error;
}
export declare function ensure<T extends any[]>(condition: boolean, ErrorToThrow: ErrorConstructor<T>, ...errorArgs: T): asserts condition;
export {};
//# sourceMappingURL=utils.d.ts.map