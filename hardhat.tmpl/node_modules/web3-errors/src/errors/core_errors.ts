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

import { BaseWeb3Error } from '../web3_error_base.js';
import { ERR_CORE_HARDFORK_MISMATCH } from '../error_codes.js';

export class ConfigHardforkMismatchError extends BaseWeb3Error {
	public code = ERR_CORE_HARDFORK_MISMATCH;

	public constructor(defaultHardfork: string, commonHardFork: string) {
		super(
			`Web3Config hardfork doesnt match in defaultHardfork ${defaultHardfork} and common.hardfork ${commonHardFork}`,
		);
	}
}

export class ConfigChainMismatchError extends BaseWeb3Error {
	public code = ERR_CORE_HARDFORK_MISMATCH;

	public constructor(defaultHardfork: string, commonHardFork: string) {
		super(
			`Web3Config chain doesnt match in defaultHardfork ${defaultHardfork} and common.hardfork ${commonHardFork}`,
		);
	}
}
