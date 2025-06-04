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
 * @file is-user-ethereum-address-in-bloom.ts
 * @author Josh Stevens <joshstevens19@hotmail.co.uk>
 * @date 2019
 */

import { isUserEthereumAddressInBloom } from 'web3-utils';

// $ExpectType boolean
isUserEthereumAddressInBloom('0x08200081a06415012858022200cc48143008908c0000824e5405b41520795989024800380a8d4b198910b422b231086c3a62cc402e2573070306f180446440ad401016c3e30781115844d028c89028008a12240c0a2c184c0425b90d7af0530002f981221aa565809132000818c82805023a132a25150400010530ba0080420a10a137054454021882505080a6b6841082d84151010400ba8100c8802d440d060388084052c1300105a0868410648a40540c0f0460e190400807008914361118000a5202e94445ccc088311050052c8002807205212a090d90ba428030266024a910644b1042011aaae05391cc2094c45226400000380880241282ce4e12518c', '0x494bfa3a4576ba6cfe835b0deb78834f0c3e3994');
