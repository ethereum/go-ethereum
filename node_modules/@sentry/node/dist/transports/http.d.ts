import { Event, Response, TransportOptions } from '@sentry/types';
import { BaseTransport } from './base';
/** Node http module transport */
export declare class HTTPTransport extends BaseTransport {
    options: TransportOptions;
    /** Create a new instance and set this.agent */
    constructor(options: TransportOptions);
    /**
     * @inheritDoc
     */
    sendEvent(event: Event): Promise<Response>;
}
//# sourceMappingURL=http.d.ts.map