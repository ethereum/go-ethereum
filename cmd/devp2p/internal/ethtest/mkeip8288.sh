#!/bin/sh
# Regenerates testdata/eip8288: the chain and proof wrapper fixtures of the eth/75 and
# eth/76 tests. go-ethereum implements neither leanSPHINCS signing, recursive proving nor
# EIP-8288 blocks, so the fixtures come from Nethermind's Eip8288Bench tool, and block 1
# is built by a Nethermind node from the same checkout through the Engine API.
#
# Usage: mkeip8288.sh <nethermind checkout>
# Needs docker and python3. The checkout must provide the tool's wire-fixtures command.
set -eu

nethermind=$(cd "$1" && pwd)
here=$(cd "$(dirname "$0")" && pwd)
out=$here/testdata/eip8288
work=$(mktemp -d "$nethermind/.eip8288-fixtures.XXXXXX")
node=eip8288-fixtures-$$
jwt=0x7365637265747365637265747365637265747365637265747365637265747365
trap 'docker rm -f $node >/dev/null 2>&1 || true; rm -rf "$work"' EXIT

arch=$(docker info --format '{{.Architecture}}')
case $arch in x86_64 | amd64) rid=linux-x64 ;; *) rid=linux-arm64 ;; esac
w=/src/$(basename "$work")
cp "$here/testdata/genesis.json" "$work/system-contracts.json"
docker run --rm -e HOME=/tmp -e DOTNET_CLI_HOME=/tmp -v "$nethermind:/src" -w /src mcr.microsoft.com/dotnet/sdk:10.0 sh -ec "
  dotnet publish tools/Eip8288Bench/Eip8288Bench.csproj -c Release -r $rid --self-contained false -o $w/tool
  dotnet publish src/Nethermind/Nethermind.Runner/Nethermind.Runner.csproj -c Release -r $rid --self-contained false -o $w/node
  cd $w && dotnet tool/Eip8288Bench.dll wire-fixtures --out=$w/out --system-contracts=$w/system-contracts.json"

printf %s "$jwt" > "$work/jwt"
docker run -d --name $node -p 127.0.0.1:18545:8545 -p 127.0.0.1:18551:8551 -v "$work:/w" mcr.microsoft.com/dotnet/aspnet:10.0 \
  /w/node/nethermind --config=none --datadir=/data --Init.ChainSpecPath=/w/out/genesis.json --Init.DiscoveryEnabled=false \
  --JsonRpc.Enabled=true --JsonRpc.Host=0.0.0.0 --JsonRpc.EnabledModules=eth,debug --JsonRpc.EngineHost=0.0.0.0 \
  --JsonRpc.EnginePort=8551 --JsonRpc.JwtSecretFile=/w/jwt >/dev/null
python3 - http://127.0.0.1:18545 http://127.0.0.1:18551 "$jwt" "$work/out/chain.rlp" <<'EOF'
# Builds one empty block on genesis through the Engine API and writes it as chain.rlp.
import base64, hashlib, hmac, json, sys, time, urllib.request

rpc, engine, secret, out = sys.argv[1], sys.argv[2], bytes.fromhex(sys.argv[3][2:]), sys.argv[4]

def b64(data):
    return base64.urlsafe_b64encode(data).rstrip(b"=")

def call(url, method, *params, auth=False):
    headers = {"content-type": "application/json"}
    if auth:
        claims = b64(b'{"alg":"HS256","typ":"JWT"}') + b"." + b64(json.dumps({"iat": int(time.time())}).encode())
        headers["authorization"] = "Bearer " + (claims + b"." + b64(hmac.new(secret, claims, hashlib.sha256).digest())).decode()
    body = json.dumps({"jsonrpc": "2.0", "id": 1, "method": method, "params": list(params)}).encode()
    reply = json.load(urllib.request.urlopen(urllib.request.Request(url, body, headers)))
    if "error" in reply:
        sys.exit(f"{method}: {reply['error']}")
    return reply["result"]

for _ in range(120):
    try:
        genesis = call(rpc, "eth_getBlockByNumber", "0x0", False)
        break
    except OSError:
        time.sleep(1)
else:
    sys.exit("node did not start")
root = "0x" + "00" * 32
state = {"headBlockHash": genesis["hash"], "safeBlockHash": genesis["hash"], "finalizedBlockHash": genesis["hash"]}
attrs = {"timestamp": hex(int(genesis["timestamp"], 16) + 12), "prevRandao": root, "suggestedFeeRecipient": "0x" + "00" * 20,
         "withdrawals": [], "parentBeaconBlockRoot": root}
payload_id = call(engine, "engine_forkchoiceUpdatedV3", state, attrs, auth=True)["payloadId"]
time.sleep(1)
built = call(engine, "engine_getPayloadV5", payload_id, auth=True)
payload = built["executionPayload"]
status = call(engine, "engine_newPayloadV4", payload, [], root, built["executionRequests"], auth=True)
if status["status"] != "VALID":
    sys.exit(f"engine_newPayloadV4: {status}")
head = payload["blockHash"]
call(engine, "engine_forkchoiceUpdatedV3", {"headBlockHash": head, "safeBlockHash": head, "finalizedBlockHash": head}, None, auth=True)
with open(out, "wb") as f:
    f.write(bytes.fromhex(call(rpc, "debug_getRawBlock", "0x1")[2:]))
EOF

mkdir -p "$out"
cp "$work/out/genesis.json" "$work/out/fixtures.json" "$work/out/chain.rlp" "$out/"
echo "wrote $out"
