import { Hub } from '@sentry/hub';
import { Options, TraceparentData, Transaction } from '@sentry/types';
export declare const TRACEPARENT_REGEXP: RegExp;
/**
 * Determines if tracing is currently enabled.
 *
 * Tracing is enabled when at least one of `tracesSampleRate` and `tracesSampler` is defined in the SDK config.
 */
export declare function hasTracingEnabled(options: Options): boolean;
/**
 * Extract transaction context data from a `sentry-trace` header.
 *
 * @param traceparent Traceparent string
 *
 * @returns Object containing data from the header, or undefined if traceparent string is malformed
 */
export declare function extractTraceparentData(traceparent: string): TraceparentData | undefined;
/** Grabs active transaction off scope, if any */
export declare function getActiveTransaction<T extends Transaction>(hub?: Hub): T | undefined;
/**
 * Converts from milliseconds to seconds
 * @param time time in ms
 */
export declare function msToSec(time: number): number;
/**
 * Converts from seconds to milliseconds
 * @param time time in seconds
 */
export declare function secToMs(time: number): number;
export { stripUrlQueryAndFragment } from '@sentry/utils';
//# sourceMappingURL=utils.d.ts.map