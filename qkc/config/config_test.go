// Ported verbatim from github.com/QuarkChain/goquarkchain/cluster/config (byte-compatible).
// Adaptation: the python cluster_config_template.json lives in ./testdata/ here
// (was ../../tests/testdata/testnet/ in goquarkchain).

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/qkc/account"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/assert"
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

func TestClusterConfig(t *testing.T) {
	var (
		chainSize         uint32 = 2
		shardSizePerChain uint32 = 4
	)
	cluster := NewClusterConfig()
	jsonConfig, err := json.Marshal(cluster)
	if err != nil {
		t.Fatalf("cluster struct marshal error: %v", err)
	}

	// Make sure reward tax rate is correctly marshalled
	if !strings.Contains(string(jsonConfig), "\"REWARD_TAX_RATE\":0.5") {
		t.Error("reward tax rate is not correctly marshalled")
	}

	var c ClusterConfig
	err = json.Unmarshal(jsonConfig, &c)
	if err != nil {
		t.Fatalf("UnMarsshal cluster config error: %v", err)
	}
	if c.DbPathRoot != "./db" {
		t.Fatalf("db path root error")
	}

	_, err = c.GetSlaveConfig("S0")
	if err != nil {
		t.Fatalf("slave should not to be empty: %v", err)
	}
	if c.P2P == nil {
		t.Fatalf("")
	}
	quarkchain := c.Quarkchain
	if quarkchain.RewardTaxRate.Cmp(new(big.Rat).SetFloat64(0.5)) != 0 {
		t.Errorf("wrong marshaling of reward tax rate")
	}

	shardIds := quarkchain.GetGenesisShardIds()
	// make sure the default chainsize and shardsize
	if len(shardIds) != 2*3 {
		t.Fatalf("shard id list is not enough.")
	}
	for _, fullShardId := range shardIds {
		if quarkchain.GetGenesisRootHeight(fullShardId) != 0 {
			t.Fatalf("genesis height is not equal to 0.")
		}
	}
	initializeIds := quarkchain.GetInitializedShardIdsBeforeRootHeight(0)
	if len(initializeIds) != 0 {
		t.Fatalf("the list of ids should be empty.")
	}
	quarkchain.Update(chainSize, shardSizePerChain, 10, 10)
	fullShardIDByConfig, err := quarkchain.GetShardSizeByChainId(1)
	if err != nil {
		panic(err)
	}
	if fullShardIDByConfig != 4 {
		t.Fatalf("quarkchain update function set shard size failed, shard size: %d", fullShardIDByConfig)
	}
}

func TestSlaveConfig(t *testing.T) {
	s := []byte(`{
		"IP": "1.2.3.4",
		"PORT": 123,
		"ID": "S1",
		"FULL_SHARD_ID_LIST": ["0x00000004"]
	}`)

	var sc SlaveConfig
	assert.NoError(t, json.Unmarshal(s, &sc))
	assert.Equal(t, uint32(4), sc.FullShardList[0])

	jsonConfig, err := json.Marshal(&sc)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(jsonConfig), "FULL_SHARD_ID_LIST\":[\"0x4\"]"))
}

func TestLoadClusterConfig(t *testing.T) {
	var (
		goClstr ClusterConfig
		pyClstr ClusterConfig
	)
	if err := loadConfig("./test_config.json", &goClstr); err != nil {
		t.Fatalf("Failed to load json file, err: %v", err)
	}

	if err := loadConfig("./testdata/cluster_config_template.json", &pyClstr); err != nil {
		t.Fatalf("Failed to load python config file, err: %v", err)
	}

	if !reflect.DeepEqual(goClstr.SlaveList, pyClstr.SlaveList) {
		t.Fatalf("go config slave list is not equal to python config")
	}
	if goClstr.Quarkchain.ChainSize != pyClstr.Quarkchain.ChainSize {
		t.Fatalf("go config chain size is not equal to python config")
	}
	for i, goChain := range goClstr.Quarkchain.Chains {
		pyCHain := pyClstr.Quarkchain.Chains[i]
		if goChain.ChainID != pyCHain.ChainID {
			t.Fatalf("go config chain size is not equal to python config")
		}
		if goChain.ShardSize != pyCHain.ShardSize {
			t.Fatalf("go config chain size is not equal to python config")
		}
		// NOTE: upstream goquarkchain (cluster/config/config_test.go:142-150) also
		// compares Genesis/PoswConfig/ConsensusConfig here, but the comparisons are
		// inverted (missing "!") and so validate nothing. They cannot be corrected
		// by adding "!": test_config.json and the python cluster_config_template.json
		// deliberately differ on those value fields, so equality would always fail.
		// This test only verifies that the Go and Python configs describe the same
		// cluster topology (slave list, chain count, per-chain CHAIN_ID/SHARD_SIZE).
	}
}

// TestShardConfigDerivation checks that loading a cluster config derives each
// per-shard config the way pyquarkchain's ClusterConfig.from_dict does (and
// unlike goquarkchain, which copies the chain coinbase and the full alloc
// verbatim): the coinbase is rewritten into the shard, and GENESIS.ALLOC is
// filtered down to the addresses that belong to the shard. The goshard slave
// shares its config with a pyquarkchain master, so it must match pyquarkchain.
// See qkc/config/cluster_config.go (UnmarshalJSON).
func TestShardConfigDerivation(t *testing.T) {
	var c ClusterConfig
	if err := loadConfig("./test_config.json", &c); err != nil {
		t.Fatalf("Failed to load json file, err: %v", err)
	}
	q := c.Quarkchain

	// (1) Coinbase rewritten into each shard: full-shard-key == the shard's full
	// shard id, recipient preserved from the chain coinbase.
	for fsid, sc := range q.shards {
		if sc.CoinbaseAddress.FullShardKey != fsid {
			t.Errorf("shard %d: coinbase full-shard-key = %d, want %d", fsid, sc.CoinbaseAddress.FullShardKey, fsid)
		}
		if want := q.Chains[sc.ChainID].CoinbaseAddress.Recipient; sc.CoinbaseAddress.Recipient != want {
			t.Errorf("shard %d: coinbase recipient = %x, want %x (chain coinbase)", fsid, sc.CoinbaseAddress.Recipient, want)
		}
	}

	// (2) GENESIS.ALLOC filtered by shard. In test_config.json only chain 2 has
	// allocations (3 addresses, all with shard bits 0). With SHARD_SIZE=2 they all
	// belong to shard 0, so shard 1 is filtered down to none.
	shard0 := uint32(2)<<16 | 2 | 0
	shard1 := uint32(2)<<16 | 2 | 1
	if got := len(q.shards[shard0].Genesis.Alloc); got != 3 {
		t.Errorf("chain 2 shard 0 alloc count = %d, want 3", got)
	}
	if got := len(q.shards[shard1].Genesis.Alloc); got != 0 {
		t.Errorf("chain 2 shard 1 alloc count = %d, want 0 (filtered out)", got)
	}
}

func TestShardGenesis(t *testing.T) {
	var (
		shardGensis ShardGenesis
	)
	s := []byte(`{"ROOT_HEIGHT":0,"VERSION":0,"HEIGHT":0,"HASH_PREV_MINOR_BLOCK":"0000000000000000000000000000000000000000000000000000000000000000","HASH_MERKLE_ROOT":"0000000000000000000000000000000000000000000000000000000000000000","TIMESTAMP":1519147489,"DIFFICULTY":5000000000,"GAS_LIMIT":12000000,"NONCE":0,"EXTRA_DATA":"497420776173207468652062657374206f662074696d65732c206974207761732074686520776f727374206f662074696d65732c202e2e2e202d20436861726c6573204469636b656e73","ALLOC":{}}`)
	assert.NoError(t, json.Unmarshal(s, &shardGensis))
	assert.Equal(t, common.FromHex("497420776173207468652062657374206f662074696d65732c206974207761732074686520776f727374206f662074696d65732c202e2e2e202d20436861726c6573204469636b656e73"), shardGensis.ExtraData)
	jsonConfig, err := json.Marshal(&shardGensis)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonConfig), string(s))
}

func TestShardGenesisAlloc(t *testing.T) {
	s := []byte(`{"ROOT_HEIGHT":0,"VERSION":0,"HEIGHT":0,
		"HASH_PREV_MINOR_BLOCK":"0000000000000000000000000000000000000000000000000000000000000000",
		"HASH_MERKLE_ROOT":"0000000000000000000000000000000000000000000000000000000000000000",
		"TIMESTAMP":1519147489,"DIFFICULTY":5000000000,"GAS_LIMIT":12000000,"NONCE":0,
		"EXTRA_DATA":"49742077","ALLOC":{"8e3B4695B15aC4Ef6DA92C6141Def52d65Ba897400000000": {
		   "balances": {
						"QKC": 600000000000000000000000000,
						"QI": 600000000000000000000000000
					  },
		   "code": "608060405260043610610112576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806306fdde0314610117578063095ea7b3146101a757806318160ddd1461020c57806323b872dd14610237578063313ce567146102bc57806342966c68146102ed57806370a082311461033257806379c650681461038957806379cc6790146103d657806385e436bf1461043b5780638da5cb5b1461046857806395d89b41146104bf578063a6f2ae3a1461054f578063a9059cbb14610559578063b414d4b6146105be578063cae9ca5114610619578063dd62ed3e146106c4578063e724529c1461073b578063f2fde38b1461078a578063fc37987b146107cd575b600080fd5b34801561012357600080fd5b5061012c6107f8565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561016c578082015181840152602081019050610151565b50505050905090810190601f1680156101995780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b3480156101b357600080fd5b506101f2600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610896565b604051808215151515815260200191505060405180910390f35b34801561021857600080fd5b50610221610988565b6040518082815260200191505060405180910390f35b34801561024357600080fd5b506102a2600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019092919050505061098e565b604051808215151515815260200191505060405180910390f35b3480156102c857600080fd5b506102d1610abb565b604051808260ff1660ff16815260200191505060405180910390f35b3480156102f957600080fd5b5061031860048036038101908080359060200190929190505050610ace565b604051808215151515815260200191505060405180910390f35b34801561033e57600080fd5b50610373600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610bd2565b6040518082815260200191505060405180910390f35b34801561039557600080fd5b506103d4600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610bea565b005b3480156103e257600080fd5b50610421600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610d5b565b604051808215151515815260200191505060405180910390f35b34801561044757600080fd5b5061046660048036038101908080359060200190929190505050610f75565b005b34801561047457600080fd5b5061047d610fda565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156104cb57600080fd5b506104d4610fff565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156105145780820151818401526020810190506104f9565b50505050905090810190601f1680156105415780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b61055761109d565b005b34801561056557600080fd5b506105a4600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506110b4565b604051808215151515815260200191505060405180910390f35b3480156105ca57600080fd5b506105ff600480360381019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506110cb565b604051808215151515815260200191505060405180910390f35b34801561062557600080fd5b506106aa600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190803590602001908201803590602001908080601f01602080910402602001604051908101604052809392919081815260200183838082843782019150505050505091929192905050506110eb565b604051808215151515815260200191505060405180910390f35b3480156106d057600080fd5b50610725600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061126e565b6040518082815260200191505060405180910390f35b34801561074757600080fd5b50610788600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803515159060200190929190505050611293565b005b34801561079657600080fd5b506107cb600480360381019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506113b8565b005b3480156107d957600080fd5b506107e2611456565b6040518082815260200191505060405180910390f35b60018054600181600116156101000203166002900480601f01602080910402602001604051908101604052809291908181526020018280546001816001161561010002031660029004801561088e5780601f106108635761010080835404028352916020019161088e565b820191906000526020600020905b81548152906001019060200180831161087157829003601f168201915b505050505081565b600081600660003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040518082815260200191505060405180910390a36001905092915050565b60045481565b6000600660008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548211151515610a1b57600080fd5b81600660008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282540392505081905550610ab084848461145c565b600190509392505050565b600360009054906101000a900460ff1681565b600081600560003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410151515610b1e57600080fd5b81600560003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282540392505081905550816004600082825403925050819055503373ffffffffffffffffffffffffffffffffffffffff167fcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca5836040518082815260200191505060405180910390a260019050919050565b60056020528060005260406000206000915090505481565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610c4557600080fd5b80600560008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282540192505081905550806004600082825401925050819055503073ffffffffffffffffffffffffffffffffffffffff1660007fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a38173ffffffffffffffffffffffffffffffffffffffff163073ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a35050565b600081600560008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410151515610dab57600080fd5b600660008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548211151515610e3657600080fd5b81600560008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254039250508190555081600660008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282540392505081905550816004600082825403925050819055508273ffffffffffffffffffffffffffffffffffffffff167fcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca5836040518082815260200191505060405180910390a26001905092915050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610fd057600080fd5b8060078190555050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60028054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156110955780601f1061106a57610100808354040283529160200191611095565b820191906000526020600020905b81548152906001019060200180831161107857829003601f168201915b505050505081565b6000600754340290506110b130338361145c565b50565b60006110c133848461145c565b6001905092915050565b60086020528060005260406000206000915054906101000a900460ff1681565b6000808490506110fb8585610896565b15611265578073ffffffffffffffffffffffffffffffffffffffff16638f4ffcb1338630876040518563ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018481526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200180602001828103825283818151815260200191508051906020019080838360005b838110156111f55780820151818401526020810190506111da565b50505050905090810190601f1680156112225780820380516001836020036101000a031916815260200191505b5095505050505050600060405180830381600087803b15801561124457600080fd5b505af1158015611258573d6000803e3d6000fd5b5050505060019150611266565b5b509392505050565b6006602052816000526040600020602052806000526040600020600091509150505481565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156112ee57600080fd5b80600860008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff0219169083151502179055507f48335238b4855f35377ed80f164e8c6f3c366e54ac00b96a6402d4a9814a03a58282604051808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001821515151581526020019250505060405180910390a15050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561141357600080fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b60075481565b60008273ffffffffffffffffffffffffffffffffffffffff161415151561148257600080fd5b80600560008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054101515156114d057600080fd5b600560008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205481600560008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054011015151561155f57600080fd5b600860008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff161515156115b857600080fd5b600860008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff1615151561161157600080fd5b80600560008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254039250508190555080600560008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055508173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a35050505600a165627a7a723058209ea8ef14a95f9f6eb9d74c04b3e95b395dc0a4dcfb2030002bfd51d26294805d0029",
		   "storage": {
				"0x00": "0x6d3af223727309928cefcd8303a892dd0e4a3e95",
				"0x02": "0x5042000000000000000000000000000000000000000000000000000000000004",
				"0x04": "0x204fce5e3e25026110000000",
				"0x3e32c5f924af070531924c4a313f23e54e250d52155e88c663a72280f5eef390": "0x204fce5e3e25026110000000",
				"0x07": "0x0a",
				"0x01": "0x50656f706c65426f6f6b00000000000000000000000000000000000000000014",
				"0x03": "0x12"
		  }
		}}}`)
	var shardGensis ShardGenesis
	assert.NoError(t, json.Unmarshal(s, &shardGensis))
	address, err := account.CreatAddressFromBytes(common.FromHex("0x8e3B4695B15aC4Ef6DA92C6141Def52d65Ba897400000000"))
	assert.NoError(t, err)
	assert.NotNil(t, shardGensis.Alloc[address])
	val := shardGensis.Alloc[address].Storage[common.HexToHash("0x01")]
	assert.Equal(t, common.FromHex("0x50656f706c65426f6f6b00000000000000000000000000000000000000000014"), val[:])
	jsonConfig, err := json.Marshal(&shardGensis)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonConfig), `608060405260043610610112576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff`)
}

func loadConfig(file string, cfg *ClusterConfig) error {
	var (
		content []byte
		err     error
	)
	if content, err = ioutil.ReadFile(file); err != nil {
		return errors.New(file + ", " + err.Error())
	}
	return json.Unmarshal(content, cfg)
}

func TestGetRootPOSWConfigDiffDivider(t *testing.T) {
	blockTime := uint64(1646064000)
	config := NewRootPOSWConfig()
	config.BoostTimestamp = 0
	if config.DiffDivider != config.GetDiffDivider(blockTime) {
		t.Fatalf("Boost has been disable and GetDiffDivider %d should equal to DiffDivider %d.",
			config.DiffDivider, config.GetDiffDivider(blockTime))
	}
	config.BoostTimestamp = blockTime + 1
	if config.DiffDivider != config.GetDiffDivider(blockTime) {
		t.Fatalf("Boost does not enable as blockTime < BoostTimestamp, GetDiffDivider %d should equal to DiffDivider %d.",
			config.DiffDivider, config.GetDiffDivider(blockTime))
	}
	config.BoostTimestamp = blockTime - 1
	if config.DiffDivider*config.BoostMultiplierPerStep != config.GetDiffDivider(blockTime) {
		t.Fatalf("Boost has been enable and GetDiffDivider %d should equal to DiffDivider * BoostStepInterval %d",
			config.GetDiffDivider(blockTime), config.DiffDivider*config.BoostMultiplierPerStep)
	}
	config.BoostTimestamp = blockTime - config.BoostStepInterval*config.BoostSteps + 1
	if config.DiffDivider*pow(config.BoostMultiplierPerStep, config.BoostSteps) != config.GetDiffDivider(blockTime) {
		t.Fatalf("Boost has been enable and GetDiffDivider %d should equal to DiffDivider * pow (BoostStepInterval, config.BoostSteps) %d",
			config.GetDiffDivider(blockTime), config.DiffDivider*pow(config.BoostMultiplierPerStep, config.BoostSteps))
	}
	config.BoostTimestamp = blockTime - config.BoostStepInterval*(config.BoostSteps+1) - 1
	if config.DiffDivider*pow(config.BoostMultiplierPerStep, config.BoostSteps) != config.GetDiffDivider(blockTime) {
		t.Fatalf("Boost has been enable and GetDiffDivider %d should equal to DiffDivider * pow (BoostStepInterval, config.BoostSteps) %d",
			config.GetDiffDivider(blockTime), config.DiffDivider*pow(config.BoostMultiplierPerStep, config.BoostSteps))
	}
}
