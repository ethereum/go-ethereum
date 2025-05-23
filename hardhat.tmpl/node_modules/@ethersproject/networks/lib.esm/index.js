"use strict";
import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);
;
function isRenetworkable(value) {
    return (value && typeof (value.renetwork) === "function");
}
function ethDefaultProvider(network) {
    const func = function (providers, options) {
        if (options == null) {
            options = {};
        }
        const providerList = [];
        if (providers.InfuraProvider && options.infura !== "-") {
            try {
                providerList.push(new providers.InfuraProvider(network, options.infura));
            }
            catch (error) { }
        }
        if (providers.EtherscanProvider && options.etherscan !== "-") {
            try {
                providerList.push(new providers.EtherscanProvider(network, options.etherscan));
            }
            catch (error) { }
        }
        if (providers.AlchemyProvider && options.alchemy !== "-") {
            try {
                providerList.push(new providers.AlchemyProvider(network, options.alchemy));
            }
            catch (error) { }
        }
        if (providers.PocketProvider && options.pocket !== "-") {
            // These networks are currently faulty on Pocket as their
            // network does not handle the Berlin hardfork, which is
            // live on these ones.
            // @TODO: This goes away once Pocket has upgraded their nodes
            const skip = ["goerli", "ropsten", "rinkeby", "sepolia"];
            try {
                const provider = new providers.PocketProvider(network, options.pocket);
                if (provider.network && skip.indexOf(provider.network.name) === -1) {
                    providerList.push(provider);
                }
            }
            catch (error) { }
        }
        if (providers.CloudflareProvider && options.cloudflare !== "-") {
            try {
                providerList.push(new providers.CloudflareProvider(network));
            }
            catch (error) { }
        }
        if (providers.AnkrProvider && options.ankr !== "-") {
            try {
                const skip = ["ropsten"];
                const provider = new providers.AnkrProvider(network, options.ankr);
                if (provider.network && skip.indexOf(provider.network.name) === -1) {
                    providerList.push(provider);
                }
            }
            catch (error) { }
        }
        if (providers.QuickNodeProvider && options.quicknode !== "-") {
            try {
                providerList.push(new providers.QuickNodeProvider(network, options.quicknode));
            }
            catch (error) { }
        }
        if (providerList.length === 0) {
            return null;
        }
        if (providers.FallbackProvider) {
            let quorum = 1;
            if (options.quorum != null) {
                quorum = options.quorum;
            }
            else if (network === "homestead") {
                quorum = 2;
            }
            return new providers.FallbackProvider(providerList, quorum);
        }
        return providerList[0];
    };
    func.renetwork = function (network) {
        return ethDefaultProvider(network);
    };
    return func;
}
function etcDefaultProvider(url, network) {
    const func = function (providers, options) {
        if (providers.JsonRpcProvider) {
            return new providers.JsonRpcProvider(url, network);
        }
        return null;
    };
    func.renetwork = function (network) {
        return etcDefaultProvider(url, network);
    };
    return func;
}
const homestead = {
    chainId: 1,
    ensAddress: "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e",
    name: "homestead",
    _defaultProvider: ethDefaultProvider("homestead")
};
const ropsten = {
    chainId: 3,
    ensAddress: "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e",
    name: "ropsten",
    _defaultProvider: ethDefaultProvider("ropsten")
};
const classicMordor = {
    chainId: 63,
    name: "classicMordor",
    _defaultProvider: etcDefaultProvider("https://www.ethercluster.com/mordor", "classicMordor")
};
// See: https://chainlist.org
const networks = {
    unspecified: { chainId: 0, name: "unspecified" },
    homestead: homestead,
    mainnet: homestead,
    morden: { chainId: 2, name: "morden" },
    ropsten: ropsten,
    testnet: ropsten,
    rinkeby: {
        chainId: 4,
        ensAddress: "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e",
        name: "rinkeby",
        _defaultProvider: ethDefaultProvider("rinkeby")
    },
    kovan: {
        chainId: 42,
        name: "kovan",
        _defaultProvider: ethDefaultProvider("kovan")
    },
    goerli: {
        chainId: 5,
        ensAddress: "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e",
        name: "goerli",
        _defaultProvider: ethDefaultProvider("goerli")
    },
    kintsugi: { chainId: 1337702, name: "kintsugi" },
    sepolia: {
        chainId: 11155111,
        ensAddress: "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e",
        name: "sepolia",
        _defaultProvider: ethDefaultProvider("sepolia")
    },
    holesky: {
        chainId: 17000,
        name: "holesky",
        _defaultProvider: ethDefaultProvider("holesky")
    },
    // ETC (See: #351)
    classic: {
        chainId: 61,
        name: "classic",
        _defaultProvider: etcDefaultProvider("https:/\/www.ethercluster.com/etc", "classic")
    },
    classicMorden: { chainId: 62, name: "classicMorden" },
    classicMordor: classicMordor,
    classicTestnet: classicMordor,
    classicKotti: {
        chainId: 6,
        name: "classicKotti",
        _defaultProvider: etcDefaultProvider("https:/\/www.ethercluster.com/kotti", "classicKotti")
    },
    xdai: { chainId: 100, name: "xdai" },
    matic: {
        chainId: 137,
        name: "matic",
        _defaultProvider: ethDefaultProvider("matic")
    },
    maticmum: {
        chainId: 80001,
        name: "maticmum",
        _defaultProvider: ethDefaultProvider("maticmum")
    },
    optimism: {
        chainId: 10,
        name: "optimism",
        _defaultProvider: ethDefaultProvider("optimism")
    },
    "optimism-kovan": { chainId: 69, name: "optimism-kovan" },
    "optimism-goerli": { chainId: 420, name: "optimism-goerli" },
    "optimism-sepolia": { chainId: 11155420, name: "optimism-sepolia" },
    arbitrum: { chainId: 42161, name: "arbitrum" },
    "arbitrum-rinkeby": { chainId: 421611, name: "arbitrum-rinkeby" },
    "arbitrum-goerli": { chainId: 421613, name: "arbitrum-goerli" },
    "arbitrum-sepolia": { chainId: 421614, name: "arbitrum-sepolia" },
    bnb: { chainId: 56, name: "bnb" },
    bnbt: { chainId: 97, name: "bnbt" },
};
/**
 *  getNetwork
 *
 *  Converts a named common networks or chain ID (network ID) to a Network
 *  and verifies a network is a valid Network..
 */
export function getNetwork(network) {
    // No network (null)
    if (network == null) {
        return null;
    }
    if (typeof (network) === "number") {
        for (const name in networks) {
            const standard = networks[name];
            if (standard.chainId === network) {
                return {
                    name: standard.name,
                    chainId: standard.chainId,
                    ensAddress: (standard.ensAddress || null),
                    _defaultProvider: (standard._defaultProvider || null)
                };
            }
        }
        return {
            chainId: network,
            name: "unknown"
        };
    }
    if (typeof (network) === "string") {
        const standard = networks[network];
        if (standard == null) {
            return null;
        }
        return {
            name: standard.name,
            chainId: standard.chainId,
            ensAddress: standard.ensAddress,
            _defaultProvider: (standard._defaultProvider || null)
        };
    }
    const standard = networks[network.name];
    // Not a standard network; check that it is a valid network in general
    if (!standard) {
        if (typeof (network.chainId) !== "number") {
            logger.throwArgumentError("invalid network chainId", "network", network);
        }
        return network;
    }
    // Make sure the chainId matches the expected network chainId (or is 0; disable EIP-155)
    if (network.chainId !== 0 && network.chainId !== standard.chainId) {
        logger.throwArgumentError("network chainId mismatch", "network", network);
    }
    // @TODO: In the next major version add an attach function to a defaultProvider
    // class and move the _defaultProvider internal to this file (extend Network)
    let defaultProvider = network._defaultProvider || null;
    if (defaultProvider == null && standard._defaultProvider) {
        if (isRenetworkable(standard._defaultProvider)) {
            defaultProvider = standard._defaultProvider.renetwork(network);
        }
        else {
            defaultProvider = standard._defaultProvider;
        }
    }
    // Standard Network (allow overriding the ENS address)
    return {
        name: network.name,
        chainId: standard.chainId,
        ensAddress: (network.ensAddress || standard.ensAddress || null),
        _defaultProvider: defaultProvider
    };
}
//# sourceMappingURL=index.js.map