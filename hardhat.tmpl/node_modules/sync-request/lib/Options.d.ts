/// <reference types="node" />
import { Options as AsyncOptions } from 'then-request';
import { FormData, FormDataEntry } from './FormData';
export interface BaseOptions extends Pick<AsyncOptions, 'allowRedirectHeaders' | 'followRedirects' | 'gzip' | 'headers' | 'maxRedirects' | 'maxRetries' | 'qs' | 'json'> {
    agent?: boolean;
    cache?: 'file';
    retry?: boolean;
    retryDelay?: number;
    socketTimeout?: number;
    timeout?: number;
    body?: string | Buffer;
}
export interface Options extends BaseOptions {
    form?: FormData;
}
export interface MessageOptions extends BaseOptions {
    form?: FormDataEntry[];
}
