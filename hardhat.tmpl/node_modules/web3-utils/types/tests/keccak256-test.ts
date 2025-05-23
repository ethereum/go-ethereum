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
 * @file keccak256-test.ts
 * @author Josh Stevens <joshstevens19@hotmail.co.uk>
 * @date 2018
 */

import BN = require('bn.js');
import {keccak256} from 'web3-utils';

// $ExpectType string
keccak256('234');
// $ExpectType string
keccak256(new BN(3));

// $ExpectError
keccak256(['string']);
// $ExpectError
keccak256(234);
// $ExpectError
keccak256([4]);
// $ExpectError
keccak256({});
// $ExpectError
keccak256(true);
// $ExpectError
keccak256(null);
// $ExpectError
keccak256(undefined);
