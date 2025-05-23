/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
import { JsonRpcResponse } from 'web3-types';
import { InvalidResponseError } from 'web3-errors';
import { EventEmitter } from 'eventemitter3';
import { Timeout } from './promise_helpers.js';

export class ChunkResponseParser {
	private lastChunk: string | undefined;
	private lastChunkTimeout: Timeout | undefined;
	private _clearQueues: (() => void) | undefined;
	private readonly eventEmitter: EventEmitter;
	private readonly autoReconnect: boolean;
	private readonly chunkTimeout: number;

	public constructor(eventEmitter: EventEmitter, autoReconnect: boolean) {
		this.eventEmitter = eventEmitter;
		this.autoReconnect = autoReconnect;
		this.chunkTimeout = 1000 * 15;
	}
	private clearQueues(): void {
		if (typeof this._clearQueues === 'function') {
			this._clearQueues();
		}
	}

	public onError(clearQueues?: () => void) {
		this._clearQueues = clearQueues;
	}

	public parseResponse(data: string): JsonRpcResponse[] {
		const returnValues: JsonRpcResponse[] = [];

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
				result = JSON.parse(chunkData) as unknown as JsonRpcResponse;
			} catch (e) {
				this.lastChunk = chunkData;

				// start timeout to cancel all requests
				if (this.lastChunkTimeout) {
					clearTimeout(this.lastChunkTimeout);
				}

				this.lastChunkTimeout = setTimeout(() => {
					if (this.autoReconnect) return;
					this.clearQueues();
					this.eventEmitter.emit(
						'error',
						new InvalidResponseError({
							id: 1,
							jsonrpc: '2.0',
							error: { code: 2, message: 'Chunk timeout' },
						}),
					);
				}, this.chunkTimeout);
				return;
			}

			// cancel timeout and set chunk to null
			clearTimeout(this.lastChunkTimeout);
			this.lastChunk = undefined;

			if (result) returnValues.push(result);
		});

		return returnValues;
	}
}
