
# Config

- The `bor dumpconfig` command prints the default configurations, in the TOML format, on the terminal.
    - One can `pipe (>)` this to a file (say `config.toml`) and use it to start bor.
        - Command to provide a config file: `bor server -config config.toml`
- Bor uses TOML, HCL, and JSON format config files.
- This is the format of the config file in TOML:
    - **NOTE: The values of these following flags are just for reference**
    - `config.toml` file:
```
chain = "mainnet"
identity = "myIdentity"
log-level = "INFO"
datadir = "/var/lib/bor/data"
keystore = "path/to/keystore"
syncmode = "full"
gcmode = "full"
snapshot = true
ethstats = ""

[p2p]
maxpeers = 30
maxpendpeers = 50
bind = "0.0.0.0"
port = 30303
nodiscover = false
nat = "any"

[p2p.discovery]
v5disc = false
bootnodes = ["enode://d860a01f9722d78051619d1e2351aba3f43f943f6f00718d1b9baa4101932a1f5011f16bb2b1bb35db20d6fe28fa0bf09636d26a87d31de9ec6203eeedb1f666@18.138.108.67:30303", "enode://22a8232c3abc76a16ae9d6c3b164f98775fe226f0917b0ca871128a74a8e9630b458460865bab457221f1d448dd9791d24c4e5d88786180ac185df813a68d4de@3.209.45.79:30303"]
bootnodesv4 = []
bootnodesv5 = ["enr:-KG4QOtcP9X1FbIMOe17QNMKqDxCpm14jcX5tiOE4_TyMrFqbmhPZHK_ZPG2Gxb1GE2xdtodOfx9-cgvNtxnRyHEmC0ghGV0aDKQ9aX9QgAAAAD__________4JpZIJ2NIJpcIQDE8KdiXNlY3AyNTZrMaEDhpehBDbZjM_L9ek699Y7vhUJ-eAdMyQW_Fil522Y0fODdGNwgiMog3VkcIIjKA", "enr:-KG4QDyytgmE4f7AnvW-ZaUOIi9i79qX4JwjRAiXBZCU65wOfBu-3Nb5I7b_Rmg3KCOcZM_C3y5pg7EBU5XGrcLTduQEhGV0aDKQ9aX9QgAAAAD__________4JpZIJ2NIJpcIQ2_DUbiXNlY3AyNTZrMaEDKnz_-ps3UUOfHWVYaskI5kWYO_vtYMGYCQRAR3gHDouDdGNwgiMog3VkcIIjKA"]
static-nodes = ["enode://8499da03c47d637b20eee24eec3c356c9a2e6148d6fe25ca195c7949ab8ec2c03e3556126b0d7ed644675e78c4318b08691b7b57de10e5f0d40d05b09238fa0a@52.187.207.27:30303"]
trusted-nodes = ["enode://2b252ab6a1d0f971d9722cb839a42cb81db019ba44c08754628ab4a823487071b5695317c8ccd085219c3a03af063495b2f1da8d18218da2d6a82981b45e6ffc@65.108.70.101:30303"]
dns = []

[heimdall]
url = "http://localhost:1317"
"bor.without" = false

[txpool]
locals = ["$ADDRESS1", "$ADDRESS2"]
nolocals = false
journal = ""
rejournal = "1h0m0s"
pricelimit = 30000000000
pricebump = 10
accountslots = 16
globalslots = 32768
accountqueue = 16
globalqueue = 32768
lifetime = "3h0m0s"

[miner]
mine = false
etherbase = ""
extradata = ""
gaslimit = 20000000
gasprice = "30000000000"

[jsonrpc]
ipcdisable = false
ipcpath = "/var/lib/bor/bor.ipc"
gascap = 50000000
txfeecap = 5e+00

[jsonrpc.http]
enabled = false
port = 8545
prefix = ""
host = "localhost"
api = ["eth", "net", "web3", "txpool", "bor"]
vhosts = ["*"]
corsdomain = ["*"]

[jsonrpc.ws]
enabled = false
port = 8546
prefix = ""
host = "localhost"
api = ["web3", "net"]
vhosts = ["*"]
corsdomain = ["*"]

[jsonrpc.graphql]
enabled = false
port = 0
prefix = ""
host = ""
api = []
vhosts = ["*"]
corsdomain = ["*"]

[gpo]
blocks = 20
percentile = 60
maxprice = "5000000000000"
ignoreprice = "2"

[telemetry]
metrics = false
expensive = false
prometheus-addr = ""
opencollector-endpoint = ""

[telemetry.influx]
influxdb = false
endpoint = ""
database = ""
username = ""
password = ""
influxdbv2 = false
token = ""
bucket = ""
organization = ""

[cache]
cache = 1024
gc = 25
snapshot = 10
database = 50
trie = 15
journal = "triecache"
rejournal = "1h0m0s"
noprefetch = false
preimages = false
txlookuplimit = 2350000

[accounts]
unlock = ["$ADDRESS1", "$ADDRESS2"]
password = "path/to/password.txt"
allow-insecure-unlock = false
lightkdf = false
disable-bor-wallet = false

[grpc]
addr = ":3131"

[developer]
dev = false
period = 0
```
