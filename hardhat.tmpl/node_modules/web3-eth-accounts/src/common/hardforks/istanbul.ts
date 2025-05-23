export default {
	name: 'istanbul',
	comment: 'HF targeted for December 2019 following the Constantinople/Petersburg HF',
	url: 'https://eips.ethereum.org/EIPS/eip-1679',
	status: 'Final',
	gasConfig: {},
	gasPrices: {
		blake2Round: {
			v: 1,
			d: 'Gas cost per round for the Blake2 F precompile',
		},
		ecAdd: {
			v: 150,
			d: 'Gas costs for curve addition precompile',
		},
		ecMul: {
			v: 6000,
			d: 'Gas costs for curve multiplication precompile',
		},
		ecPairing: {
			v: 45000,
			d: 'Base gas costs for curve pairing precompile',
		},
		ecPairingWord: {
			v: 34000,
			d: 'Gas costs regarding curve pairing precompile input length',
		},
		txDataNonZero: {
			v: 16,
			d: 'Per byte of data attached to a transaction that is not equal to zero. NOTE: Not payable on data of calls between transactions',
		},
		sstoreSentryGasEIP2200: {
			v: 2300,
			d: 'Minimum gas required to be present for an SSTORE call, not consumed',
		},
		sstoreNoopGasEIP2200: {
			v: 800,
			d: "Once per SSTORE operation if the value doesn't change",
		},
		sstoreDirtyGasEIP2200: {
			v: 800,
			d: 'Once per SSTORE operation if a dirty value is changed',
		},
		sstoreInitGasEIP2200: {
			v: 20000,
			d: 'Once per SSTORE operation from clean zero to non-zero',
		},
		sstoreInitRefundEIP2200: {
			v: 19200,
			d: 'Once per SSTORE operation for resetting to the original zero value',
		},
		sstoreCleanGasEIP2200: {
			v: 5000,
			d: 'Once per SSTORE operation from clean non-zero to something else',
		},
		sstoreCleanRefundEIP2200: {
			v: 4200,
			d: 'Once per SSTORE operation for resetting to the original non-zero value',
		},
		sstoreClearRefundEIP2200: {
			v: 15000,
			d: 'Once per SSTORE operation for clearing an originally existing storage slot',
		},
		balance: {
			v: 700,
			d: 'Base fee of the BALANCE opcode',
		},
		extcodehash: {
			v: 700,
			d: 'Base fee of the EXTCODEHASH opcode',
		},
		chainid: {
			v: 2,
			d: 'Base fee of the CHAINID opcode',
		},
		selfbalance: {
			v: 5,
			d: 'Base fee of the SELFBALANCE opcode',
		},
		sload: {
			v: 800,
			d: 'Base fee of the SLOAD opcode',
		},
	},
	vm: {},
	pow: {},
};
