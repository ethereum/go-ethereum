import { type ContractABI, type ContractInfo, Decoder, createContract, deployContract, events } from './decoder.ts';
import { default as ERC1155 } from './erc1155.ts';
import { default as ERC20 } from './erc20.ts';
import { default as ERC721 } from './erc721.ts';
import { KYBER_NETWORK_PROXY_CONTRACT } from './kyber.ts';
import { UNISWAP_V2_ROUTER_CONTRACT } from './uniswap-v2.ts';
import { UNISWAP_V3_ROUTER_CONTRACT } from './uniswap-v3.ts';
import { default as WETH } from './weth.ts';
export { ERC1155, ERC20, ERC721, KYBER_NETWORK_PROXY_CONTRACT, UNISWAP_V2_ROUTER_CONTRACT, UNISWAP_V3_ROUTER_CONTRACT, WETH, };
export { Decoder, createContract, deployContract, events };
export type { ContractABI, ContractInfo };
export declare const TOKENS: Record<string, ContractInfo>;
export declare const CONTRACTS: Record<string, ContractInfo>;
export declare const tokenFromSymbol: (symbol: string) => {
    contract: string;
} & ContractInfo;
export type DecoderOpt = {
    customContracts?: Record<string, ContractInfo>;
    noDefault?: boolean;
};
export declare const decodeData: (to: string, data: string, amount?: bigint, opt?: DecoderOpt) => import("./decoder.ts").SignatureInfo | import("./decoder.ts").SignatureInfo[] | undefined;
export declare const decodeTx: (transaction: string, opt?: DecoderOpt) => import("./decoder.ts").SignatureInfo | import("./decoder.ts").SignatureInfo[] | undefined;
export declare const decodeEvent: (to: string, topics: string[], data: string, opt?: DecoderOpt) => import("./decoder.ts").SignatureInfo | import("./decoder.ts").SignatureInfo[] | undefined;
//# sourceMappingURL=index.d.ts.map