
# Config

Toml files format used in geth are being deprecated.

Bor uses uses JSON and [HCL](https://github.com/hashicorp/hcl) formats to create configuration files. This is the format in HCL alongside the default values:

```
chain = "mainnet"
log-level = "info"
data-dir = ""
sync-mode = "fast"
gc-mode = "full"
snapshot = true
ethstats = ""
whitelist = {}

p2p {
    max-peers = 30
    max-pend-peers = 50
    bind = "0.0.0.0"
    port = 30303
    no-discover = false
    nat = "any"
    discovery {
        v5-enabled = false
        bootnodes = []
        bootnodesv4 = []
        bootnodesv5 = []
        staticNodes = []
        trustedNodes = []
        dns = []
    }
}

heimdall {
    url = "http://localhost:1317"
    without = false
}

txpool {
    locals = []
    no-locals = false
    journal = ""
    rejournal = "1h"
    price-limit = 1
    price-bump = 10
    account-slots = 16
    global-slots = 4096
    account-queue = 64
    global-queue = 1024
    lifetime = "3h"
}

sealer {
    enabled = false
    etherbase = ""
    gas-ceil = 8000000
    extra-data = ""
}

gpo {
    blocks = 20
    percentile = 60
}

jsonrpc {
    ipc-disable = false
    ipc-path = ""
    modules = ["web3", "net"]
    cors = ["*"]
    vhost = ["*"]
    
    http {
        enabled = false
        port = 8545
        prefix = ""
        host = "localhost"
    }

    ws {
        enabled = false
        port = 8546
        prefix = ""
        host = "localhost"
    }

    graphqh {
        enabled = false
    }
}

telemetry {
    enabled = false
    expensive = false

    influxdb {
        v1-enabled = false
        endpoint = ""
        database = ""
        username = ""
        password = ""
        v2-enabled = false
        token = ""
        bucket = ""
        organization = ""
    }
}

cache {
    cache = 1024
    perc-database = 50
    perc-trie = 15
    perc-gc = 25
    perc-snapshot = 10
    journal = "triecache"
    rejournal = "60m"
    no-prefetch = false
    preimages = false
    tx-lookup-limit = 2350000
}

accounts {
    unlock = []
    password-file = ""
    allow-insecure-unlock = false
    use-lightweight-kdf = false
}

grpc {
    addr = ":3131"
}
```
