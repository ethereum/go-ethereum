import { Integration } from '@sentry/types';
/** Global Promise Rejection handler */
export declare class OnUncaughtException implements Integration {
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
    readonly handler: (error: Error) => void;
    /**
     * @inheritDoc
     */
    constructor(_options?: {
        /**
         * Default onFatalError handler
         * @param firstError Error that has been thrown
         * @param secondError If this was called multiple times this will be set
         */
        onFatalError?(firstError: Error, secondError?: Error): void;
    });
    /**
     * @inheritDoc
     */
    setupOnce(): void;
    /**
     * @hidden
     */
    private _makeErrorHandler;
}
//# sourceMappingURL=onuncaughtexception.d.ts.map