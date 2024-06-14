import { Integration } from '@sentry/types';
/** http module integration */
export declare class Http implements Integration {
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
    private readonly _breadcrumbs;
    /**
     * @inheritDoc
     */
    private readonly _tracing;
    /**
     * @inheritDoc
     */
    constructor(options?: {
        breadcrumbs?: boolean;
        tracing?: boolean;
    });
    /**
     * @inheritDoc
     */
    setupOnce(): void;
}
//# sourceMappingURL=http.d.ts.map