import { BaseClient, Scope } from '@sentry/core';
import { Event, EventHint } from '@sentry/types';
import { NodeBackend, NodeOptions } from './backend';
/**
 * The Sentry Node SDK Client.
 *
 * @see NodeOptions for documentation on configuration options.
 * @see SentryClient for usage documentation.
 */
export declare class NodeClient extends BaseClient<NodeBackend, NodeOptions> {
    /**
     * Creates a new Node SDK instance.
     * @param options Configuration options for this SDK.
     */
    constructor(options: NodeOptions);
    /**
     * @inheritDoc
     */
    protected _prepareEvent(event: Event, scope?: Scope, hint?: EventHint): PromiseLike<Event | null>;
}
//# sourceMappingURL=client.d.ts.map