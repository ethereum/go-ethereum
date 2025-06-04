import type * as Undici from "undici";
export declare function sendGetRequest(url: URL): Promise<Undici.Dispatcher.ResponseData>;
export declare function sendPostRequest(url: URL, body: string, headers?: Record<string, string>): Promise<Undici.Dispatcher.ResponseData>;
export declare function isSuccessStatusCode(statusCode: number): boolean;
//# sourceMappingURL=undici.d.ts.map