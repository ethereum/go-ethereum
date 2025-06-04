import { createContract, tokenFromSymbol } from '../abi/index.ts';
import { type IWeb3Provider, createDecimal } from '../utils.ts';

const ABI = [
  {
    type: 'function',
    name: 'latestRoundData',
    outputs: [
      { name: 'roundId', type: 'uint80' },
      { name: 'answer', type: 'int256' },
      { name: 'startedAt', type: 'uint256' },
      { name: 'updatedAt', type: 'uint256' },
      { name: 'answeredInRound', type: 'uint80' },
    ],
  },
] as const;

export const TOKENS: Record<string, { decimals: number; contract: string; tokenContract: string }> =
  {
    '1INCH': {
      decimals: 8,
      contract: '0xc929ad75b72593967de83e7f7cda0493458261d9',
      tokenContract: '0x111111111117dc0aa78b770fa6a738034120c302',
    },
    AAPL: {
      decimals: 8,
      contract: '0x139c8512cde1778e9b9a8e721ce1aebd4dd43587',
      tokenContract: '0x7edc9e8a1196259b7c6aba632037a9443d4e14f7',
    },
    AAVE: {
      decimals: 8,
      contract: '0x547a514d5e3769680ce22b2361c10ea13619e8a9',
      tokenContract: '0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9',
    },
    ADX: {
      decimals: 8,
      contract: '0x231e764b44b2c1b7ca171fa8021a24ed520cde10',
      tokenContract: '0x4470bb87d77b963a013db939be332f927f2b992e',
    },
    AKRO: {
      decimals: 8,
      contract: '0xb23d105df4958b4b81757e12f2151b5b5183520b',
      tokenContract: '0x8ab7404063ec4dbcfd4598215992dc3f8ec853d7',
    },
    AMP: {
      decimals: 8,
      contract: '0x8797abc4641de76342b8ace9c63e3301dc35e3d8',
      tokenContract: '0xff20817765cb7f73d4bde2e66e067e58d11095c2',
    },
    AMPL: {
      decimals: 18,
      contract: '0xe20ca8d7546932360e37e9d72c1a47334af57706',
      tokenContract: '0xd46ba6d942050d489dbd938a2c909a5d5039a161',
    },
    AMZN: {
      decimals: 8,
      contract: '0x8994115d287207144236c13be5e2bdbf6357d9fd',
      tokenContract: '0xd6a073d973f95b7ce2ecf2b19224fa12103cf460',
    },
    ANKR: {
      decimals: 8,
      contract: '0x7eed379bf00005cfed29fed4009669de9bcc21ce',
      tokenContract: '0x8290333cef9e6d528dd5618fb97a76f268f3edd4',
    },
    BADGER: {
      decimals: 8,
      contract: '0x66a47b7206130e6ff64854ef0e1edfa237e65339',
      tokenContract: '0x3472a5a71965499acd81997a54bba8d852c6e53d',
    },
    BAND: {
      decimals: 8,
      contract: '0x919c77acc7373d000b329c1276c76586ed2dd19f',
      tokenContract: '0xba11d00c5f74255f56a5e366f4f77f5a186d7f55',
    },
    BAT: {
      decimals: 8,
      contract: '0x9441d7556e7820b5ca42082cfa99487d56aca958',
      tokenContract: '0x0d8775f648430679a709e98d2b0cb6250d2887ef',
    },
    BNB: {
      decimals: 8,
      contract: '0x14e613ac84a31f709eadbdf89c6cc390fdc9540a',
      tokenContract: '0xb8c77482e45f1f44de1745f52c74426c631bdd52',
    },
    BNT: {
      decimals: 8,
      contract: '0x1e6cf0d433de4fe882a437abc654f58e1e78548c',
      tokenContract: '0x1f573d6fb3f13d689ff844b4ce37794d79a7ff1c',
    },
    BTM: {
      decimals: 8,
      contract: '0x9fccf42d21ab278e205e7bb310d8979f8f4b5751',
      tokenContract: '0xcb97e65f07da24d46bcdd078ebebd7c6e6e3d750',
    },
    BUSD: {
      decimals: 8,
      contract: '0x833d8eb16d306ed1fbb5d7a2e019e106b960965a',
      tokenContract: '0x4fabb145d64652a948d72533023f6e7a623c7c53',
    },
    COMP: {
      decimals: 8,
      contract: '0xdbd020caef83efd542f4de03e3cf0c28a4428bd5',
      tokenContract: '0xc00e94cb662c3520282e6f5717214004a7f26888',
    },
    COVER: {
      decimals: 8,
      contract: '0x0ad50393f11ffac4dd0fe5f1056448ecb75226cf',
      tokenContract: '0x4688a8b1f292fdab17e9a90c8bc379dc1dbd8713',
    },
    CRO: {
      decimals: 8,
      contract: '0x00cb80cf097d9aa9a3779ad8ee7cf98437eae050',
      tokenContract: '0xa0b73e1ff0b80914ab6fe0444e65848c4c34450b',
    },
    CRV: {
      decimals: 8,
      contract: '0xcd627aa160a6fa45eb793d19ef54f5062f20f33f',
      tokenContract: '0xd533a949740bb3306d119cc777fa900ba034cd52',
    },
    DAI: {
      decimals: 8,
      contract: '0xaed0c38402a5d19df6e4c03f4e2dced6e29c1ee9',
      tokenContract: '0x60d9564303c70d3f040ea9393d98d94f767d020c',
    },
    DPI: {
      decimals: 8,
      contract: '0xd2a593bf7594ace1fad597adb697b5645d5eddb2',
      tokenContract: '0x1494ca1f11d487c2bbe4543e90080aeba4ba3c2b',
    },
    EOS: {
      decimals: 8,
      contract: '0x10a43289895eaff840e8d45995bba89f9115ecee',
      tokenContract: '0x86fa049857e0209aa7d9e616f7eb3b3b78ecfdb0',
    },
    FXS: {
      decimals: 8,
      contract: '0x6ebc52c8c1089be9eb3945c4350b68b8e4c2233f',
      tokenContract: '0x3432b6a60d23ca0dfca7761b7ab56459d9c964d0',
    },
    HT: {
      decimals: 8,
      contract: '0xe1329b3f6513912caf589659777b66011aee5880',
      tokenContract: '0x6f259637dcd74c767781e37bc6133cd6a68aa161',
    },
    IOST: {
      decimals: 8,
      contract: '0xd0935838935349401c73a06fcde9d63f719e84e5',
      tokenContract: '0xfa1a856cfa3409cfa145fa4e20eb270df3eb21ab',
    },
    KNC: {
      decimals: 8,
      contract: '0xf8ff43e991a81e6ec886a3d281a2c6cc19ae70fc',
      tokenContract: '0xdd974d5c2e2928dea5f71b9825b8b646686bd200',
    },
    LINK: {
      decimals: 8,
      contract: '0x2c1d072e956affc0d435cb7ac38ef18d24d9127c',
      tokenContract: '0x514910771af9ca656af840dff83e8264ecf986ca',
    },
    LRC: {
      decimals: 8,
      contract: '0xfd33ec6abaa1bdc3d9c6c85f1d6299e5a1a5511f',
      tokenContract: '0xef68e7c694f40c8202821edf525de3782458639f',
    },
    MATIC: {
      decimals: 8,
      contract: '0x7bac85a8a13a4bcd8abb3eb7d6b4d632c5a57676',
      tokenContract: '0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0',
    },
    MKR: {
      decimals: 8,
      contract: '0xec1d1b3b0443256cc3860e24a46f108e699484aa',
      tokenContract: '0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2',
    },
    MTA: {
      decimals: 8,
      contract: '0xc751e86208f0f8af2d5cd0e29716ca7ad98b5ef5',
      tokenContract: '0xa3bed4e1c75d00fa6f4e5e6922db7261b5e9acd2',
    },
    NFLX: {
      decimals: 8,
      contract: '0x67c2e69c5272b94af3c90683a9947c39dc605dde',
      tokenContract: '0x0a3dc37762f0102175fd43d3871d7fa855626146',
    },
    NMR: {
      decimals: 8,
      contract: '0xcc445b35b3636bc7cc7051f4769d8982ed0d449a',
      tokenContract: '0x1776e1f26f98b1a5df9cd347953a26dd3cb46671',
    },
    OCEAN: {
      decimals: 8,
      contract: '0x7ece4e4e206ed913d991a074a19c192142726797',
      tokenContract: '0x967da4048cd07ab37855c090aaf366e4ce1b9f48',
    },
    OKB: {
      decimals: 8,
      contract: '0x22134617ae0f6ca8d89451e5ae091c94f7d743dc',
      tokenContract: '0x75231f58b43240c9718dd58b4967c5114342a86c',
    },
    OMG: {
      decimals: 8,
      contract: '0x7d476f061f8212a8c9317d5784e72b4212436e93',
      tokenContract: '0xd26114cd6ee289accf82350c8d8487fedb8a0c07',
    },
    OXT: {
      decimals: 8,
      contract: '0xd75aaae4af0c398ca13e2667be57af2cca8b5de6',
      tokenContract: '0x4575f41308ec1483f3d399aa9a2826d74da13deb',
    },
    REN: {
      decimals: 8,
      contract: '0x0f59666ede214281e956cb3b2d0d69415aff4a01',
      tokenContract: '0x408e41876cccdc0f92210600ef50372656052a38',
    },
    SAND: {
      decimals: 8,
      contract: '0x35e3f7e558c04ce7eee1629258ecbba03b36ec56',
      tokenContract: '0x3845badade8e6dff049820680d1f14bd3903a5d0',
    },
    SNX: {
      decimals: 8,
      contract: '0xdc3ea94cd0ac27d9a86c180091e7f78c683d3699',
      tokenContract: '0xc011a73ee8576fb46f5e1c5751ca3b9fe0af2a6f',
    },
    SUSHI: {
      decimals: 8,
      contract: '0xcc70f09a6cc17553b2e31954cd36e4a2d89501f7',
      tokenContract: '0x6b3595068778dd592e39a122f4f5a5cf09c90fe2',
    },
    SXP: {
      decimals: 8,
      contract: '0xfb0cfd6c19e25db4a08d8a204a387cea48cc138f',
      tokenContract: '0x8ce9137d39326ad0cd6491fb5cc0cba0e089b6a9',
    },
    UNI: {
      decimals: 8,
      contract: '0x553303d460ee0afb37edff9be42922d8ff63220e',
      tokenContract: '0x1f9840a85d5af5bf1d1762f925bdaddc4201f984',
    },
    USDC: {
      decimals: 8,
      contract: '0x8fffffd4afb6115b954bd326cbe7b4ba576818f6',
      tokenContract: '0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48',
    },
    USDK: {
      decimals: 8,
      contract: '0xfac81ea9dd29d8e9b212acd6edbeb6de38cb43af',
      tokenContract: '0x1c48f86ae57291f7686349f12601910bd8d470bb',
    },
    USDT: {
      decimals: 8,
      contract: '0x3e7d1eab13ad0104d2750b8863b489d65364e32d',
      tokenContract: '0xdac17f958d2ee523a2206206994597c13d831ec7',
    },
    YFI: {
      decimals: 8,
      contract: '0xa027702dbb89fbd58938e4324ac03b58d812b0e1',
      tokenContract: '0x0bc529c00c6401aef6d220be8c6ea1667f6ad93e',
    },
    ZRX: {
      decimals: 8,
      contract: '0x2885d15b8af22648b98b122b22fdf4d2a56c6023',
      tokenContract: '0xe41d2489571d322189246dafa5ebde1f4699f498',
    },
    SUSD: {
      decimals: 8,
      contract: '0xad35bd71b9afe6e4bdc266b345c198eadef9ad94',
      tokenContract: '0x57ab1e02fee23774580c119740129eac7081e9d3',
    },
    // Wrapped tokens uses price of original coin
    WBTC: {
      decimals: 8,
      contract: '0xf4030086522a5beea4988f8ca5b36dbc97bee88c',
      tokenContract: tokenFromSymbol('WBTC')!.contract,
    },
    WETH: {
      decimals: 8,
      contract: '0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419',
      tokenContract: tokenFromSymbol('WETH')!.contract,
    },
  };

export default class Chainlink {
  readonly net: IWeb3Provider;
  constructor(net: IWeb3Provider) {
    this.net = net;
  }
  async price(contract: string, decimals: number): Promise<number> {
    const prices = createContract(ABI, this.net, contract);
    let res = await prices.latestRoundData.call();
    const num = Number.parseFloat(createDecimal(decimals).encode(res.answer));
    if (Number.isNaN(num)) throw new Error('invalid data received');
    return num;
  }

  async coinPrice(symbol: string): Promise<number> {
    // Only common coins
    const COINS: Record<string, { decimals: number; contract: string }> = {
      BCH: { decimals: 8, contract: '0x9f0f69428f923d6c95b781f89e165c9b2df9789d' },
      BTC: { decimals: 8, contract: '0xf4030086522a5beea4988f8ca5b36dbc97bee88c' },
      DOGE: { decimals: 8, contract: '0x2465cefd3b488be410b941b1d4b2767088e2a028' },
      ETH: { decimals: 8, contract: '0x5f4ec3df9cbd43714fe2740f5e3616155c5b8419' },
      XMR: { decimals: 8, contract: '0xfa66458cce7dd15d8650015c4fce4d278271618f' },
      ZEC: { decimals: 8, contract: '0xd54b033d48d0475f19c5fccf7484e8a981848501' },
    };
    const coin = COINS[symbol.toUpperCase()];
    if (!coin) throw new Error(`micro-web3/chainlink: unknown coin: ${symbol}`);
    return await this.price(coin.contract, coin.decimals);
  }

  async tokenPrice(symbol: string): Promise<number> {
    const token = TOKENS[symbol.toUpperCase()];
    if (!token) throw new Error(`micro-web3/chainlink: unknown token: ${symbol}`);
    return await this.price(token.contract, token.decimals);
  }
}
