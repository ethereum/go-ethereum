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

import {
	HexString,
	JsonRpcPayload,
	JsonRpcResponse,
	Transaction,
	Web3APIMethod,
	Web3APIReturnType,
} from 'web3-types';
import { Schema } from 'web3-validator';

export type TransactionTypeParser = (transaction: Transaction) => HexString | undefined;

export interface Method {
	name: string;
	call: string;
}

export interface ExtensionObject {
	property?: string;
	methods: Method[];
}

export interface RequestManagerMiddleware<API> {
	processRequest<ParamType = unknown[]>(
		request: JsonRpcPayload<ParamType>,
		options?: { [key: string]: unknown },
	): Promise<JsonRpcPayload<ParamType>>;

	processResponse<
		AnotherMethod extends Web3APIMethod<API>,
		ResponseType = Web3APIReturnType<API, AnotherMethod>,
	>(
		response: JsonRpcResponse<ResponseType>,
		options?: { [key: string]: unknown },
	): Promise<JsonRpcResponse<ResponseType>>;
}

export type CustomTransactionSchema = {
	type: string;
	properties: Record<string, Schema>;
};
