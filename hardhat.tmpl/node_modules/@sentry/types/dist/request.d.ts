/** Possible SentryRequest types that can be used to make a distinction between Sentry features */
export declare type SentryRequestType = 'event' | 'transaction' | 'session';
/** A generic client request. */
export interface SentryRequest {
    body: string;
    type: SentryRequestType;
    url: string;
}
/** Request data included in an event as sent to Sentry */
export interface Request {
    url?: string;
    method?: string;
    data?: any;
    query_string?: string;
    cookies?: {
        [key: string]: string;
    };
    env?: {
        [key: string]: string;
    };
    headers?: {
        [key: string]: string;
    };
}
//# sourceMappingURL=request.d.ts.map