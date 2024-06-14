import { IncomingHttpHeaders } from 'http';
/**
 * A response from a web request
 */
declare class Response<TBody> {
    readonly statusCode: number;
    readonly headers: IncomingHttpHeaders;
    readonly body: TBody;
    readonly url: string;
    constructor(statusCode: number, headers: IncomingHttpHeaders, body: TBody, url: string);
    isError(): boolean;
    getBody(encoding: string): string;
    getBody(): TBody;
}
export = Response;
