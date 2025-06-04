import { Integration } from '@sentry/types';
/** Adds SDK info to an event. */
export declare class LinkedErrors implements Integration {
    /**
     * @inheritDoc
     */
    static id: string;
    /**
     * @inheritDoc
     */
    readonly name: string;
    /**
     * @inheritDoc
     */
    private readonly _key;
    /**
     * @inheritDoc
     */
    private readonly _limit;
    /**
     * @inheritDoc
     */
    constructor(options?: {
        key?: string;
        limit?: number;
    });
    /**
     * @inheritDoc
     */
    setupOnce(): void;
    /**
     * @inheritDoc
     */
    private _handler;
    /**
     * @inheritDoc
     */
    private _walkErrorTree;
}
//# sourceMappingURL=linkederrors.d.ts.map