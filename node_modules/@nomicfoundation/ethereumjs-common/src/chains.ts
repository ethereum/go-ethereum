import type { ChainConfig } from './types.js'

type ChainsDict = {
  [key: string]: ChainConfig
}

export const chains: ChainsDict = {
  mainnet: {
    name: 'mainnet',
    chainId: 1,
    networkId: 1,
    defaultHardfork: 'shanghai',
    consensus: {
      type: 'pow',
      algorithm: 'ethash',
      ethash: {},
    },
    comment: 'The Ethereum main chain',
    url: 'https://ethstats.net/',
    genesis: {
      gasLimit: 5000,
      difficulty: 17179869184,
      nonce: '0x0000000000000042',
      extraData: '0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa',
    },
    hardforks: [
      {
        name: 'chainstart',
        block: 0,
        forkHash: '0xfc64ec04',
      },
      {
        name: 'homestead',
        block: 1150000,
        forkHash: '0x97c2c34c',
      },
      {
        name: 'dao',
        block: 1920000,
        forkHash: '0x91d1f948',
      },
      {
        name: 'tangerineWhistle',
        block: 2463000,
        forkHash: '0x7a64da13',
      },
      {
        name: 'spuriousDragon',
        block: 2675000,
        forkHash: '0x3edd5b10',
      },
      {
        name: 'byzantium',
        block: 4370000,
        forkHash: '0xa00bc324',
      },
      {
        name: 'constantinople',
        block: 7280000,
        forkHash: '0x668db0af',
      },
      {
        name: 'petersburg',
        block: 7280000,
        forkHash: '0x668db0af',
      },
      {
        name: 'istanbul',
        block: 9069000,
        forkHash: '0x879d6e30',
      },
      {
        name: 'muirGlacier',
        block: 9200000,
        forkHash: '0xe029e991',
      },
      {
        name: 'berlin',
        block: 12244000,
        forkHash: '0x0eb440f6',
      },
      {
        name: 'london',
        block: 12965000,
        forkHash: '0xb715077d',
      },
      {
        name: 'arrowGlacier',
        block: 13773000,
        forkHash: '0x20c327fc',
      },
      {
        name: 'grayGlacier',
        block: 15050000,
        forkHash: '0xf0afd0e3',
      },
      {
        // The forkHash will remain same as mergeForkIdTransition is post merge
        // terminal block: https://etherscan.io/block/15537393
        name: 'paris',
        ttd: '58750000000000000000000',
        block: 15537394,
        forkHash: '0xf0afd0e3',
      },
      {
        name: 'mergeForkIdTransition',
        block: null,
        forkHash: null,
      },
      {
        name: 'shanghai',
        block: null,
        timestamp: '1681338455',
        forkHash: '0xdce96c2d',
      },
      {
        name: 'cancun',
        block: null,
        forkHash: null,
      },
    ],
    bootstrapNodes: [
      {
        ip: '18.138.108.67',
        port: 30303,
        id: 'd860a01f9722d78051619d1e2351aba3f43f943f6f00718d1b9baa4101932a1f5011f16bb2b1bb35db20d6fe28fa0bf09636d26a87d31de9ec6203eeedb1f666',
        location: 'ap-southeast-1-001',
        comment: 'bootnode-aws-ap-southeast-1-001',
      },
      {
        ip: '3.209.45.79',
        port: 30303,
        id: '22a8232c3abc76a16ae9d6c3b164f98775fe226f0917b0ca871128a74a8e9630b458460865bab457221f1d448dd9791d24c4e5d88786180ac185df813a68d4de',
        location: 'us-east-1-001',
        comment: 'bootnode-aws-us-east-1-001',
      },
      {
        ip: '65.108.70.101',
        port: 30303,
        id: '2b252ab6a1d0f971d9722cb839a42cb81db019ba44c08754628ab4a823487071b5695317c8ccd085219c3a03af063495b2f1da8d18218da2d6a82981b45e6ffc',
        location: 'eu-west-1-001',
        comment: 'bootnode-hetzner-hel',
      },
      {
        ip: '157.90.35.166',
        port: 30303,
        id: '4aeb4ab6c14b23e2c4cfdce879c04b0748a20d8e9b59e25ded2a08143e265c6c25936e74cbc8e641e3312ca288673d91f2f93f8e277de3cfa444ecdaaf982052',
        location: 'eu-central-1-001',
        comment: 'bootnode-hetzner-fsn',
      },
    ],
    dnsNetworks: [
      'enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.mainnet.ethdisco.net',
    ],
  },
  goerli: {
    name: 'goerli',
    chainId: 5,
    networkId: 5,
    defaultHardfork: 'shanghai',
    consensus: {
      type: 'poa',
      algorithm: 'clique',
      clique: {
        period: 15,
        epoch: 30000,
      },
    },
    comment: 'Cross-client PoA test network',
    url: 'https://github.com/goerli/testnet',
    genesis: {
      timestamp: '0x5c51a607',
      gasLimit: 10485760,
      difficulty: 1,
      nonce: '0x0000000000000000',
      extraData:
        '0x22466c6578692069732061207468696e6722202d204166726900000000000000e0a2bd4258d2768837baa26a28fe71dc079f84c70000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
    },
    hardforks: [
      {
        name: 'chainstart',
        block: 0,
        forkHash: '0xa3f5ab08',
      },
      {
        name: 'homestead',
        block: 0,
        forkHash: '0xa3f5ab08',
      },
      {
        name: 'tangerineWhistle',
        block: 0,
        forkHash: '0xa3f5ab08',
      },
      {
        name: 'spuriousDragon',
        block: 0,
        forkHash: '0xa3f5ab08',
      },
      {
        name: 'byzantium',
        block: 0,
        forkHash: '0xa3f5ab08',
      },
      {
        name: 'constantinople',
        block: 0,
        forkHash: '0xa3f5ab08',
      },
      {
        name: 'petersburg',
        block: 0,
        forkHash: '0xa3f5ab08',
      },
      {
        name: 'istanbul',
        block: 1561651,
        forkHash: '0xc25efa5c',
      },
      {
        name: 'berlin',
        block: 4460644,
        forkHash: '0x757a1c47',
      },
      {
        name: 'london',
        block: 5062605,
        forkHash: '0xb8c6299d',
      },
      {
        // The forkHash will remain same as mergeForkIdTransition is post merge,
        // terminal block: https://goerli.etherscan.io/block/7382818
        name: 'paris',
        ttd: '10790000',
        block: 7382819,
        forkHash: '0xb8c6299d',
      },
      {
        name: 'mergeForkIdTransition',
        block: null,
        forkHash: null,
      },
      {
        name: 'shanghai',
        block: null,
        timestamp: '1678832736',
        forkHash: '0xf9843abf',
      },
      {
        name: 'cancun',
        block: null,
        timestamp: '1705473120',
        forkHash: '0x70cc14e2',
      },
    ],
    bootstrapNodes: [
      {
        ip: '51.141.78.53',
        port: 30303,
        id: '011f758e6552d105183b1761c5e2dea0111bc20fd5f6422bc7f91e0fabbec9a6595caf6239b37feb773dddd3f87240d99d859431891e4a642cf2a0a9e6cbb98a',
        location: '',
        comment: 'Upstream bootnode 1',
      },
      {
        ip: '13.93.54.137',
        port: 30303,
        id: '176b9417f511d05b6b2cf3e34b756cf0a7096b3094572a8f6ef4cdcb9d1f9d00683bf0f83347eebdf3b81c3521c2332086d9592802230bf528eaf606a1d9677b',
        location: '',
        comment: 'Upstream bootnode 2',
      },
      {
        ip: '94.237.54.114',
        port: 30313,
        id: '46add44b9f13965f7b9875ac6b85f016f341012d84f975377573800a863526f4da19ae2c620ec73d11591fa9510e992ecc03ad0751f53cc02f7c7ed6d55c7291',
        location: '',
        comment: 'Upstream bootnode 3',
      },
      {
        ip: '18.218.250.66',
        port: 30313,
        id: 'b5948a2d3e9d486c4d75bf32713221c2bd6cf86463302339299bd227dc2e276cd5a1c7ca4f43a0e9122fe9af884efed563bd2a1fd28661f3b5f5ad7bf1de5949',
        location: '',
        comment: 'Upstream bootnode 4',
      },
      {
        ip: '3.11.147.67',
        port: 30303,
        id: 'a61215641fb8714a373c80edbfa0ea8878243193f57c96eeb44d0bc019ef295abd4e044fd619bfc4c59731a73fb79afe84e9ab6da0c743ceb479cbb6d263fa91',
        location: '',
        comment: 'Ethereum Foundation bootnode',
      },
      {
        ip: '51.15.116.226',
        port: 30303,
        id: 'a869b02cec167211fb4815a82941db2e7ed2936fd90e78619c53eb17753fcf0207463e3419c264e2a1dd8786de0df7e68cf99571ab8aeb7c4e51367ef186b1dd',
        location: '',
        comment: 'Goerli Initiative bootnode',
      },
      {
        ip: '51.15.119.157',
        port: 30303,
        id: '807b37ee4816ecf407e9112224494b74dd5933625f655962d892f2f0f02d7fbbb3e2a94cf87a96609526f30c998fd71e93e2f53015c558ffc8b03eceaf30ee33',
        location: '',
        comment: 'Goerli Initiative bootnode',
      },
      {
        ip: '51.15.119.157',
        port: 40303,
        id: 'a59e33ccd2b3e52d578f1fbd70c6f9babda2650f0760d6ff3b37742fdcdfdb3defba5d56d315b40c46b70198c7621e63ffa3f987389c7118634b0fefbbdfa7fd',
        location: '',
        comment: 'Goerli Initiative bootnode',
      },
    ],
    dnsNetworks: [
      'enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.goerli.ethdisco.net',
    ],
  },
  sepolia: {
    name: 'sepolia',
    chainId: 11155111,
    networkId: 11155111,
    defaultHardfork: 'shanghai',
    consensus: {
      type: 'pow',
      algorithm: 'ethash',
      ethash: {},
    },
    comment: 'PoW test network to replace Ropsten',
    url: 'https://github.com/ethereum/go-ethereum/pull/23730',
    genesis: {
      timestamp: '0x6159af19',
      gasLimit: 30000000,
      difficulty: 131072,
      nonce: '0x0000000000000000',
      extraData: '0x5365706f6c69612c20417468656e732c204174746963612c2047726565636521',
    },
    hardforks: [
      {
        name: 'chainstart',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'homestead',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'tangerineWhistle',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'spuriousDragon',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'byzantium',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'constantinople',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'petersburg',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'istanbul',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'muirGlacier',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'berlin',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'london',
        block: 0,
        forkHash: '0xfe3366e7',
      },
      {
        // The forkHash will remain same as mergeForkIdTransition is post merge,
        // terminal block: https://sepolia.etherscan.io/block/1450408
        name: 'paris',
        ttd: '17000000000000000',
        block: 1450409,
        forkHash: '0xfe3366e7',
      },
      {
        name: 'mergeForkIdTransition',
        block: 1735371,
        forkHash: '0xb96cbd13',
      },
      {
        name: 'shanghai',
        block: null,
        timestamp: '1677557088',
        forkHash: '0xf7f9bc08',
      },
      {
        name: 'cancun',
        block: null,
        timestamp: '1706655072',
        forkHash: '0x88cf81d9',
      },
    ],
    bootstrapNodes: [
      {
        ip: '18.168.182.86',
        port: 30303,
        id: '9246d00bc8fd1742e5ad2428b80fc4dc45d786283e05ef6edbd9002cbc335d40998444732fbe921cb88e1d2c73d1b1de53bae6a2237996e9bfe14f871baf7066',
        location: '',
        comment: 'geth',
      },
      {
        ip: '52.14.151.177',
        port: 30303,
        id: 'ec66ddcf1a974950bd4c782789a7e04f8aa7110a72569b6e65fcd51e937e74eed303b1ea734e4d19cfaec9fbff9b6ee65bf31dcb50ba79acce9dd63a6aca61c7',
        location: '',
        comment: 'besu',
      },
      {
        ip: '165.22.196.173',
        port: 30303,
        id: 'ce970ad2e9daa9e14593de84a8b49da3d54ccfdf83cbc4fe519cb8b36b5918ed4eab087dedd4a62479b8d50756b492d5f762367c8d20329a7854ec01547568a6',
        location: '',
        comment: 'EF',
      },
      {
        ip: '65.108.95.67',
        port: 30303,
        id: '075503b13ed736244896efcde2a992ec0b451357d46cb7a8132c0384721742597fc8f0d91bbb40bb52e7d6e66728d36a1fda09176294e4a30cfac55dcce26bc6',
        location: '',
        comment: 'lodestar',
      },
    ],
    dnsNetworks: [
      'enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.sepolia.ethdisco.net',
    ],
  },
  holesky: {
    name: 'holesky',
    chainId: 17000,
    networkId: 17000,
    defaultHardfork: 'paris',
    consensus: {
      type: 'pos',
      algorithm: 'casper',
    },
    comment: 'PoS test network to replace Goerli',
    url: 'https://github.com/eth-clients/holesky/',
    genesis: {
      baseFeePerGas: '0x3B9ACA00',
      difficulty: '0x01',
      extraData: '0x',
      gasLimit: '0x17D7840',
      nonce: '0x0000000000001234',
      timestamp: '0x65156994',
    },
    hardforks: [
      {
        name: 'chainstart',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'homestead',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'tangerineWhistle',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'spuriousDragon',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'byzantium',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'constantinople',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'petersburg',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'istanbul',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'muirGlacier',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'berlin',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'london',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'paris',
        ttd: '0',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'mergeForkIdTransition',
        block: 0,
        forkHash: '0xc61a6098',
      },
      {
        name: 'shanghai',
        block: null,
        timestamp: '1696000704',
        forkHash: '0xfd4f016b',
      },
      {
        name: 'cancun',
        block: null,
        timestamp: '1707305664',
        forkHash: '0x9b192ad0',
      },
    ],
    bootstrapNodes: [
      {
        ip: '146.190.13.128',
        port: 30303,
        id: 'ac906289e4b7f12df423d654c5a962b6ebe5b3a74cc9e06292a85221f9a64a6f1cfdd6b714ed6dacef51578f92b34c60ee91e9ede9c7f8fadc4d347326d95e2b',
        location: '',
        comment: 'bootnode 1',
      },
      {
        ip: '178.128.136.233',
        port: 30303,
        id: 'a3435a0155a3e837c02f5e7f5662a2f1fbc25b48e4dc232016e1c51b544cb5b4510ef633ea3278c0e970fa8ad8141e2d4d0f9f95456c537ff05fdf9b31c15072',
        location: '',
        comment: 'bootnode 2',
      },
    ],
    dnsNetworks: [
      'enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.holesky.ethdisco.net',
    ],
  },
  kaustinen: {
    name: 'kaustinen',
    chainId: 69420,
    networkId: 69420,
    defaultHardfork: 'prague',
    consensus: {
      type: 'pos',
      algorithm: 'casper',
    },
    comment: 'Verkle kaustinen testnet 2 (likely temporary, do not hard-wire into production code)',
    url: 'https://github.com/eth-clients/kaustinen/',
    genesis: {
      difficulty: '0x01',
      extraData: '0x',
      gasLimit: '0x17D7840',
      nonce: '0x0000000000001234',
      timestamp: '0x65608a64',
    },
    hardforks: [
      {
        name: 'chainstart',
        block: 0,
      },
      {
        name: 'homestead',
        block: 0,
      },
      {
        name: 'tangerineWhistle',
        block: 0,
      },
      {
        name: 'spuriousDragon',
        block: 0,
      },
      {
        name: 'byzantium',
        block: 0,
      },
      {
        name: 'constantinople',
        block: 0,
      },
      {
        name: 'petersburg',
        block: 0,
      },
      {
        name: 'istanbul',
        block: 0,
      },
      {
        name: 'berlin',
        block: 0,
      },
      {
        name: 'london',
        block: 0,
      },
      {
        name: 'paris',
        ttd: '0',
        block: 0,
      },
      {
        name: 'mergeForkIdTransition',
        block: 0,
      },
      {
        name: 'shanghai',
        block: null,
        timestamp: '0',
      },
      {
        name: 'prague',
        block: null,
        timestamp: '1700825700',
      },
    ],
    bootstrapNodes: [],
    dnsNetworks: [],
  },
}
