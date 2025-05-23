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
export type JsonRpcId = string | number | undefined;
export type JsonRpcResult = string | number | boolean | Record<string, unknown>;
export type JsonRpcIdentifier = string & ('2.0' | '1.0');

export interface JsonRpcError<T = JsonRpcResult> {
	readonly code: number;
	readonly message: string;
	readonly data?: T;
}

export interface JsonRpcResponseWithError<Error = JsonRpcResult> {
	readonly id: JsonRpcId;
	readonly jsonrpc: JsonRpcIdentifier;
	readonly error: JsonRpcError<Error>;
	readonly result?: never;
}

export interface JsonRpcResponseWithResult<T = JsonRpcResult> {
	readonly id: JsonRpcId;
	readonly jsonrpc: JsonRpcIdentifier;
	readonly error?: never;
	readonly result: T;
}

export interface SubscriptionParams<T = JsonRpcResult> {
	readonly subscription: string; // for subscription id
	readonly result: T;
}

export interface JsonRpcSubscriptionResultOld<T = JsonRpcResult> {
	readonly error?: never;
	readonly params?: never;
	readonly type: string;
	readonly data: SubscriptionParams<T>;
}

export interface JsonRpcNotification<T = JsonRpcResult> {
	readonly id?: JsonRpcId;
	readonly jsonrpc: JsonRpcIdentifier;
	readonly method: string; // for subscription
	readonly params: SubscriptionParams<T>; // for subscription results
	readonly result?: never;
	readonly data?: never;
	readonly error?: never;
}

export interface JsonRpcSubscriptionResult {
	readonly id: number;
	readonly jsonrpc: string;
	readonly result: string;
	readonly method: never;
	readonly params: never;
	readonly data?: never;
}

export interface JsonRpcRequest<T = unknown[]> {
	readonly id: JsonRpcId;
	readonly jsonrpc: JsonRpcIdentifier;
	readonly method: string;
	readonly params?: T;
}

export interface JsonRpcOptionalRequest<ParamType = unknown[]>
	extends Omit<JsonRpcRequest<ParamType>, 'id' | 'jsonrpc'> {
	readonly id?: JsonRpcId;
	readonly jsonrpc?: JsonRpcIdentifier;
}

export type JsonRpcBatchRequest = JsonRpcRequest[];

export type JsonRpcPayload<Param = unknown[]> = JsonRpcRequest<Param> | JsonRpcBatchRequest;

export type JsonRpcBatchResponse<Result = JsonRpcResult, Error = JsonRpcResult> =
	| (JsonRpcResponseWithError<Error> | JsonRpcResponseWithResult<Result>)[];

export type JsonRpcResponse<Result = JsonRpcResult, Error = JsonRpcResult> =
	| JsonRpcResponseWithError<Error>
	| JsonRpcResponseWithResult<Result>
	| JsonRpcBatchResponse<Result, Error>
	| JsonRpcNotification<Result>;
