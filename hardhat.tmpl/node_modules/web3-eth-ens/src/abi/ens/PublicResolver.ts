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

// https://github.com/ensdomains/ens-contracts/blob/master/contracts/resolvers/PublicResolver.sol
export const PublicResolverAbi = [
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				indexed: false,
				internalType: 'address',
				name: 'a',
				type: 'address',
			},
		],
		name: 'AddrChanged',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				indexed: false,
				internalType: 'uint256',
				name: 'coinType',
				type: 'uint256',
			},
			{
				indexed: false,
				internalType: 'bytes',
				name: 'newAddress',
				type: 'bytes',
			},
		],
		name: 'AddressChanged',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'address',
				name: 'owner',
				type: 'address',
			},
			{
				indexed: true,
				internalType: 'address',
				name: 'operator',
				type: 'address',
			},
			{
				indexed: false,
				internalType: 'bool',
				name: 'approved',
				type: 'bool',
			},
		],
		name: 'ApprovalForAll',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				indexed: false,
				internalType: 'bytes',
				name: 'hash',
				type: 'bytes',
			},
		],
		name: 'ContenthashChanged',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				indexed: false,
				internalType: 'bytes',
				name: 'name',
				type: 'bytes',
			},
			{
				indexed: false,
				internalType: 'uint16',
				name: 'resource',
				type: 'uint16',
			},
			{
				indexed: false,
				internalType: 'bytes',
				name: 'record',
				type: 'bytes',
			},
		],
		name: 'DNSRecordChanged',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				indexed: false,
				internalType: 'bytes',
				name: 'name',
				type: 'bytes',
			},
			{
				indexed: false,
				internalType: 'uint16',
				name: 'resource',
				type: 'uint16',
			},
		],
		name: 'DNSRecordDeleted',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
		],
		name: 'DNSZoneCleared',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				indexed: false,
				internalType: 'bytes',
				name: 'lastzonehash',
				type: 'bytes',
			},
			{
				indexed: false,
				internalType: 'bytes',
				name: 'zonehash',
				type: 'bytes',
			},
		],
		name: 'DNSZonehashChanged',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				indexed: true,
				internalType: 'bytes4',
				name: 'interfaceID',
				type: 'bytes4',
			},
			{
				indexed: false,
				internalType: 'address',
				name: 'implementer',
				type: 'address',
			},
		],
		name: 'InterfaceChanged',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				indexed: false,
				internalType: 'string',
				name: 'name',
				type: 'string',
			},
		],
		name: 'NameChanged',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				indexed: false,
				internalType: 'bytes32',
				name: 'x',
				type: 'bytes32',
			},
			{
				indexed: false,
				internalType: 'bytes32',
				name: 'y',
				type: 'bytes32',
			},
		],
		name: 'PubkeyChanged',
		type: 'event',
	},
	{
		anonymous: false,
		inputs: [
			{
				indexed: true,
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				indexed: true,
				internalType: 'string',
				name: 'indexedKey',
				type: 'string',
			},
			{
				indexed: false,
				internalType: 'string',
				name: 'key',
				type: 'string',
			},
		],
		name: 'TextChanged',
		type: 'event',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				internalType: 'uint256',
				name: 'contentTypes',
				type: 'uint256',
			},
		],
		name: 'ABI',
		outputs: [
			{
				internalType: 'uint256',
				name: '',
				type: 'uint256',
			},
			{
				internalType: 'bytes',
				name: '',
				type: 'bytes',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
		],
		name: 'addr',
		outputs: [
			{
				internalType: 'address payable',
				name: '',
				type: 'address',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				internalType: 'uint256',
				name: 'coinType',
				type: 'uint256',
			},
		],
		name: 'addr',
		outputs: [
			{
				internalType: 'bytes',
				name: '',
				type: 'bytes',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
		],
		name: 'contenthash',
		outputs: [
			{
				internalType: 'bytes',
				name: '',
				type: 'bytes',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				internalType: 'bytes32',
				name: 'name',
				type: 'bytes32',
			},
			{
				internalType: 'uint16',
				name: 'resource',
				type: 'uint16',
			},
		],
		name: 'dnsRecord',
		outputs: [
			{
				internalType: 'bytes',
				name: '',
				type: 'bytes',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				internalType: 'bytes32',
				name: 'name',
				type: 'bytes32',
			},
		],
		name: 'hasDNSRecords',
		outputs: [
			{
				internalType: 'bool',
				name: '',
				type: 'bool',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				internalType: 'bytes4',
				name: 'interfaceID',
				type: 'bytes4',
			},
		],
		name: 'interfaceImplementer',
		outputs: [
			{
				internalType: 'address',
				name: '',
				type: 'address',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'address',
				name: 'account',
				type: 'address',
			},
			{
				internalType: 'address',
				name: 'operator',
				type: 'address',
			},
		],
		name: 'isApprovedForAll',
		outputs: [
			{
				internalType: 'bool',
				name: '',
				type: 'bool',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
		],
		name: 'name',
		outputs: [
			{
				internalType: 'string',
				name: '',
				type: 'string',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
		],
		name: 'pubkey',
		outputs: [
			{
				internalType: 'bytes32',
				name: 'x',
				type: 'bytes32',
			},
			{
				internalType: 'bytes32',
				name: 'y',
				type: 'bytes32',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes4',
				name: 'interfaceID',
				type: 'bytes4',
			},
		],
		name: 'supportsInterface',
		outputs: [
			{
				internalType: 'bool',
				name: '',
				type: 'bool',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				internalType: 'string',
				name: 'key',
				type: 'string',
			},
		],
		name: 'text',
		outputs: [
			{
				internalType: 'string',
				name: '',
				type: 'string',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
		],
		name: 'zonehash',
		outputs: [
			{
				internalType: 'bytes',
				name: '',
				type: 'bytes',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
	{
		inputs: [
			{
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
			{
				internalType: 'address',
				name: 'a',
				type: 'address',
			},
		],
		name: 'setAddr',
		outputs: [],
		stateMutability: 'nonpayable',
		type: 'function',
	},
] as const;
