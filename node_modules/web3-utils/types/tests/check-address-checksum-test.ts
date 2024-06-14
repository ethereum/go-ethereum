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
 * @file check-address-checksum-test.ts
 * @author Josh Stevens <joshstevens19@hotmail.co.uk>
 * @date 2018
 */

import BN = require('bn.js');
import {checkAddressChecksum} from 'web3-utils';

// $ExpectType boolean
checkAddressChecksum('0x8ee7f17bb3f88b01247c21ab6603880b64ae53e811f5e01138822e558cf1ab51');

// $ExpectError
checkAddressChecksum('0xFb6916095CA1dF60bb79CE92ce3Ea74C37c5D359', 31);
// $ExpectError
checkAddressChecksum('0xFb6916095CA1dF60bb79CE92ce3Ea74C37c5D359', undefined);
// $ExpectError
checkAddressChecksum([4]);
// $ExpectError
checkAddressChecksum(['string']);
// $ExpectError
checkAddressChecksum(345);
// $ExpectError
checkAddressChecksum(new BN(3));
// $ExpectError
checkAddressChecksum({});
// $ExpectError
checkAddressChecksum(true);
// $ExpectError
checkAddressChecksum(null);
// $ExpectError
checkAddressChecksum(undefined);
// $ExpectError
checkAddressChecksum('0xd1220a0cf47c7b9be7a2e6ba89f429762e7b9adb', 'string');
// $ExpectError
checkAddressChecksum('0xd1220a0cf47c7b9be7a2e6ba89f429762e7b9adb', [4]);
// $ExpectError
checkAddressChecksum('0xd1220a0cf47c7b9be7a2e6ba89f429762e7b9adb', new BN(3));
// $ExpectError
checkAddressChecksum('0xd1220a0cf47c7b9be7a2e6ba89f429762e7b9adb', {});
// $ExpectError
checkAddressChecksum('0xd1220a0cf47c7b9be7a2e6ba89f429762e7b9adb', true);
// $ExpectError
checkAddressChecksum('0xd1220a0cf47c7b9be7a2e6ba89f429762e7b9adb', null);
