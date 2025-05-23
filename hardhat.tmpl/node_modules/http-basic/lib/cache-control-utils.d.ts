import { CachedResponse } from './CachedResponse';
import Response = require('http-response-object');
export declare type Policy = {
    maxage: number | null;
};
/**
 * returns true if this response is cacheable (according to cache-control headers)
 */
export declare function isCacheable<T>(res: Response<T> | CachedResponse): boolean;
/**
 * if the response is cacheable, returns an object detailing the maxage of the cache
 * otherwise returns null
 */
export declare function cachePolicy<T>(res: Response<T> | CachedResponse): Policy | null;
