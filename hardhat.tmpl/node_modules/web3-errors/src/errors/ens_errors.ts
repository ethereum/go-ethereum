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

/* eslint-disable max-classes-per-file */

import {
	ERR_ENS_CHECK_INTERFACE_SUPPORT,
	ERR_ENS_NETWORK_NOT_SYNCED,
	ERR_ENS_UNSUPPORTED_NETWORK,
} from '../error_codes.js';
import { BaseWeb3Error } from '../web3_error_base.js';

export class ENSCheckInterfaceSupportError extends BaseWeb3Error {
	public code = ERR_ENS_CHECK_INTERFACE_SUPPORT;
	public constructor(errorDetails: string) {
		super(`ENS resolver check interface support error. "${errorDetails}"`);
	}
}

export class ENSUnsupportedNetworkError extends BaseWeb3Error {
	public code = ERR_ENS_UNSUPPORTED_NETWORK;
	public constructor(networkType: string) {
		super(`ENS is not supported on network ${networkType}`);
	}
}

export class ENSNetworkNotSyncedError extends BaseWeb3Error {
	public code = ERR_ENS_NETWORK_NOT_SYNCED;
	public constructor() {
		super(`Network not synced`);
	}
}
