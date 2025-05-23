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

import { BaseWeb3Error, ERR_VALIDATION } from 'web3-errors';
import { Web3ValidationErrorObject } from 'web3-types';

const errorFormatter = (error: Web3ValidationErrorObject): string => {
	if (error.message) {
		return error.message;
	}

	return 'unspecified error';
};

export class Web3ValidatorError extends BaseWeb3Error {
	public code = ERR_VALIDATION;
	public readonly errors: Web3ValidationErrorObject[];

	public constructor(errors: Web3ValidationErrorObject[]) {
		super();

		this.errors = errors;

		super.message = `Web3 validator found ${
			errors.length
		} error[s]:\n${this._compileErrors().join('\n')}`;
	}

	private _compileErrors(): string[] {
		return this.errors.map(errorFormatter);
	}
}
