/// <reference types="node" />
import * as http from 'http';
import { URL } from 'url';
/**
 * Checks whether given url points to Sentry server
 * @param url url to verify
 */
export declare function isSentryRequest(url: string): boolean;
/**
 * Assemble a URL to be used for breadcrumbs and spans.
 *
 * @param requestOptions RequestOptions object containing the component parts for a URL
 * @returns Fully-formed URL
 */
export declare function extractUrl(requestOptions: RequestOptions): string;
/**
 * Handle various edge cases in the span description (for spans representing http(s) requests).
 *
 * @param description current `description` property of the span representing the request
 * @param requestOptions Configuration data for the request
 * @param Request Request object
 *
 * @returns The cleaned description
 */
export declare function cleanSpanDescription(description: string | undefined, requestOptions: RequestOptions, request: http.ClientRequest): string | undefined;
export declare type RequestOptions = http.RequestOptions & {
    hash?: string;
    search?: string;
    pathname?: string;
    href?: string;
};
declare type RequestCallback = (response: http.IncomingMessage) => void;
export declare type RequestMethodArgs = [RequestOptions | string | URL, RequestCallback?] | [string | URL, RequestOptions, RequestCallback?];
export declare type RequestMethod = (...args: RequestMethodArgs) => http.ClientRequest;
/**
 * Convert a URL object into a RequestOptions object.
 *
 * Copied from Node's internals (where it's used in http(s).request() and http(s).get()), modified only to use the
 * RequestOptions type above.
 *
 * See https://github.com/nodejs/node/blob/master/lib/internal/url.js.
 */
export declare function urlToOptions(url: URL): RequestOptions;
/**
 * Normalize inputs to `http(s).request()` and `http(s).get()`.
 *
 * Legal inputs to `http(s).request()` and `http(s).get()` can take one of ten forms:
 *     [ RequestOptions | string | URL ],
 *     [ RequestOptions | string | URL, RequestCallback ],
 *     [ string | URL, RequestOptions ], and
 *     [ string | URL, RequestOptions, RequestCallback ].
 *
 * This standardizes to one of two forms: [ RequestOptions ] and [ RequestOptions, RequestCallback ]. A similar thing is
 * done as the first step of `http(s).request()` and `http(s).get()`; this just does it early so that we can interact
 * with the args in a standard way.
 *
 * @param requestArgs The inputs to `http(s).request()` or `http(s).get()`, as an array.
 *
 * @returns Equivalent args of the form [ RequestOptions ] or [ RequestOptions, RequestCallback ].
 */
export declare function normalizeRequestArgs(requestArgs: RequestMethodArgs): [RequestOptions] | [RequestOptions, RequestCallback];
export {};
//# sourceMappingURL=http.d.ts.map