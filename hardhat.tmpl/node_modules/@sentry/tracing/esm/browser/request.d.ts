import { Span } from '../span';
export declare const DEFAULT_TRACING_ORIGINS: (string | RegExp)[];
/** Options for Request Instrumentation */
export interface RequestInstrumentationOptions {
    /**
     * List of strings / regex where the integration should create Spans out of. Additionally this will be used
     * to define which outgoing requests the `sentry-trace` header will be attached to.
     *
     * Default: ['localhost', /^\//] {@see DEFAULT_TRACING_ORIGINS}
     */
    tracingOrigins: Array<string | RegExp>;
    /**
     * Flag to disable patching all together for fetch requests.
     *
     * Default: true
     */
    traceFetch: boolean;
    /**
     * Flag to disable patching all together for xhr requests.
     *
     * Default: true
     */
    traceXHR: boolean;
    /**
     * This function will be called before creating a span for a request with the given url.
     * Return false if you don't want a span for the given url.
     *
     * By default it uses the `tracingOrigins` options as a url match.
     */
    shouldCreateSpanForRequest?(url: string): boolean;
}
/** Data returned from fetch callback */
export interface FetchData {
    args: any[];
    fetchData?: {
        method: string;
        url: string;
        __span?: string;
    };
    response?: any;
    startTimestamp: number;
    endTimestamp?: number;
}
/** Data returned from XHR request */
export interface XHRData {
    xhr?: {
        __sentry_xhr__?: {
            method: string;
            url: string;
            status_code: number;
            data: Record<string, any>;
        };
        __sentry_xhr_span_id__?: string;
        setRequestHeader?: (key: string, val: string) => void;
        __sentry_own_request__?: boolean;
    };
    startTimestamp: number;
    endTimestamp?: number;
}
export declare const defaultRequestInstrumentationOptions: RequestInstrumentationOptions;
/** Registers span creators for xhr and fetch requests  */
export declare function registerRequestInstrumentation(_options?: Partial<RequestInstrumentationOptions>): void;
/**
 * Create and track fetch request spans
 */
export declare function fetchCallback(handlerData: FetchData, shouldCreateSpan: (url: string) => boolean, spans: Record<string, Span>): void;
/**
 * Create and track xhr request spans
 */
export declare function xhrCallback(handlerData: XHRData, shouldCreateSpan: (url: string) => boolean, spans: Record<string, Span>): void;
//# sourceMappingURL=request.d.ts.map