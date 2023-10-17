// NOTE: Every redirect here must also be included in netlify.toml while we are using Netlify

const redirects = [
  {
    source: '/getting-started/dev-mode',
    destination: '/docs/developers/dapp-developer/dev-mode',
    permanent: true
  },
  {
    source: '/docs/getting-started/dev-mode',
    destination: '/docs/developers/dapp-developer/dev-mode',
    permanent: true
  },
  {
    source: '/docs/install-and-build/installing-geth',
    destination: '/docs/getting-started/installing-geth',
    permanent: true
  },
  {
    source: '/docs/install-and-build/backup-restore',
    destination: '/docs/getting-started/backup-restore',
    permanent: true
  },
  {
    source: '/docs/interface/command-line-options',
    destination: '/docs/fundamentals/command-line-options',
    permanent: true
  },
  {
    source: '/docs/interface/pruning',
    destination: '/docs/fundamentals/pruning',
    permanent: true
  },
  {
    source: '/docs/interface/consensus-clients',
    destination: '/docs/getting-started/consensus-clients',
    permanent: true
  },
  {
    source: '/docs/interface/peer-to-peer',
    destination: '/docs/fundamentals/peer-to-peer',
    permanent: true
  },
  {
    source: '/docs/interface/les',
    destination: '/docs/fundamentals/les',
    permanent: true
  },
  {
    source: '/docs/interface/managing-your-accounts',
    destination: '/docs/fundamentals/account-management',
    permanent: true
  },
  {
    source: '/docs/interface/javascript-console',
    destination: '/docs/interacting-with-geth/javascript-console',
    permanent: true
  },
  {
    source: '/getting-started/private-network',
    destination: '/docs/fundamentals/private-network',
    permanent: true
  },
  {
    source: '/docs/interface/private-network',
    destination: '/docs/fundamentals/private-network',
    permanent: true
  },
  {
    source: '/docs/interface/mining',
    destination: '/docs/fundamentals/mining',
    permanent: true
  },
  {
    source: '/docs/interface/metrics',
    destination: '/docs/monitoring/metrics',
    permanent: true
  },
  {
    source: '/docs/dapp/native',
    destination: '/docs/developers/dapp-developer/native',
    permanent: true
  },
  {
    source: '/docs/dapp/tracing',
    destination: '/docs/developers/evm-tracing',
    permanent: true
  },
  {
    source: '/docs/dapp/custom-tracer',
    destination: '/docs/developers/evm-tracing/custom-tracer',
    permanent: true
  },
  {
    source: '/docs/dapp/builtin-tracers',
    destination: '/docs/developers/evm-tracing/built-in-tracers',
    permanent: true
  },
  {
    source: '/docs/dapp/native-accounts',
    destination: '/docs/developers/dapp-developer/native-accounts',
    permanent: true
  },
  {
    source: '/docs/dapp/native-bindings',
    destination: '/docs/developers/dapp-developer/native-bindings',
    permanent: true
  },
  {
    source: '/docs/dapp/mobile',
    destination: '/docs/developers/dapp-developer/mobile',
    permanent: true
  },
  {
    source: '/docs/dapp/mobile-accounts',
    destination: '/docs/developers/dapp-developer/mobile',
    permanent: true
  },
  {
    source: '/docs/rpc/server',
    destination: '/docs/interacting-with-geth/rpc',
    permanent: true
  },
  {
    source: '/docs/rpc/pubsub',
    destination: '/docs/interacting-with-geth/rpc/pubsub',
    permanent: true
  },
  {
    source: '/docs/rpc/batch',
    destination: '/docs/interacting-with-geth/rpc/batch',
    permanent: true
  },
  {
    source: '/docs/rpc/graphql',
    destination: '/docs/interacting-with-geth/rpc/graphql',
    permanent: true
  },
  {
    source: '/docs/rpc/ns-admin',
    destination: '/docs/interacting-with-geth/rpc/ns-admin',
    permanent: true
  },
  {
    source: '/docs/rpc/ns-clique',
    destination: '/docs/interacting-with-geth/rpc/ns-clique',
    permanent: true
  },
  {
    source: '/docs/rpc/ns-debug',
    destination: '/docs/interacting-with-geth/rpc/ns-debug',
    permanent: true
  },
  {
    source: '/docs/rpc/ns-eth',
    destination: '/docs/interacting-with-geth/rpc/ns-eth',
    permanent: true
  },
  {
    source: '/docs/rpc/ns-les',
    destination: '/docs/interacting-with-geth/rpc/ns-les',
    permanent: true
  },
  {
    source: '/docs/rpc/ns-miner',
    destination: '/docs/interacting-with-geth/rpc/ns-miner',
    permanent: true
  },
  {
    source: '/docs/rpc/ns-net',
    destination: '/docs/interacting-with-geth/rpc/ns-net',
    permanent: true
  },
  {
    source: '/docs/rpc/ns-personal',
    destination: '/docs/interacting-with-geth/rpc/ns-personal',
    permanent: true
  },
  {
    source: '/docs/rpc/ns-txpool',
    destination: '/docs/interacting-with-geth/rpc/ns-txpool',
    permanent: true
  },
  {
    source: '/docs/rpc/objects',
    destination: '/docs',
    permanent: true
  },
  {
    source: '/docs/interacting-with-geth/rpc/objects',
    destination: '/docs',
    permanent: true
  },
  {
    source: '/docs/developers/dev-guide',
    destination: '/docs/developers/geth-developer/dev-guide',
    permanent: true
  },
  {
    source: '/docs/developers/code-review-guidelines',
    destination: '/docs/developers/geth-developer/code-review-guidelines',
    permanent: true
  },
  {
    source: '/docs/developers/issue-handling-workflow',
    destination: '/docs/developers/geth-developer/issue-handling-workflow',
    permanent: true
  },
  {
    source: '/docs/developers/dns-discovery-setup',
    destination: '/docs/developers/geth-developer/dns-discovery-setup',
    permanent: true
  },
  {
    source: '/docs/clef/introduction',
    destination: '/docs/tools/clef/introduction',
    permanent: true
  },
  {
    source: '/docs/clef/tutorial',
    destination: '/docs/tools/clef/tutorial',
    permanent: true
  },
  {
    source: '/docs/clef/cliquesigning',
    destination: '/docs/tools/clef/clique-signing',
    permanent: true
  },
  {
    source: '/docs/clef/rules',
    destination: '/docs/tools/clef/rules',
    permanent: true
  },
  {
    source: '/docs/clef/setup',
    destination: '/docs/tools/clef/setup',
    permanent: true
  },
  {
    source: '/docs/clef/apis',
    destination: '/docs/tools/clef/apis',
    permanent: true
  },
  {
    source: '/docs/clef/datatypes',
    destination: '/docs/tools/clef/datatypes',
    permanent: true
  },
  {
    source: '/docs/interface/sync-mode',
    destination: '/docs/fundamentals/sync-modes',
    permanent: true
  },
  {
    source: '/docs/interface/hardware',
    destination: '/docs/getting-started/hardware-requirements',
    permanent: true
  },
  {
    source: '/docs/interacting-with-geth',
    destination: '/docs/interacting-with-geth/rpc',
    permanent: true
  },
  {
    source: '/docs/developers/dapp-developer',
    destination: '/docs/developers/dapp-developer/dev-mode',
    permanent: true
  },
  {
    source: '/docs/developers/geth-developer',
    destination: '/docs/developers/geth-developer/dev-guide',
    permanent: true
  },
  {
    source: '/docs/monitoring',
    destination: '/docs/monitoring/dashboards',
    permanent: true
  },
  {
    source: '/docs/tools',
    destination: '/docs/tools/clef/introduction',
    permanent: true
  },
  {
    source: '/docs/tools/clef',
    destination: '/docs/tools/clef/introduction',
    permanent: true
  },
  {
    source: '/docs/developers/contributing',
    destination: '/docs/developers/geth-developer/contributing',
    permanent: true
  },
  {
    source: '/docs/interacting-with-geth/rpc/ns-personal-deprecation',
    destination: '/docs/interacting-with-geth/rpc/ns-personal',
    permanent: true
  }
];

module.exports = {
  redirects
};
