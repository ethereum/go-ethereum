export declare type ConnectionInfo = {
    url: string;
    headers?: {
        [key: string]: string | number;
    };
    user?: string;
    password?: string;
    allowInsecureAuthentication?: boolean;
    allowGzip?: boolean;
    throttleLimit?: number;
    throttleSlotInterval?: number;
    throttleCallback?: (attempt: number, url: string) => Promise<boolean>;
    skipFetchSetup?: boolean;
    fetchOptions?: Record<string, string>;
    errorPassThrough?: boolean;
    timeout?: number;
};
export interface OnceBlockable {
    once(eventName: "block", handler: () => void): void;
}
export interface OncePollable {
    once(eventName: "poll", handler: () => void): void;
}
export declare type PollOptions = {
    timeout?: number;
    floor?: number;
    ceiling?: number;
    interval?: number;
    retryLimit?: number;
    onceBlock?: OnceBlockable;
    oncePoll?: OncePollable;
};
export declare type FetchJsonResponse = {
    statusCode: number;
    headers: {
        [header: string]: string;
    };
};
export declare function _fetchData<T = Uint8Array>(connection: string | ConnectionInfo, body?: Uint8Array, processFunc?: (value: Uint8Array, response: FetchJsonResponse) => T): Promise<T>;
export declare function fetchJson(connection: string | ConnectionInfo, json?: string, processFunc?: (value: any, response: FetchJsonResponse) => any): Promise<any>;
export declare function poll<T>(func: () => Promise<T>, options?: PollOptions): Promise<T>;
//# sourceMappingURL=index.d.ts.map