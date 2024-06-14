import { Hardfork } from './enums.js'

import type { EIPConfig } from './types.js'

type EIPsDict = {
  [key: string]: EIPConfig
}

enum Status {
  Draft = 'draft',
  Review = 'review',
  Final = 'final',
}

export const EIPs: EIPsDict = {
  1153: {
    comment: 'Transient storage opcodes',
    url: 'https://eips.ethereum.org/EIPS/eip-1153',
    status: Status.Review,
    minimumHardfork: Hardfork.Chainstart,
    requiredEIPs: [],
    gasPrices: {
      tstore: {
        v: 100,
        d: 'Base fee of the TSTORE opcode',
      },
      tload: {
        v: 100,
        d: 'Base fee of the TLOAD opcode',
      },
    },
  },
  1559: {
    comment: 'Fee market change for ETH 1.0 chain',
    url: 'https://eips.ethereum.org/EIPS/eip-1559',
    status: Status.Final,
    minimumHardfork: Hardfork.Berlin,
    requiredEIPs: [2930],
    gasConfig: {
      baseFeeMaxChangeDenominator: {
        v: 8,
        d: 'Maximum base fee change denominator',
      },
      elasticityMultiplier: {
        v: 2,
        d: 'Maximum block gas target elasticity',
      },
      initialBaseFee: {
        v: 1000000000,
        d: 'Initial base fee on first EIP1559 block',
      },
    },
  },
  2315: {
    comment: 'Simple subroutines for the EVM',
    url: 'https://eips.ethereum.org/EIPS/eip-2315',
    status: Status.Draft,
    minimumHardfork: Hardfork.Istanbul,
    requiredEIPs: [],
    gasPrices: {
      beginsub: {
        v: 2,
        d: 'Base fee of the BEGINSUB opcode',
      },
      returnsub: {
        v: 5,
        d: 'Base fee of the RETURNSUB opcode',
      },
      jumpsub: {
        v: 10,
        d: 'Base fee of the JUMPSUB opcode',
      },
    },
  },
  2565: {
    comment: 'ModExp gas cost',
    url: 'https://eips.ethereum.org/EIPS/eip-2565',
    status: Status.Final,
    minimumHardfork: Hardfork.Byzantium,
    requiredEIPs: [],
    gasPrices: {
      modexpGquaddivisor: {
        v: 3,
        d: 'Gquaddivisor from modexp precompile for gas calculation',
      },
    },
  },
  2718: {
    comment: 'Typed Transaction Envelope',
    url: 'https://eips.ethereum.org/EIPS/eip-2718',
    status: Status.Final,
    minimumHardfork: Hardfork.Chainstart,
    requiredEIPs: [],
  },
  2929: {
    comment: 'Gas cost increases for state access opcodes',
    url: 'https://eips.ethereum.org/EIPS/eip-2929',
    status: Status.Final,
    minimumHardfork: Hardfork.Chainstart,
    requiredEIPs: [],
    gasPrices: {
      coldsload: {
        v: 2100,
        d: 'Gas cost of the first read of storage from a given location (per transaction)',
      },
      coldaccountaccess: {
        v: 2600,
        d: 'Gas cost of the first read of a given address (per transaction)',
      },
      warmstorageread: {
        v: 100,
        d: "Gas cost of reading storage locations which have already loaded 'cold'",
      },
      sstoreCleanGasEIP2200: {
        v: 2900,
        d: 'Once per SSTORE operation from clean non-zero to something else',
      },
      sstoreNoopGasEIP2200: {
        v: 100,
        d: "Once per SSTORE operation if the value doesn't change",
      },
      sstoreDirtyGasEIP2200: {
        v: 100,
        d: 'Once per SSTORE operation if a dirty value is changed',
      },
      sstoreInitRefundEIP2200: {
        v: 19900,
        d: 'Once per SSTORE operation for resetting to the original zero value',
      },
      sstoreCleanRefundEIP2200: {
        v: 4900,
        d: 'Once per SSTORE operation for resetting to the original non-zero value',
      },
      call: {
        v: 0,
        d: 'Base fee of the CALL opcode',
      },
      callcode: {
        v: 0,
        d: 'Base fee of the CALLCODE opcode',
      },
      delegatecall: {
        v: 0,
        d: 'Base fee of the DELEGATECALL opcode',
      },
      staticcall: {
        v: 0,
        d: 'Base fee of the STATICCALL opcode',
      },
      balance: {
        v: 0,
        d: 'Base fee of the BALANCE opcode',
      },
      extcodesize: {
        v: 0,
        d: 'Base fee of the EXTCODESIZE opcode',
      },
      extcodecopy: {
        v: 0,
        d: 'Base fee of the EXTCODECOPY opcode',
      },
      extcodehash: {
        v: 0,
        d: 'Base fee of the EXTCODEHASH opcode',
      },
      sload: {
        v: 0,
        d: 'Base fee of the SLOAD opcode',
      },
      sstore: {
        v: 0,
        d: 'Base fee of the SSTORE opcode',
      },
    },
  },
  2930: {
    comment: 'Optional access lists',
    url: 'https://eips.ethereum.org/EIPS/eip-2930',
    status: Status.Final,
    minimumHardfork: Hardfork.Istanbul,
    requiredEIPs: [2718, 2929],
    gasPrices: {
      accessListStorageKeyCost: {
        v: 1900,
        d: 'Gas cost per storage key in an Access List transaction',
      },
      accessListAddressCost: {
        v: 2400,
        d: 'Gas cost per storage key in an Access List transaction',
      },
    },
  },
  3074: {
    comment: 'AUTH and AUTHCALL opcodes',
    url: 'https://eips.ethereum.org/EIPS/eip-3074',
    status: Status.Review,
    minimumHardfork: Hardfork.London,
    requiredEIPs: [],
    gasPrices: {
      auth: {
        v: 3100,
        d: 'Gas cost of the AUTH opcode',
      },
      authcall: {
        v: 0,
        d: 'Gas cost of the AUTHCALL opcode',
      },
      authcallValueTransfer: {
        v: 6700,
        d: 'Paid for CALL when the value transfer is non-zero',
      },
    },
  },
  3198: {
    comment: 'BASEFEE opcode',
    url: 'https://eips.ethereum.org/EIPS/eip-3198',
    status: Status.Final,
    minimumHardfork: Hardfork.London,
    requiredEIPs: [],
    gasPrices: {
      basefee: {
        v: 2,
        d: 'Gas cost of the BASEFEE opcode',
      },
    },
  },
  3529: {
    comment: 'Reduction in refunds',
    url: 'https://eips.ethereum.org/EIPS/eip-3529',
    status: Status.Final,
    minimumHardfork: Hardfork.Berlin,
    requiredEIPs: [2929],
    gasConfig: {
      maxRefundQuotient: {
        v: 5,
        d: 'Maximum refund quotient; max tx refund is min(tx.gasUsed/maxRefundQuotient, tx.gasRefund)',
      },
    },
    gasPrices: {
      selfdestructRefund: {
        v: 0,
        d: 'Refunded following a selfdestruct operation',
      },
      sstoreClearRefundEIP2200: {
        v: 4800,
        d: 'Once per SSTORE operation for clearing an originally existing storage slot',
      },
    },
  },
  3540: {
    comment: 'EVM Object Format (EOF) v1',
    url: 'https://eips.ethereum.org/EIPS/eip-3540',
    status: Status.Review,
    minimumHardfork: Hardfork.London,
    requiredEIPs: [3541],
  },
  3541: {
    comment: 'Reject new contracts starting with the 0xEF byte',
    url: 'https://eips.ethereum.org/EIPS/eip-3541',
    status: Status.Final,
    minimumHardfork: Hardfork.Berlin,
    requiredEIPs: [],
  },
  3554: {
    comment: 'Difficulty Bomb Delay to December 1st 2021',
    url: 'https://eips.ethereum.org/EIPS/eip-3554',
    status: Status.Final,
    minimumHardfork: Hardfork.MuirGlacier,
    requiredEIPs: [],
    pow: {
      difficultyBombDelay: {
        v: 9500000,
        d: 'the amount of blocks to delay the difficulty bomb with',
      },
    },
  },
  3607: {
    comment: 'Reject transactions from senders with deployed code',
    url: 'https://eips.ethereum.org/EIPS/eip-3607',
    status: Status.Final,
    minimumHardfork: Hardfork.Chainstart,
    requiredEIPs: [],
  },
  3651: {
    comment: 'Warm COINBASE',
    url: 'https://eips.ethereum.org/EIPS/eip-3651',
    status: Status.Review,
    minimumHardfork: Hardfork.London,
    requiredEIPs: [2929],
  },
  3670: {
    comment: 'EOF - Code Validation',
    url: 'https://eips.ethereum.org/EIPS/eip-3670',
    status: 'Review',
    minimumHardfork: Hardfork.London,
    requiredEIPs: [3540],
    gasConfig: {},
    gasPrices: {},
    vm: {},
    pow: {},
  },
  3675: {
    comment: 'Upgrade consensus to Proof-of-Stake',
    url: 'https://eips.ethereum.org/EIPS/eip-3675',
    status: Status.Final,
    minimumHardfork: Hardfork.London,
    requiredEIPs: [],
  },
  3855: {
    comment: 'PUSH0 instruction',
    url: 'https://eips.ethereum.org/EIPS/eip-3855',
    status: Status.Review,
    minimumHardfork: Hardfork.Chainstart,
    requiredEIPs: [],
    gasPrices: {
      push0: {
        v: 2,
        d: 'Base fee of the PUSH0 opcode',
      },
    },
  },
  3860: {
    comment: 'Limit and meter initcode',
    url: 'https://eips.ethereum.org/EIPS/eip-3860',
    status: Status.Review,
    minimumHardfork: Hardfork.SpuriousDragon,
    requiredEIPs: [],
    gasPrices: {
      initCodeWordCost: {
        v: 2,
        d: 'Gas to pay for each word (32 bytes) of initcode when creating a contract',
      },
    },
    vm: {
      maxInitCodeSize: {
        v: 49152,
        d: 'Maximum length of initialization code when creating a contract',
      },
    },
  },
  4345: {
    comment: 'Difficulty Bomb Delay to June 2022',
    url: 'https://eips.ethereum.org/EIPS/eip-4345',
    status: Status.Final,
    minimumHardfork: Hardfork.London,
    requiredEIPs: [],
    pow: {
      difficultyBombDelay: {
        v: 10700000,
        d: 'the amount of blocks to delay the difficulty bomb with',
      },
    },
  },
  4399: {
    comment: 'Supplant DIFFICULTY opcode with PREVRANDAO',
    url: 'https://eips.ethereum.org/EIPS/eip-4399',
    status: Status.Review,
    minimumHardfork: Hardfork.London,
    requiredEIPs: [],
    gasPrices: {
      prevrandao: {
        v: 2,
        d: 'Base fee of the PREVRANDAO opcode (previously DIFFICULTY)',
      },
    },
  },
  4788: {
    comment: 'Beacon block root in the EVM',
    url: 'https://eips.ethereum.org/EIPS/eip-4788',
    status: Status.Draft,
    minimumHardfork: Hardfork.Cancun,
    requiredEIPs: [],
    gasPrices: {},
    vm: {
      historicalRootsLength: {
        v: 8191,
        d: 'The modulo parameter of the beaconroot ring buffer in the beaconroot statefull precompile',
      },
    },
  },
  4844: {
    comment: 'Shard Blob Transactions',
    url: 'https://eips.ethereum.org/EIPS/eip-4844',
    status: Status.Draft,
    minimumHardfork: Hardfork.Paris,
    requiredEIPs: [1559, 2718, 2930, 4895],
    gasConfig: {
      blobGasPerBlob: {
        v: 131072,
        d: 'The base fee for blob gas per blob',
      },
      targetBlobGasPerBlock: {
        v: 393216,
        d: 'The target blob gas consumed per block',
      },
      maxblobGasPerBlock: {
        v: 786432,
        d: 'The max blob gas allowable per block',
      },
      blobGasPriceUpdateFraction: {
        v: 3338477,
        d: 'The denominator used in the exponential when calculating a blob gas price',
      },
    },
    gasPrices: {
      simpleGasPerBlob: {
        v: 12000,
        d: 'The basic gas fee for each blob',
      },
      minBlobGasPrice: {
        v: 1,
        d: 'The minimum fee per blob gas',
      },
      kzgPointEvaluationGasPrecompilePrice: {
        v: 50000,
        d: 'The fee associated with the point evaluation precompile',
      },
      blobhash: {
        v: 3,
        d: 'Base fee of the BLOBHASH opcode',
      },
    },
    sharding: {
      blobCommitmentVersionKzg: {
        v: 1,
        d: 'The number indicated a versioned hash is a KZG commitment',
      },
      fieldElementsPerBlob: {
        v: 4096,
        d: 'The number of field elements allowed per blob',
      },
    },
  },
  4895: {
    comment: 'Beacon chain push withdrawals as operations',
    url: 'https://eips.ethereum.org/EIPS/eip-4895',
    status: Status.Review,
    minimumHardfork: Hardfork.Paris,
    requiredEIPs: [],
  },
  5133: {
    comment: 'Delaying Difficulty Bomb to mid-September 2022',
    url: 'https://eips.ethereum.org/EIPS/eip-5133',
    status: Status.Draft,
    minimumHardfork: Hardfork.GrayGlacier,
    requiredEIPs: [],
    pow: {
      difficultyBombDelay: {
        v: 11400000,
        d: 'the amount of blocks to delay the difficulty bomb with',
      },
    },
  },
  5656: {
    comment: 'MCOPY - Memory copying instruction',
    url: 'https://eips.ethereum.org/EIPS/eip-5656',
    status: Status.Draft,
    minimumHardfork: Hardfork.Shanghai,
    requiredEIPs: [],
    gasPrices: {
      mcopy: {
        v: 3,
        d: 'Base fee of the MCOPY opcode',
      },
    },
  },
  6780: {
    comment: 'SELFDESTRUCT only in same transaction',
    url: 'https://eips.ethereum.org/EIPS/eip-6780',
    status: Status.Draft,
    minimumHardfork: Hardfork.London,
    requiredEIPs: [],
  },
  6800: {
    comment: 'Ethereum state using a unified verkle tree (experimental)',
    url: 'https://github.com/ethereum/EIPs/pull/6800',
    status: Status.Draft,
    minimumHardfork: Hardfork.London,
    requiredEIPs: [],
    gasConfig: {},
    gasPrices: {
      tx: {
        v: 34300,
        d: 'Per transaction. NOTE: Not payable on data of calls between transactions',
      },
    },
    vm: {},
    pow: {},
  },
  7516: {
    comment: 'BLOBBASEFEE opcode',
    url: 'https://eips.ethereum.org/EIPS/eip-7516',
    status: Status.Draft,
    minimumHardfork: Hardfork.Paris,
    requiredEIPs: [4844],
    gasPrices: {
      blobbasefee: {
        v: 2,
        d: 'Gas cost of the BLOBBASEFEE opcode',
      },
    },
  },
}
