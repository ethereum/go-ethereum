import { Hub } from '@sentry/hub';
import { EventProcessor, Integration } from '@sentry/types';
/** Tracing integration for node-mysql package */
export declare class Mysql implements Integration {
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
//# sourceMappingURL=mysql.d.ts.map