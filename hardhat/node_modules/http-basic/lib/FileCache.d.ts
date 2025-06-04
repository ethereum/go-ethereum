import { ICache } from './ICache';
import { CachedResponse } from './CachedResponse';
export default class FileCache implements ICache {
    private readonly _location;
    constructor(location: string);
    getResponse(url: string, callback: (err: null | Error, response: null | CachedResponse) => void): void;
    setResponse(url: string, response: CachedResponse): void;
    updateResponseHeaders(url: string, response: Pick<CachedResponse, 'headers' | 'requestTimestamp'>): void;
    invalidateResponse(url: string, callback: (err: NodeJS.ErrnoException | null) => void): void;
}
