/// <reference types="node" />
import { Agent } from 'http';
import { IncomingHttpHeaders } from 'http';
import Response = require('http-response-object');
import { ICache, CachedResponse } from 'http-basic';
import FormData = require('form-data');
interface Options {
    allowRedirectHeaders?: string[];
    cache?: 'file' | 'memory' | ICache;
    agent?: boolean | Agent;
    followRedirects?: boolean;
    gzip?: boolean;
    headers?: IncomingHttpHeaders;
    maxRedirects?: number;
    maxRetries?: number;
    retry?: boolean | ((err: NodeJS.ErrnoException | null, res: Response<NodeJS.ReadableStream | Buffer | string> | void, attemptNumber: number) => boolean);
    retryDelay?: number | ((err: NodeJS.ErrnoException | null, res: Response<NodeJS.ReadableStream | Buffer | string> | void, attemptNumber: number) => number);
    socketTimeout?: number;
    timeout?: number;
    isMatch?: (requestHeaders: IncomingHttpHeaders, cachedResponse: CachedResponse, defaultValue: boolean) => boolean;
    isExpired?: (cachedResponse: CachedResponse, defaultValue: boolean) => boolean;
    canCache?: (res: Response<NodeJS.ReadableStream>, defaultValue: boolean) => boolean;
    qs?: {
        [key: string]: any;
    };
    json?: any;
    form?: FormData;
    body?: string | Buffer | NodeJS.ReadableStream;
}
export { Options };
