import type { FetchGetUrlFunc, FetchRequest, FetchCancelSignal, GetUrlResponse } from "./fetch.js";
declare global {
    class Headers {
        constructor(values: Array<[string, string]>);
        forEach(func: (v: string, k: string) => void): void;
    }
    class Response {
        status: number;
        statusText: string;
        headers: Headers;
        arrayBuffer(): Promise<ArrayBuffer>;
    }
    type FetchInit = {
        method?: string;
        headers?: Headers;
        body?: Uint8Array;
    };
    function fetch(url: string, init: FetchInit): Promise<Response>;
}
export declare function createGetUrl(options?: Record<string, any>): FetchGetUrlFunc;
export declare function getUrl(req: FetchRequest, _signal?: FetchCancelSignal): Promise<GetUrlResponse>;
//# sourceMappingURL=geturl-browser.d.ts.map