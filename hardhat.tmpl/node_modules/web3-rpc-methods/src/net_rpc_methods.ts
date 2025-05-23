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
import { Web3RequestManager } from 'web3-core';
import { Web3NetAPI } from 'web3-types';

export async function getId(requestManager: Web3RequestManager<Web3NetAPI>) {
	return requestManager.send({
		method: 'net_version',
		params: [],
	});
}

export async function getPeerCount(requestManager: Web3RequestManager<Web3NetAPI>) {
	return requestManager.send({
		method: 'net_peerCount',
		params: [],
	});
}

export async function isListening(requestManager: Web3RequestManager<Web3NetAPI>) {
	return requestManager.send({
		method: 'net_listening',
		params: [],
	});
}
