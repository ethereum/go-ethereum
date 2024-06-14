export declare type FETCH_OPT = {
    method?: string;
    type?: 'text' | 'json' | 'bytes';
    redirect: boolean;
    expectStatusCode?: number | false;
    headers: Record<string, string>;
    data?: object;
    full: boolean;
    keepAlive: boolean;
    cors: boolean;
    referrer: boolean;
    sslAllowSelfSigned: boolean;
    sslPinnedCertificates?: string[];
    _redirectCount: number;
};
export declare class InvalidCertError extends Error {
    readonly fingerprint256: string;
    constructor(msg: string, fingerprint256: string);
}
export declare class InvalidStatusCodeError extends Error {
    readonly statusCode: number;
    constructor(statusCode: number);
}
export default function fetchUrl(url: string, options?: Partial<FETCH_OPT>): Promise<any>;
