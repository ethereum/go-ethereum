import { Integration } from '@sentry/types';
/** JSDoc */
interface InboundFiltersOptions {
    allowUrls: Array<string | RegExp>;
    denyUrls: Array<string | RegExp>;
    ignoreErrors: Array<string | RegExp>;
    ignoreInternal: boolean;
    /** @deprecated use {@link InboundFiltersOptions.allowUrls} instead. */
    whitelistUrls: Array<string | RegExp>;
    /** @deprecated use {@link InboundFiltersOptions.denyUrls} instead. */
    blacklistUrls: Array<string | RegExp>;
}
/** Inbound filters configurable by the user */
export declare class InboundFilters implements Integration {
    private readonly _options;
    /**
     * @inheritDoc
     */
    static id: string;
    /**
     * @inheritDoc
     */
    name: string;
    constructor(_options?: Partial<InboundFiltersOptions>);
    /**
     * @inheritDoc
     */
    setupOnce(): void;
    /** JSDoc */
    private _shouldDropEvent;
    /** JSDoc */
    private _isSentryError;
    /** JSDoc */
    private _isIgnoredError;
    /** JSDoc */
    private _isDeniedUrl;
    /** JSDoc */
    private _isAllowedUrl;
    /** JSDoc */
    private _mergeOptions;
    /** JSDoc */
    private _getPossibleEventMessages;
    /** JSDoc */
    private _getEventFilterUrl;
}
export {};
//# sourceMappingURL=inboundfilters.d.ts.map