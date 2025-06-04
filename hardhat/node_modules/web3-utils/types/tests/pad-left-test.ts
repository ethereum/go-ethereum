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
/**
 * @file pad-left-test.ts
 * @author Josh Stevens <joshstevens19@hotmail.co.uk>
 * @date 2018
 */

import BN = require('bn.js');
import {padLeft} from 'web3-utils';

const bigNumber = new BN(3);

// $ExpectType string
padLeft('0x3456ff', 20);
// $ExpectType string
padLeft(0x3456ff, 20);
// $ExpectType string
padLeft('Hello', 20, 'x');

// $ExpectError
padLeft(bigNumber, 20);
// $ExpectError
padLeft(['string'], 20);
// $ExpectError
padLeft([4], 20);
// $ExpectError
padLeft({}, 20);
// $ExpectError
padLeft(true, 20);
// $ExpectError
padLeft(null, 20);
// $ExpectError
padLeft(undefined, 20);
// $ExpectError
padLeft('0x3456ff', bigNumber);
// $ExpectError
padLeft('0x3456ff', ['string']);
// $ExpectError
padLeft('0x3456ff', [4]);
// $ExpectError
padLeft('0x3456ff', {});
// $ExpectError
padLeft('0x3456ff', true);
// $ExpectError
padLeft('0x3456ff', null);
// $ExpectError
padLeft('0x3456ff', undefined);
// $ExpectError
padLeft('Hello', 20, bigNumber);
// $ExpectError
padLeft('Hello', 20, ['string']);
// $ExpectError
padLeft('Hello', 20, [4]);
// $ExpectError
padLeft('Hello', 20, {});
// $ExpectError
padLeft('Hello', 20, true);
// $ExpectError
padLeft('Hello', 20, null);
