package testnet

import (
	"math/big"
	"os"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/crypto"
)

var (
	BaseXDC     = big.NewInt(0).Mul(big.NewInt(10), big.NewInt(100000000000000000)) // 1 XDC
	RpcEndpoint = "http://127.0.0.1:8545/"
	MainKey, _  = crypto.HexToECDSA(os.Getenv("MAIN_ADDRESS_KEY"))
	MainAddr    = crypto.PubkeyToAddress(MainKey.PublicKey) //0x17F2beD710ba50Ed27aEa52fc4bD7Bda5ED4a037

	// TRC21 Token
	MinTRC21Apply  = big.NewInt(0).Mul(big.NewInt(10), BaseXDC) // 10 XDC
	TRC21TokenCap  = big.NewInt(0).Mul(big.NewInt(1000000000000), BaseXDC)
	TRC21TokenFee  = big.NewInt(0)
	XDCXListingFee = big.NewInt(0).Mul(big.NewInt(1000), BaseXDC) // 1000 XDC

	// XDCX
	MaxRelayers               = big.NewInt(200)
	MaxTokenList              = big.NewInt(200)
	MinDeposit                = big.NewInt(0).Mul(big.NewInt(25000), BaseXDC) // 25000 XDC
	CollateralDepositRate     = big.NewInt(150)
	CollateralLiquidationRate = big.NewInt(110)
	CollateralRecallRate      = big.NewInt(200)
	TradeFee                  = uint16(10)  // trade fee decimals 10^4
	LendingTradeFee           = uint16(100) // lending trade fee decimals 10^4
	// 1m , 1d,7d,30d
	Terms                 = []*big.Int{big.NewInt(60), big.NewInt(86400), big.NewInt(7 * 86400), big.NewInt(30 * 86400)}
	RelayerCoinbaseKey, _ = crypto.HexToECDSA(os.Getenv("RELAYER_COINBASE_KEY")) //
	RelayerCoinbaseAddr   = crypto.PubkeyToAddress(RelayerCoinbaseKey.PublicKey) // 0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e

	OwnerRelayerKey, _ = crypto.HexToECDSA(os.Getenv("RELAYER_OWNER_KEY"))
	OwnerRelayerAddr   = crypto.PubkeyToAddress(OwnerRelayerKey.PublicKey) //0x703c4b2bD70c169f5717101CaeE543299Fc946C7

	XDCNative = common.HexToAddress("0x0000000000000000000000000000000000000001")

	TokenNameList = []string{"BTC", "ETH", "XRP", "LTC", "BNB", "ADA", "ETC", "BCH", "EOS", "USDT"}
	TeamAddresses = []common.Address{
		common.HexToAddress("0xE3584D2D430eF34FF9fEeCBEBE6E0f6980082F05"), // Test1
		common.HexToAddress("0x16a73f3a64eca79e117258e66dfd7071cc8312a9"), // BTCXDC
		common.HexToAddress("0xac177441ac2237b2f79ecff1b8f6bca39e27ef9f"), // ETHXDC
		common.HexToAddress("0x4215250e55984c75bbce8ae639b86a6cad8ec126"), // XRPXDC
		common.HexToAddress("0x6b70ca959814866dd5c426d63d47dde9cc6c32d2"), // LTCXDC
		common.HexToAddress("0x33df079fe9b9cd7fb23a1085e4eaaa8eb6952cb3"), // BNBXDC
		common.HexToAddress("0x3cab8292137804688714670640d19f9d7a60c472"), // ADAXDC
		common.HexToAddress("0x9415d953d47c5f155cac9de7b24a756f352eafbf"), // ETCXDC
		common.HexToAddress("0xe32d2e7c8e8809e45c8e2332830b48d9e231e3f2"), // BCHXDC
		common.HexToAddress("0xf76ddbda664ea47088937e1cf9ff15036714dee3"), // EOSXDC
		common.HexToAddress("0xc465ee82440dada9509feb235c7cd7d896acf13c"), // ETHBTC
		common.HexToAddress("0xb95bdc136c579dc3fd2b2424a8e925a90228d2c2"), // XRPBTC
		common.HexToAddress("0xe36c1842365595D44854eEcd64B11c8115E133EF"), // XDCUSDT
		common.HexToAddress("0xaaC1959F6F0fb539F653409079Ec4146267B7555"), // BTCUSDT
		common.HexToAddress("0x726DA688e2e09f01A2e1aB4c10F25B7CEdD4a0f3"), // ETHUSDT

	}

	Required = big.NewInt(2)
	Owners   = []common.Address{
		common.HexToAddress("0x244e17B2141288a6F00E79E8feC2341f827d156f"),
		common.HexToAddress("0xd106159eC58BD2EAf5B62eF4e9cDb286170B0Bb9"),
		common.HexToAddress("0x0197BE034Bf0Bd2b3adDC84366a5681Bb7545888"),
	}
)
