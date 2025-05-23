import { DsnLike } from './dsn';
import { Event } from './event';
import { Response } from './response';
import { Session } from './session';
/** Transport used sending data to Sentry */
export interface Transport {
    /**
     * Sends the event to the Store endpoint in Sentry.
     *
     * @param event Event that should be sent to Sentry.
     */
    sendEvent(event: Event): PromiseLike<Response>;
    /**
     * Sends the session to the Store endpoint in Sentry.
     *
     * @param body Session that should be sent to Sentry.
     */
    sendSession?(session: Session): PromiseLike<Response>;
    /**
     * Call this function to wait until all pending requests have been sent.
     *
     * @param timeout Number time in ms to wait until the buffer is drained.
     */
    close(timeout?: number): PromiseLike<boolean>;
}
/** JSDoc */
export declare type TransportClass<T extends Transport> = new (options: TransportOptions) => T;
/** JSDoc */
export interface TransportOptions {
    /** Sentry DSN */
    dsn: DsnLike;
    /** Define custom headers */
    headers?: {
        [key: string]: string;
    };
    /** Set a HTTP proxy that should be used for outbound requests. */
    httpProxy?: string;
    /** Set a HTTPS proxy that should be used for outbound requests. */
    httpsProxy?: string;
    /** HTTPS proxy certificates path */
    caCerts?: string;
    /** Fetch API init parameters */
    fetchParameters?: {
        [key: string]: string;
    };
}
//# sourceMappingURL=transport.d.ts.map