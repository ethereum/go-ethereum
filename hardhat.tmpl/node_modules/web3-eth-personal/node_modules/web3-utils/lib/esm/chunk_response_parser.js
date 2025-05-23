import { InvalidResponseError } from 'web3-errors';
export class ChunkResponseParser {
    constructor(eventEmitter, autoReconnect) {
        this.eventEmitter = eventEmitter;
        this.autoReconnect = autoReconnect;
        this.chunkTimeout = 1000 * 15;
    }
    clearQueues() {
        if (typeof this._clearQueues === 'function') {
            this._clearQueues();
        }
    }
    onError(clearQueues) {
        this._clearQueues = clearQueues;
    }
    parseResponse(data) {
        const returnValues = [];
        // DE-CHUNKER
        const dechunkedData = data
            .replace(/\}[\n\r]?\{/g, '}|--|{') // }{
            .replace(/\}\][\n\r]?\[\{/g, '}]|--|[{') // }][{
            .replace(/\}[\n\r]?\[\{/g, '}|--|[{') // }[{
            .replace(/\}\][\n\r]?\{/g, '}]|--|{') // }]{
            .split('|--|');
        dechunkedData.forEach(_chunkData => {
            // prepend the last chunk
            let chunkData = _chunkData;
            if (this.lastChunk) {
                chunkData = this.lastChunk + chunkData;
            }
            let result;
            try {
                result = JSON.parse(chunkData);
            }
            catch (e) {
                this.lastChunk = chunkData;
                // start timeout to cancel all requests
                if (this.lastChunkTimeout) {
                    clearTimeout(this.lastChunkTimeout);
                }
                this.lastChunkTimeout = setTimeout(() => {
                    if (this.autoReconnect)
                        return;
                    this.clearQueues();
                    this.eventEmitter.emit('error', new InvalidResponseError({
                        id: 1,
                        jsonrpc: '2.0',
                        error: { code: 2, message: 'Chunk timeout' },
                    }));
                }, this.chunkTimeout);
                return;
            }
            // cancel timeout and set chunk to null
            clearTimeout(this.lastChunkTimeout);
            this.lastChunk = undefined;
            if (result)
                returnValues.push(result);
        });
        return returnValues;
    }
}
//# sourceMappingURL=chunk_response_parser.js.map