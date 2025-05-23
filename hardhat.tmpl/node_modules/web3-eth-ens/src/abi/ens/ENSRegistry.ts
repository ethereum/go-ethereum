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

// https://github.com/ensdomains/ens-contracts/blob/master/contracts/registry/ENSRegistry.sol
export const ENSRegistryAbi = [
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
				internalType: 'bytes32',
				name: 'label',
				type: 'bytes32',
			},
			{
				indexed: false,
				internalType: 'address',
				name: 'owner',
				type: 'address',
			},
		],
		name: 'NewOwner',
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
				internalType: 'address',
				name: 'resolver',
				type: 'address',
			},
		],
		name: 'NewResolver',
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
				internalType: 'address',
				name: 'owner',
				type: 'address',
			},
		],
		name: 'Transfer',
		type: 'event',
	},
	{
		inputs: [
			{
				internalType: 'address',
				name: 'owner',
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
		name: 'owner',
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
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
		],
		name: 'recordExists',
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
		name: 'resolver',
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
				internalType: 'bytes32',
				name: 'node',
				type: 'bytes32',
			},
		],
		name: 'ttl',
		outputs: [
			{
				internalType: 'uint64',
				name: '',
				type: 'uint64',
			},
		],
		stateMutability: 'view',
		type: 'function',
	},
] as const;
