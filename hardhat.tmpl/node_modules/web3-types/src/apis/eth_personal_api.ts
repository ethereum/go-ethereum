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
import { Address, Transaction } from '../eth_types.js';
import { HexString } from '../primitives_types.js';

export type EthPersonalAPI = {
	personal_listAccounts: () => Address[];
	personal_newAccount: (password: string) => Address;
	personal_unlockAccount: (address: Address, password: string, unlockDuration: number) => boolean;
	personal_lockAccount: (address: Address) => boolean;
	personal_importRawKey: (keyData: HexString, passphrase: string) => HexString;
	personal_sendTransaction: (tx: Transaction, passphrase: string) => HexString;
	personal_signTransaction: (tx: Transaction, passphrase: string) => HexString;
	personal_sign: (data: HexString, address: Address, passphrase: string) => HexString;
	personal_ecRecover: (signedData: HexString, signature: HexString) => Address;
};
