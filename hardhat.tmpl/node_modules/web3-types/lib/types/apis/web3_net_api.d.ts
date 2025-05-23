import { HexString } from '../primitives_types.js';
export type Web3NetAPI = {
    net_version: () => string;
    net_peerCount: () => HexString;
    net_listening: () => boolean;
};
//# sourceMappingURL=web3_net_api.d.ts.map