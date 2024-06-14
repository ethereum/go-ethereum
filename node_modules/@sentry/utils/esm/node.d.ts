import { ExtractedNodeRequestData } from '@sentry/types';
/**
 * Checks whether we're in the Node.js or Browser environment
 *
 * @returns Answer to given question
 */
export declare function isNodeEnv(): boolean;
/**
 * Requires a module which is protected against bundler minification.
 *
 * @param request The module path to resolve
 */
export declare function dynamicRequire(mod: any, request: string): any;
/**
 * Normalizes data from the request object, accounting for framework differences.
 *
 * @param req The request object from which to extract data
 * @param keys An optional array of keys to include in the normalized data. Defaults to DEFAULT_REQUEST_KEYS if not
 * provided.
 * @returns An object containing normalized request data
 */
export declare function extractNodeRequestData(req: {
    [key: string]: any;
}, keys?: string[]): ExtractedNodeRequestData;
//# sourceMappingURL=node.d.ts.map