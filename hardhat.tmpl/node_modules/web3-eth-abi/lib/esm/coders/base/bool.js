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
import { AbiError, InvalidBooleanError } from 'web3-errors';
import { toBool } from 'web3-utils';
import { WORD_SIZE } from '../utils.js';
import { decodeNumber, encodeNumber } from './number.js';
export function encodeBoolean(param, input) {
    let value;
    try {
        value = toBool(input);
    }
    catch (e) {
        if (e instanceof InvalidBooleanError) {
            throw new AbiError('provided input is not valid boolean value', {
                type: param.type,
                value: input,
                name: param.name,
            });
        }
    }
    return encodeNumber({ type: 'uint8', name: '' }, Number(value));
}
export function decodeBool(_param, bytes) {
    const numberResult = decodeNumber({ type: 'uint8', name: '' }, bytes);
    if (numberResult.result > 1 || numberResult.result < 0) {
        throw new AbiError('Invalid boolean value encoded', {
            boolBytes: bytes.subarray(0, WORD_SIZE),
            numberResult,
        });
    }
    return {
        result: numberResult.result === BigInt(1),
        encoded: numberResult.encoded,
        consumed: WORD_SIZE,
    };
}
//# sourceMappingURL=bool.js.map