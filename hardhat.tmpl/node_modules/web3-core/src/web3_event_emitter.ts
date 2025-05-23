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

import { EventEmitter } from 'web3-utils';

export type Web3EventMap = Record<string, unknown>;
export type Web3EventKey<T extends Web3EventMap> = string & keyof T;
export type Web3EventCallback<T> = (params: T) => void | Promise<void>;
export interface Web3Emitter<T extends Web3EventMap> {
	on<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>): void;
	once<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>): void;
	off<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>): void;
	emit<K extends Web3EventKey<T>>(eventName: K, params: T[K]): void;
}

export class Web3EventEmitter<T extends Web3EventMap> implements Web3Emitter<T> {
	private readonly _emitter = new EventEmitter();

	public on<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>) {
		// eslint-disable-next-line @typescript-eslint/no-misused-promises
		this._emitter.on(eventName, fn);
	}

	public once<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>) {
		// eslint-disable-next-line @typescript-eslint/no-misused-promises
		this._emitter.once(eventName, fn);
	}

	public off<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>) {
		// eslint-disable-next-line @typescript-eslint/no-misused-promises
		this._emitter.off(eventName, fn);
	}

	public emit<K extends Web3EventKey<T>>(eventName: K, params: T[K]) {
		this._emitter.emit(eventName, params);
	}

	public listenerCount<K extends Web3EventKey<T>>(eventName: K) {
		return this._emitter.listenerCount(eventName);
	}

	public listeners<K extends Web3EventKey<T>>(eventName: K) {
		return this._emitter.listeners(eventName);
	}

	public eventNames() {
		return this._emitter.eventNames();
	}

	public removeAllListeners() {
		return this._emitter.removeAllListeners();
	}
	public setMaxListenerWarningThreshold(maxListenersWarningThreshold: number) {
		this._emitter.setMaxListeners(maxListenersWarningThreshold);
	}
	public getMaxListeners() {
		return this._emitter.getMaxListeners();
	}
}
