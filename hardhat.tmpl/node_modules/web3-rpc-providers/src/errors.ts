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

import { BaseWeb3Error } from 'web3-errors';

const ERR_QUICK_NODE_RATE_LIMIT = 1300;
export class QuickNodeRateLimitError extends BaseWeb3Error {
	public code = ERR_QUICK_NODE_RATE_LIMIT;

	public constructor(error?: Error) {
		super(
			`You've reach the rate limit of free RPC calls from our Partner Quick Nodes. There are two options you can either create a paid Quick Nodes account and get 20% off for 2 months using WEB3JS referral code, or use Free public RPC endpoint.`,
			error,
		);
	}
}

const ERR_PROVIDER_CONFIG_OPTIONS = 1301;
export class ProviderConfigOptionsError extends BaseWeb3Error {
	public code = ERR_PROVIDER_CONFIG_OPTIONS;

	public constructor(msg: string) {
		super(`Invalid provider config options given for ${msg}`);
	}
}
/* eslint-enable max-classes-per-file */
