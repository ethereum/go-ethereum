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
 * @file from-wei-test.ts
 * @author Josh Stevens <joshstevens19@hotmail.co.uk>
 * @date 2018
 */

import BN = require('bn.js');
import {fromWei} from 'web3-utils';

const bigNumber = new BN(3);

// $ExpectType string
fromWei(bigNumber);
// $ExpectType string
fromWei('1');
// $ExpectType string
fromWei(bigNumber, 'ether');
// $ExpectType string
fromWei('1', 'ether');

// $ExpectError
fromWei(232);
// $ExpectError
fromWei(['string']);
// $ExpectError
fromWei([4]);
// $ExpectError
fromWei({});
// $ExpectError
fromWei(true);
// $ExpectError
fromWei(null);
// $ExpectError
fromWei(undefined);
// $ExpectError
fromWei(new BN(3), 'blah');
