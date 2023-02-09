module github.com/ethereum/go-ethereum

go 1.18

require (
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v0.3.0
	github.com/VictoriaMetrics/fastcache v1.6.0
	github.com/aws/aws-sdk-go-v2 v1.2.0
	github.com/aws/aws-sdk-go-v2/config v1.1.1
	github.com/aws/aws-sdk-go-v2/credentials v1.1.1
	github.com/aws/aws-sdk-go-v2/service/route53 v1.1.1
	github.com/btcsuite/btcd/btcec/v2 v2.2.0
	github.com/cespare/cp v0.1.0
	github.com/cloudflare/cloudflare-go v0.14.0
	github.com/cockroachdb/pebble v0.0.0-20230209160836-829675f94811
	github.com/consensys/gnark-crypto v0.9.1-0.20230105202408-1a7a29904a7c
	github.com/davecgh/go-spew v1.1.1
	github.com/deckarep/golang-set/v2 v2.1.0
	github.com/docker/docker v1.6.2
	github.com/dop251/goja v0.0.0-20230122112309-96b1610dd4f7
	github.com/edsrzf/mmap-go v1.0.0
	github.com/fatih/color v1.7.0
	github.com/fjl/gencodec v0.0.0-20220412091415-8bb9e558978c
	github.com/fjl/memsize v0.0.0-20190710130421-bcb5799ab5e5
	github.com/fsnotify/fsnotify v1.6.0
	github.com/gballet/go-libpcsclite v0.0.0-20190607065134-2772fd86a8ff
	github.com/gballet/go-verkle v0.0.0-20220902153445-097bd83b7732
	github.com/go-stack/stack v1.8.1
	github.com/golang-jwt/jwt/v4 v4.3.0
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.4
	github.com/google/gofuzz v1.1.1-0.20200604201612-c04b05f3adfa
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/graph-gophers/graphql-go v1.3.0
	github.com/hashicorp/go-bexpr v0.1.10
	github.com/holiman/big v0.0.0-20221017200358-a027dc42d04e
	github.com/holiman/bloomfilter/v2 v2.0.3
	github.com/holiman/uint256 v1.2.0
	github.com/huin/goupnp v1.0.3
	github.com/influxdata/influxdb v1.8.3
	github.com/influxdata/influxdb-client-go/v2 v2.4.0
	github.com/jackpal/go-nat-pmp v1.0.2
	github.com/jedisct1/go-minisign v0.0.0-20190909160543-45766022959e
	github.com/julienschmidt/httprouter v1.3.0
	github.com/karalabe/usb v0.0.2
	github.com/mattn/go-colorable v0.1.13
	github.com/mattn/go-isatty v0.0.16
	github.com/naoina/toml v0.1.2-0.20170918210437-9fafd6967416
	github.com/olekukonko/tablewriter v0.0.5
	github.com/peterh/liner v1.1.1-0.20190123174540-a2c9a5303de7
	github.com/prometheus/tsdb v0.7.1
	github.com/rs/cors v1.7.0
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible
	github.com/status-im/keycard-go v0.2.0
	github.com/stretchr/testify v1.8.0
	github.com/supranational/blst v0.3.8-0.20220526154634-513d2456b344
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/tyler-smith/go-bip39 v1.1.0
	github.com/urfave/cli/v2 v2.17.2-0.20221006022127-8f469abc00aa
	golang.org/x/crypto v0.1.0
	golang.org/x/exp v0.0.0-20230206171751-46f607a40771
	golang.org/x/sync v0.1.0
	golang.org/x/sys v0.5.0
	golang.org/x/text v0.7.0
	golang.org/x/time v0.0.0-20220922220347-f3bd1da661af
	golang.org/x/tools v0.2.0
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v0.21.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v0.8.3 // indirect
	github.com/DataDog/zstd v1.5.2 // indirect
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.0.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.0.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.1.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.1.1 // indirect
	github.com/aws/smithy-go v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cockroachdb/errors v1.9.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20220523130400-f11357ae11c7 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/deepmap/oapi-codegen v1.8.2 // indirect
	github.com/dlclark/regexp2 v1.7.0 // indirect
	github.com/garslo/gogen v0.0.0-20170306192744-1d203ffc1f61 // indirect
	github.com/getsentry/sentry-go v0.18.0 // indirect
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/influxdata/line-protocol v0.0.0-20210311194329-9aa0e372d097 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mitchellh/pointerstructure v1.2.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.39.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/mod v0.6.0 // indirect
	golang.org/x/net v0.4.0 // indirect
	golang.org/x/xerrors v0.0.0-20220517211312-f3a8303e98df // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)
