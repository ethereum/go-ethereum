import { Event, Integration, StackFrame } from '@sentry/types';
/** Internal */
interface SentryGlobal {
    Sentry?: {
        Integrations?: Integration[];
    };
    SENTRY_ENVIRONMENT?: string;
    SENTRY_DSN?: string;
    SENTRY_RELEASE?: {
        id?: string;
    };
    __SENTRY__: {
        globalEventProcessors: any;
        hub: any;
        logger: any;
    };
}
/**
 * Safely get global scope object
 *
 * @returns Global scope object
 */
export declare function getGlobalObject<T>(): T & SentryGlobal;
/**
 * UUID4 generator
 *
 * @returns string Generated UUID4.
 */
export declare function uuid4(): string;
/**
 * Parses string form of URL into an object
 * // borrowed from https://tools.ietf.org/html/rfc3986#appendix-B
 * // intentionally using regex and not <a/> href parsing trick because React Native and other
 * // environments where DOM might not be available
 * @returns parsed URL object
 */
export declare function parseUrl(url: string): {
    host?: string;
    path?: string;
    protocol?: string;
    relative?: string;
};
/**
 * Extracts either message or type+value from an event that can be used for user-facing logs
 * @returns event's description
 */
export declare function getEventDescription(event: Event): string;
/** JSDoc */
export declare function consoleSandbox(callback: () => any): any;
/**
 * Adds exception values, type and value to an synthetic Exception.
 * @param event The event to modify.
 * @param value Value of the exception.
 * @param type Type of the exception.
 * @hidden
 */
export declare function addExceptionTypeValue(event: Event, value?: string, type?: string): void;
/**
 * Adds exception mechanism to a given event.
 * @param event The event to modify.
 * @param mechanism Mechanism of the mechanism.
 * @hidden
 */
export declare function addExceptionMechanism(event: Event, mechanism?: {
    [key: string]: any;
}): void;
/**
 * A safe form of location.href
 */
export declare function getLocationHref(): string;
/**
 * Represents Semantic Versioning object
 */
interface SemVer {
    major?: number;
    minor?: number;
    patch?: number;
    prerelease?: string;
    buildmetadata?: string;
}
/**
 * Parses input into a SemVer interface
 * @param input string representation of a semver version
 */
export declare function parseSemver(input: string): SemVer;
/**
 * Extracts Retry-After value from the request header or returns default value
 * @param now current unix timestamp
 * @param header string representation of 'Retry-After' header
 */
export declare function parseRetryAfterHeader(now: number, header?: string | number | null): number;
/**
 * This function adds context (pre/post/line) lines to the provided frame
 *
 * @param lines string[] containing all lines
 * @param frame StackFrame that will be mutated
 * @param linesOfContext number of context lines we want to add pre/post
 */
export declare function addContextToFrame(lines: string[], frame: StackFrame, linesOfContext?: number): void;
/**
 * Strip the query string and fragment off of a given URL or path (if present)
 *
 * @param urlPath Full URL or path, including possible query string and/or fragment
 * @returns URL or path without query string or fragment
 */
export declare function stripUrlQueryAndFragment(urlPath: string): string;
export {};
//# sourceMappingURL=misc.d.ts.map