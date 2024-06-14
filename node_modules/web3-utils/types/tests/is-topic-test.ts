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
 * @file is-topic-test.ts
 * @author Josh Stevens <joshstevens19@hotmail.co.uk>
 * @date 2018
 */

import BN = require('bn.js');
import {isTopic} from 'web3-utils';

// $ExpectType boolean
isTopic('0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef');

// $ExpectError
isTopic(656);
// $ExpectError
isTopic(new BN(3));
// $ExpectError
isTopic(['string']);
// $ExpectError
isTopic([4]);
// $ExpectError
isTopic({});
// $ExpectError
isTopic(true);
// $ExpectError
isTopic(null);
// $ExpectError
isTopic(undefined);
