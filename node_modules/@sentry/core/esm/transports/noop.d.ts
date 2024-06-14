import { Event, Response, Transport } from '@sentry/types';
/** Noop transport */
export declare class NoopTransport implements Transport {
    /**
     * @inheritDoc
     */
    sendEvent(_: Event): PromiseLike<Response>;
    /**
     * @inheritDoc
     */
    close(_?: number): PromiseLike<boolean>;
}
//# sourceMappingURL=noop.d.ts.map