import { Integration } from '@sentry/types';
declare type UnhandledRejectionMode = 'none' | 'warn' | 'strict';
/** Global Promise Rejection handler */
export declare class OnUnhandledRejection implements Integration {
    private readonly _options;
    /**
     * @inheritDoc
     */
    static id: string;
    /**
     * @inheritDoc
     */
    name: string;
    /**
     * @inheritDoc
     */
    constructor(_options?: {
        /**
         * Option deciding what to do after capturing unhandledRejection,
         * that mimicks behavior of node's --unhandled-rejection flag.
         */
        mode: UnhandledRejectionMode;
    });
    /**
     * @inheritDoc
     */
    setupOnce(): void;
    /**
     * Send an exception with reason
     * @param reason string
     * @param promise promise
     */
    sendUnhandledPromise(reason: any, promise: any): void;
    /**
     * Handler for `mode` option
     */
    private _handleRejection;
}
export {};
//# sourceMappingURL=onunhandledrejection.d.ts.map