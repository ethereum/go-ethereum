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
/*
 * this variable contains the precalculated limits for all the numbers for uint and int types
 */
export const numberLimits = new Map();
let base = BigInt(256); // 2 ^ 8 = 256
for (let i = 8; i <= 256; i += 8) {
    numberLimits.set(`uint${i}`, {
        min: BigInt(0),
        max: base - BigInt(1),
    });
    numberLimits.set(`int${i}`, {
        min: -base / BigInt(2),
        max: base / BigInt(2) - BigInt(1),
    });
    base *= BigInt(256);
}
// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
numberLimits.set(`int`, numberLimits.get('int256'));
// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
numberLimits.set(`uint`, numberLimits.get('uint256'));
//# sourceMappingURL=numbersLimits.js.map