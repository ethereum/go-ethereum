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

import { format } from 'web3-utils';
import { Web3Context } from 'web3-core';
import { DataFormat, Web3NetAPI } from 'web3-types';
import { netRpcMethods } from 'web3-rpc-methods';

export async function getId<ReturnFormat extends DataFormat>(
	web3Context: Web3Context<Web3NetAPI>,
	returnFormat: ReturnFormat,
) {
	const response = await netRpcMethods.getId(web3Context.requestManager);

	return format({ format: 'uint' }, response as unknown as number, returnFormat);
}

export async function getPeerCount<ReturnFormat extends DataFormat>(
	web3Context: Web3Context<Web3NetAPI>,
	returnFormat: ReturnFormat,
) {
	const response = await netRpcMethods.getPeerCount(web3Context.requestManager);

	// Data returned is number in hex format
	return format({ format: 'uint' }, response as unknown as number, returnFormat);
}

export const isListening = async (web3Context: Web3Context<Web3NetAPI>) =>
	netRpcMethods.isListening(web3Context.requestManager);
