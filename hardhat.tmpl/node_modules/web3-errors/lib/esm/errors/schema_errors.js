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
import { ERR_SCHEMA_FORMAT } from '../error_codes.js';
import { BaseWeb3Error } from '../web3_error_base.js';
export class SchemaFormatError extends BaseWeb3Error {
    constructor(type) {
        super(`Format for the type ${type} is unsupported`);
        this.type = type;
        this.code = ERR_SCHEMA_FORMAT;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { type: this.type });
    }
}
//# sourceMappingURL=schema_errors.js.map