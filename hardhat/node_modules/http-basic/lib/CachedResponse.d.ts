import { Headers } from './Headers';
interface CachedResponse {
    statusCode: number;
    headers: Headers;
    body: NodeJS.ReadableStream;
    requestHeaders: Headers;
    requestTimestamp: number;
}
export { CachedResponse };
