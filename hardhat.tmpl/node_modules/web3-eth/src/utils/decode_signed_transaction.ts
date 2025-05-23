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
import {
	DataFormat,
	HexStringBytes,
	SignedTransactionInfoAPI,
	TransactionSignedAPI,
} from 'web3-types';
import { bytesToHex, format, hexToBytes, keccak256 } from 'web3-utils';
import { TransactionFactory } from 'web3-eth-accounts';
import { detectRawTransactionType } from './detect_transaction_type.js';
import { formatTransaction } from './format_transaction.js';
import { type CustomTransactionSchema } from '../types.js';

/**
 * Decodes an [RLP](https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/#top) encoded transaction.
 *
 * @param encodedSignedTransaction The RLP encoded transaction.
 * @param returnFormat ({@link DataFormat} Specifies how the return data should be formatted.
 * @returns {@link SignedTransactionInfoAPI}, an object containing the RLP encoded signed transaction (accessed via the `raw` property) and the signed transaction object (accessed via the `tx` property).
 */
export function decodeSignedTransaction<ReturnFormat extends DataFormat>(
	encodedSignedTransaction: HexStringBytes,
	returnFormat: ReturnFormat,
	options: { fillInputAndData?: boolean; transactionSchema?: CustomTransactionSchema } = {
		fillInputAndData: false,
	},
): SignedTransactionInfoAPI {
	return {
		raw: format({ format: 'bytes' }, encodedSignedTransaction, returnFormat),
		tx: formatTransaction(
			{
				...TransactionFactory.fromSerializedData(
					hexToBytes(encodedSignedTransaction),
				).toJSON(),
				hash: bytesToHex(keccak256(hexToBytes(encodedSignedTransaction))),
				type: detectRawTransactionType(hexToBytes(encodedSignedTransaction)),
			} as TransactionSignedAPI,
			returnFormat,
			{
				fillInputAndData: options.fillInputAndData,
				transactionSchema: options.transactionSchema,
			},
		),
	};
}
