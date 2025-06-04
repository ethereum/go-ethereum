import { getCurrentHub } from '@sentry/hub';
export var TRACEPARENT_REGEXP = new RegExp('^[ \\t]*' + // whitespace
    '([0-9a-f]{32})?' + // trace_id
    '-?([0-9a-f]{16})?' + // span_id
    '-?([01])?' + // sampled
    '[ \\t]*$');
/**
 * Determines if tracing is currently enabled.
 *
 * Tracing is enabled when at least one of `tracesSampleRate` and `tracesSampler` is defined in the SDK config.
 */
export function hasTracingEnabled(options) {
    return 'tracesSampleRate' in options || 'tracesSampler' in options;
}
/**
 * Extract transaction context data from a `sentry-trace` header.
 *
 * @param traceparent Traceparent string
 *
 * @returns Object containing data from the header, or undefined if traceparent string is malformed
 */
export function extractTraceparentData(traceparent) {
    var matches = traceparent.match(TRACEPARENT_REGEXP);
    if (matches) {
        var parentSampled = void 0;
        if (matches[3] === '1') {
            parentSampled = true;
        }
        else if (matches[3] === '0') {
            parentSampled = false;
        }
        return {
            traceId: matches[1],
            parentSampled: parentSampled,
            parentSpanId: matches[2],
        };
    }
    return undefined;
}
/** Grabs active transaction off scope, if any */
export function getActiveTransaction(hub) {
    if (hub === void 0) { hub = getCurrentHub(); }
    var _a, _b;
    return (_b = (_a = hub) === null || _a === void 0 ? void 0 : _a.getScope()) === null || _b === void 0 ? void 0 : _b.getTransaction();
}
/**
 * Converts from milliseconds to seconds
 * @param time time in ms
 */
export function msToSec(time) {
    return time / 1000;
}
/**
 * Converts from seconds to milliseconds
 * @param time time in seconds
 */
export function secToMs(time) {
    return time * 1000;
}
// so it can be used in manual instrumentation without necessitating a hard dependency on @sentry/utils
export { stripUrlQueryAndFragment } from '@sentry/utils';
//# sourceMappingURL=utils.js.map