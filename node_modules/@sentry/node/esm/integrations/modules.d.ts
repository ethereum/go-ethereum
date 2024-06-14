import { EventProcessor, Hub, Integration } from '@sentry/types';
/** Add node modules / packages to the event */
export declare class Modules implements Integration {
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
    setupOnce(addGlobalEventProcessor: (callback: EventProcessor) => void, getCurrentHub: () => Hub): void;
    /** Fetches the list of modules and the versions loaded by the entry file for your node.js app. */
    private _getModules;
}
//# sourceMappingURL=modules.d.ts.map