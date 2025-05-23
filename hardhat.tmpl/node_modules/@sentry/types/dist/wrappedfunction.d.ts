/** JSDoc */
export interface WrappedFunction extends Function {
    [key: string]: any;
    __sentry__?: boolean;
    __sentry_wrapped__?: WrappedFunction;
    __sentry_original__?: WrappedFunction;
}
//# sourceMappingURL=wrappedfunction.d.ts.map