import { CachedResponse } from './CachedResponse';
export default class MemoryCache {
    private readonly _cache;
    getResponse(url: string, callback: (err: null | Error, response: null | CachedResponse) => void): void;
    updateResponseHeaders(url: string, response: Pick<CachedResponse, 'headers' | 'requestTimestamp'>): void;
    setResponse(url: string, response: CachedResponse): void;
    invalidateResponse(url: string, callback: (err: NodeJS.ErrnoException | null) => void): void;
}
