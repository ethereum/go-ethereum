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
 * @file strip-hex-prefix-test.ts
 * @author Alejandro Cavallero <acavallero@iovlabs.org>
 * @date 2019
 */

import BN = require('bn.js');
import {stripHexPrefix} from 'web3-utils';

// $ExpectType string
stripHexPrefix('0xd1220a0cf47c7b9be7a2e6ba89f429762e7b9adb');

// $ExpectError
stripHexPrefix(123);
// $ExpectError
stripHexPrefix(new BN(3));
// $ExpectError
stripHexPrefix(['string']);
// $ExpectError
stripHexPrefix([4]);
// $ExpectError
stripHexPrefix({});
// $ExpectError
stripHexPrefix(true);
// $ExpectError
stripHexPrefix(null);
// $ExpectError
stripHexPrefix(undefined);
