import { HexString } from '../primitives_types.js';
export type Web3NetAPI = {
    net_version: () => string;
    net_peerCount: () => HexString;
    net_listening: () => boolean;
};
