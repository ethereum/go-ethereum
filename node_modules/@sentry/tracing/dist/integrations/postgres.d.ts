import { Hub } from '@sentry/hub';
import { EventProcessor, Integration } from '@sentry/types';
/** Tracing integration for node-postgres package */
export declare class Postgres implements Integration {
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
    setupOnce(_: (callback: EventProcessor) => void, getCurrentHub: () => Hub): void;
}
//# sourceMappingURL=postgres.d.ts.map