module github.com/ethereum/go-ethereum

go 1.21

require (
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v0.3.0
	github.com/BurntSushi/toml v1.2.0
	github.com/JekaMas/go-grpc-net-conn v0.0.0-20220708155319-6aff21f2d13d
	github.com/JekaMas/workerpool v1.1.8
	github.com/VictoriaMetrics/fastcache v1.6.0
	github.com/aws/aws-sdk-go-v2 v1.21.0
	github.com/aws/aws-sdk-go-v2/config v1.18.43
	github.com/aws/aws-sdk-go-v2/credentials v1.13.41
	github.com/aws/aws-sdk-go-v2/service/route53 v1.29.5
	github.com/btcsuite/btcd/btcec/v2 v2.3.2
	github.com/cespare/cp v1.1.1
	github.com/cloudflare/cloudflare-go v0.14.0
	github.com/cockroachdb/pebble v0.0.0-20230209160836-829675f94811
	github.com/consensys/gnark-crypto v0.12.0
	github.com/cosmos/cosmos-sdk v0.47.3
	github.com/davecgh/go-spew v1.1.1
	github.com/deckarep/golang-set/v2 v2.1.0
	github.com/docker/docker v24.0.5+incompatible
	github.com/dop251/goja v0.0.0-20230122112309-96b1610dd4f7
	github.com/edsrzf/mmap-go v1.0.0
	github.com/emirpasic/gods v1.18.1
	github.com/fatih/color v1.13.0
	github.com/fjl/gencodec v0.0.0-20220412091415-8bb9e558978c
	github.com/fjl/memsize v0.0.0-20190710130421-bcb5799ab5e5
	github.com/fsnotify/fsnotify v1.6.0
	github.com/gballet/go-libpcsclite v0.0.0-20191108122812-4678299bea08
	github.com/gballet/go-verkle v0.0.0-20220902153445-097bd83b7732
	github.com/go-stack/stack v1.8.1
	github.com/gofrs/flock v0.8.1
	github.com/golang-jwt/jwt/v4 v4.3.0
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.3
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb
	github.com/google/gofuzz v1.2.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.5.0
	github.com/graph-gophers/graphql-go v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/hashicorp/go-bexpr v0.1.10
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/hashicorp/hcl/v2 v2.10.1
	github.com/heimdalr/dag v1.2.1
	github.com/holiman/bloomfilter/v2 v2.0.3
	github.com/holiman/uint256 v1.2.2-0.20230321075855-87b91420868c
	github.com/huin/goupnp v1.0.3
	github.com/imdario/mergo v0.3.11
	github.com/influxdata/influxdb-client-go/v2 v2.4.0
	github.com/influxdata/influxdb1-client v0.0.0-20220302092344-a9ab5670611c
	github.com/jackpal/go-nat-pmp v1.0.2
	github.com/jedisct1/go-minisign v0.0.0-20190909160543-45766022959e
	github.com/json-iterator/go v1.1.12
	github.com/julienschmidt/httprouter v1.3.0
	github.com/karalabe/usb v0.0.2
	github.com/kylelemons/godebug v1.1.0
	github.com/maticnetwork/crand v1.0.2
	github.com/maticnetwork/heimdall v0.3.1-0.20230227104835-81bd1055b0bc
	github.com/maticnetwork/polyproto v0.0.3-0.20230216113155-340ea926ca53
	github.com/mattn/go-colorable v0.1.13
	github.com/mattn/go-isatty v0.0.17
	github.com/mitchellh/cli v1.1.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/pelletier/go-toml v1.9.5
	github.com/peterh/liner v1.2.0
	github.com/rs/cors v1.8.2
	github.com/ryanuber/columnize v2.1.2+incompatible
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible
	github.com/status-im/keycard-go v0.2.0
	github.com/stretchr/testify v1.8.4
	github.com/supranational/blst v0.3.11
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/tendermint/tendermint v0.34.21
	github.com/tyler-smith/go-bip39 v1.1.0
	github.com/urfave/cli/v2 v2.17.2-0.20221006022127-8f469abc00aa
	golang.org/x/crypto v0.11.0
	golang.org/x/exp v0.0.0-20230206171751-46f607a40771
	golang.org/x/sync v0.3.0
	golang.org/x/sys v0.12.0
	golang.org/x/text v0.11.0
	golang.org/x/time v0.3.0
	golang.org/x/tools v0.10.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce
)

require (
	cloud.google.com/go/compute v1.21.0 // indirect
	cloud.google.com/go/iam v1.1.1 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pelletier/go-toml/v2 v2.0.5 // indirect
	github.com/subosito/gotenv v1.4.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)

require (
	cloud.google.com/go v0.110.4 // indirect
	cloud.google.com/go/pubsub v1.32.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v0.21.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v0.8.3 // indirect
	github.com/DataDog/zstd v1.5.2 // indirect
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46 // indirect
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/aws/aws-sdk-go v1.40.45 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.23.0 // indirect
	github.com/aws/smithy-go v1.14.2 // indirect
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
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/influxdata/line-protocol v0.0.0-20210311194329-9aa0e372d097 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/pointerstructure v1.2.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.39.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	github.com/xsleonard/go-merkle v1.1.0
	go.opentelemetry.io/otel v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.19.0
	go.opentelemetry.io/otel/sdk v1.19.0
	go.uber.org/goleak v1.2.1
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/net v0.12.0 // indirect
	gonum.org/v1/gonum v0.11.0
	google.golang.org/grpc v1.58.2
	google.golang.org/protobuf v1.31.0
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools v2.2.0+incompatible
	pgregory.net/rapid v0.4.8
	rsc.io/tmplfunc v0.0.3 // indirect
)

require (
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.41 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.35 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.43 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.17.1 // indirect
	github.com/bartekn/go-bip39 v0.0.0-20171116152956-a05967ea095d // indirect
	github.com/bgentry/speakeasy v0.1.0 // indirect
	github.com/bits-and-blooms/bitset v1.7.0 // indirect
	github.com/bradfitz/gomemcache v0.0.0-20190913173617-a41fca850d0b // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce // indirect
	github.com/cbergoon/merkletree v0.2.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cosmos/go-bip39 v1.0.0 // indirect
	github.com/cosmos/ledger-cosmos-go v0.13.0 // indirect
	github.com/etcd-io/bbolt v1.3.3 // indirect
	github.com/gammazero/deque v0.2.1 // indirect
	github.com/go-kit/kit v0.12.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-redis/redis v6.15.7+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/s2a-go v0.1.4 // indirect
	github.com/googleapis/gax-go/v2 v2.11.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-wordwrap v0.0.0-20150314170334-ad45545899c7 // indirect
	github.com/mitchellh/reflectwalk v1.0.0 // indirect
	github.com/posener/complete v1.2.3 // indirect
	github.com/rakyll/statik v0.1.7 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/cobra v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.13.0 // indirect
	github.com/streadway/amqp v1.0.0 // indirect
	github.com/stumble/gorocksdb v0.0.3 // indirect
	github.com/tendermint/btcd v0.1.1 // indirect
	github.com/tendermint/crypto v0.0.0-20191022145703-50d29ede1e15 // indirect
	github.com/tendermint/go-amino v0.16.0 // indirect
	github.com/tendermint/iavl v0.12.4 // indirect
	github.com/tendermint/tm-db v0.6.7 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	github.com/zondax/ledger-go v0.14.1 // indirect
	go.opentelemetry.io/otel/metric v1.19.0 // indirect
	golang.org/x/oauth2 v0.10.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/RichardKnop/logging v0.0.0-20190827224416-1a693bdd4fae // indirect
	github.com/RichardKnop/machinery v1.7.4 // indirect
	github.com/RichardKnop/redsync v1.2.0 // indirect
	github.com/zclconf/go-cty v1.8.0 // indirect
	github.com/zondax/hid v0.9.1 // indirect
	go.mongodb.org/mongo-driver v1.3.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.19.0 // indirect
	go.opentelemetry.io/otel/trace v1.19.0
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	google.golang.org/api v0.126.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230711160842-782d3b101e98 // indirect
)

replace github.com/cosmos/cosmos-sdk => github.com/maticnetwork/cosmos-sdk v0.37.5-0.20231005133937-b1eb1f90feb7

replace github.com/tendermint/tendermint => github.com/maticnetwork/tendermint v0.26.0-dev0.0.20231005133805-2bb6a831bb2e

replace github.com/tendermint/tm-db => github.com/tendermint/tm-db v0.2.0

replace github.com/ethereum/go-ethereum => github.com/maticnetwork/bor v1.0.4

replace github.com/Masterminds/goutils => github.com/Masterminds/goutils v1.1.1

replace go.mongodb.org/mongo-driver => go.mongodb.org/mongo-driver v1.5.1
