package main

import (
	"context"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/senatus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/urfave/cli.v1"
)

var (
	createValidatorCommand = cli.Command{
		Action:    utils.MigrateFlags(createValidator),
		Name:      "validator.create",
		Usage:     "Create a new validator",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.KeyStoreDirFlag,
			utils.PasswordFileFlag,
			utils.FromAddressFlag,
			utils.ValidatorRewardAddrFlag,
			utils.ValidatorMonikerFlag,
			utils.ValidatorWebsiteFlag,
			utils.ValidatorEmailFlag,
			utils.ValidatorDetailFlag,
			utils.ValidatorStakingFlag,
			utils.NodeEndpointFlag,
			utils.GasLimitFlag,
			utils.GasPriceFlag,
			utils.NonceFlag,
		},
		Category: "VALIDATOR",
		Description: `
The validator.create command creates a new validator.
		`,
	}

	editValidatorCommand = cli.Command{
		Action:    utils.MigrateFlags(editValidator),
		Name:      "validator.edit",
		Usage:     "Edit a existing validator",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.KeyStoreDirFlag,
			utils.PasswordFileFlag,
			utils.FromAddressFlag,
			utils.ValidatorRewardAddrFlag,
			utils.ValidatorMonikerFlag,
			utils.ValidatorWebsiteFlag,
			utils.ValidatorEmailFlag,
			utils.ValidatorDetailFlag,
			utils.NodeEndpointFlag,
			utils.GasLimitFlag,
			utils.GasPriceFlag,
			utils.NonceFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	stakingCommand = cli.Command{
		Action:    utils.MigrateFlags(staking),
		Name:      "staking",
		Usage:     "staking some cet to a existing validator",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.KeyStoreDirFlag,
			utils.PasswordFileFlag,
			utils.FromAddressFlag,
			utils.ValidatorAddressFlag,
			utils.ValidatorStakingFlag,
			utils.NodeEndpointFlag,
			utils.GasLimitFlag,
			utils.GasPriceFlag,
			utils.NonceFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	unstakingCommand = cli.Command{
		Action:    utils.MigrateFlags(unstaking),
		Name:      "unstaking",
		Usage:     "Unstaking from a validator",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.KeyStoreDirFlag,
			utils.PasswordFileFlag,
			utils.FromAddressFlag,
			utils.ValidatorAddressFlag,
			utils.NodeEndpointFlag,
			utils.GasLimitFlag,
			utils.GasPriceFlag,
			utils.NonceFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	withdrawRewardCommand = cli.Command{
		Action:    utils.MigrateFlags(withdrawReward),
		Name:      "withdrawreward",
		Usage:     "Withdraw reward from a validator",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.KeyStoreDirFlag,
			utils.PasswordFileFlag,
			utils.FromAddressFlag,
			utils.ValidatorAddressFlag,
			utils.NodeEndpointFlag,
			utils.GasLimitFlag,
			utils.GasPriceFlag,
			utils.NonceFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	withdrawStakingCommand = cli.Command{
		Action:    utils.MigrateFlags(withdrawStaking),
		Name:      "withdrawstake",
		Usage:     "withdraw staking cet from a validator",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.KeyStoreDirFlag,
			utils.PasswordFileFlag,
			utils.FromAddressFlag,
			utils.ValidatorAddressFlag,
			utils.NodeEndpointFlag,
			utils.GasLimitFlag,
			utils.GasPriceFlag,
			utils.NonceFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	unjailCommand = cli.Command{
		Action:    utils.MigrateFlags(unjail),
		Name:      "unjail",
		Usage:     "Unjail validator if validator is jailed",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.KeyStoreDirFlag,
			utils.PasswordFileFlag,
			utils.FromAddressFlag,
			utils.NodeEndpointFlag,
			utils.GasLimitFlag,
			utils.GasPriceFlag,
			utils.NonceFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	validatorDespQueryCommand = cli.Command{
		Action:    utils.MigrateFlags(queryValidatorDescription),
		Name:      "validator.description.query",
		Usage:     "query validator's description(moniker, identity, website etc)",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.ValidatorAddressFlag,
			utils.NodeEndpointFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	validatorInfoQueryCommand = cli.Command{
		Action:    utils.MigrateFlags(queryValidatorInfo),
		Name:      "validator.info.query",
		Usage:     "query validator's info(rewardaddr, status, stakingAmount etd)",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.ValidatorAddressFlag,
			utils.NodeEndpointFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	activatedValidatorsListCommand = cli.Command{
		Action:    utils.MigrateFlags(queryActivatedValidators),
		Name:      "validator.activated.query",
		Usage:     "query activated validators",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.NodeEndpointFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	validatorCandidatorsListCommand = cli.Command{
		Action:    utils.MigrateFlags(queryValidatorCandidators),
		Name:      "validator.candidators.query",
		Usage:     "query validator candidators",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.NodeEndpointFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	stakingInfoQueryCommand = cli.Command{
		Action:    utils.MigrateFlags(queryStakingInfo),
		Name:      "validator.staking.query",
		Usage:     "query a address staking info",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.ValidatorAddressFlag,
			utils.StakerAddressFlag,
			utils.NodeEndpointFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	slashRecordQueryCommand = cli.Command{
		Action:    utils.MigrateFlags(querySlashRecord),
		Name:      "validator.slash.record",
		Usage:     "query a validator slash record info",
		ArgsUsage: "",
		Flags: []cli.Flag{
			utils.ValidatorAddressFlag,
			utils.NodeEndpointFlag,
		},
		Category:    "VALIDATOR",
		Description: ``,
	}

	defaultGasPrice = big.NewInt(params.MinimalGasPrice.Int64()) // 500GWEI
)

const defaultNodeHttpHost = "http://127.0.0.1:8545"
const nodeTimeout = 60 * time.Second

type validator struct {
	rewardAddr common.Address
	moniker    string
	website    string
	email      string
	details    string
}

const (
	ValidatorCreateMethod = "create"
	ValidatorEditMethod   = "edit"
	StakeMethod           = "stake"
	UnstakeMethod         = "unstake"
	UnjailedMethod        = "unjailed"
	WithdrawStakingMethod = "withdrawStaking"
	WithdrawRewardsMethod = "withdrawRewards"

	GetValidatorDespMethod        = "getValidatorDescription"
	GetValidatorInfoMethod        = "getValidatorInfo"
	GetStakingInfoMethod          = "getStakingInfo"
	GetValidatorCandidatesMethod  = "getValidatorCandidate"
	GetActivatedValidatorsMethod  = "getActivatedValidators"
	GetValidatorSlashRecordMethod = "getSlashRecord"
)

const (
	ValidatorNotExist      = "NotExist"
	ValidatorCreatedStatus = "created"
	ValidatorStakeStatus   = "staked"
	ValidatorUnstakeStatus = "unstake"
	ValidatorJailed        = "jailed"
)

var validatorStatusMap = map[uint8]string{
	0x0: ValidatorNotExist,
	0x1: ValidatorCreatedStatus,
	0x2: ValidatorStakeStatus,
	0x3: ValidatorUnstakeStatus,
	0x4: ValidatorJailed,
}

func makeValidatorInfo(ctx *cli.Context) *validator {
	val := validator{}
	if ctx.GlobalIsSet(utils.ValidatorRewardAddrFlag.Name) {
		val.rewardAddr = common.HexToAddress(strings.TrimSpace(ctx.GlobalString(utils.ValidatorRewardAddrFlag.Name)))
	} else {
		if !ctx.GlobalIsSet(utils.FromAddressFlag.Name) {
			utils.Fatalf("create or edit validator must set transaction's from address")
		}
		val.rewardAddr = common.HexToAddress(strings.TrimSpace(ctx.GlobalString(utils.FromAddressFlag.Name)))
	}

	if ctx.GlobalIsSet(utils.ValidatorMonikerFlag.Name) {
		val.moniker = strings.TrimSpace(ctx.GlobalString(utils.ValidatorMonikerFlag.Name))
	}

	if ctx.GlobalIsSet(utils.ValidatorWebsiteFlag.Name) {
		val.website = strings.TrimSpace(ctx.GlobalString(utils.ValidatorWebsiteFlag.Name))
	}

	if ctx.GlobalIsSet(utils.ValidatorEmailFlag.Name) {
		val.email = strings.TrimSpace(ctx.GlobalString(utils.ValidatorEmailFlag.Name))
	}

	if ctx.GlobalIsSet(utils.ValidatorDetailFlag.Name) {
		val.details = strings.TrimSpace(ctx.GlobalString(utils.ValidatorDetailFlag.Name))
	}

	return &val
}

func defaultKeystorDir() string {
	dataDir := node.DefaultDataDir()

	return filepath.Join(dataDir, "keystore")
}

func validatorStatus(status uint8) string {
	if _, ok := validatorStatusMap[status]; ok {
		return validatorStatusMap[status]
	}
	return "Unknow"
}

func transactionHandler(ctx *cli.Context, msg *ethereum.CallMsg) error {
	if !ctx.GlobalIsSet(utils.FromAddressFlag.Name) {
		utils.Fatalf("transaction's from address must set")
	}
	fromAddress := common.HexToAddress(strings.TrimSpace(ctx.GlobalString(utils.FromAddressFlag.Name)))
	msg.From = fromAddress
	var nodeHost string
	if ctx.GlobalIsSet(utils.NodeEndpointFlag.Name) {
		nodeHost = strings.TrimSpace(ctx.GlobalString(utils.NodeEndpointFlag.Name))
	} else {
		nodeHost = defaultNodeHttpHost
	}

	clientCtx, cancel := context.WithTimeout(context.Background(), nodeTimeout)
	defer cancel()

	client, err := ethclient.Dial(nodeHost)
	if err != nil {
		utils.Fatalf("connect to %s error: %v\n", nodeHost, err)
	}

	chainID, err := client.ChainID(clientCtx)
	if err != nil {
		utils.Fatalf("Get chain id from %s error: %v\n", nodeHost, err)
	}

	var nonce uint64
	if ctx.GlobalIsSet(utils.NonceFlag.Name) {
		nonce = ctx.GlobalUint64(utils.NonceFlag.Name)
	} else {
		nonce, err = client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			utils.Fatalf("Get adddress: %s nonce error: %v\n", fromAddress, err)
		}
	}

	var gasLimit uint64
	gasPrice := big.NewInt(0)
	if ctx.GlobalIsSet(utils.GasLimitFlag.Name) {
		gasLimit = ctx.GlobalUint64(utils.GasLimitFlag.Name)
	} else {
		gasLimit, err = client.EstimateGas(context.Background(), *msg)
		if err != nil {
			utils.Fatalf("EstimateGas error: %v\n", err)
		}
	}

	if ctx.GlobalIsSet(utils.GasPriceFlag.Name) {
		gasPrice.SetString(ctx.GlobalString(utils.GasPriceFlag.Name), 20)
	} else {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			utils.Fatalf("SuggestGasPrice error: %v", err)
		}
	}

	if gasPrice.Cmp(params.MinimalGasPrice) < 0 {
		utils.Fatalf("gas price: %v less than 100gwei", gasPrice.String())
	}

	tx := types.NewTransaction(
		nonce,
		*msg.To,
		msg.Value,
		gasLimit,
		gasPrice,
		msg.Data,
	)
	var keystoreDir string
	if ctx.GlobalIsSet(utils.KeyStoreDirFlag.Name) {
		keystoreDir = strings.TrimSpace(ctx.GlobalString(utils.KeyStoreDirFlag.Name))
	} else {
		keystoreDir = defaultKeystorDir()
	}

	var password string
	if ctx.GlobalIsSet(utils.PasswordFileFlag.Name) {
		passwords := utils.MakePasswordList(ctx)
		if passwords != nil {
			password = passwords[0]
		}
	} else {
		promptText := fmt.Sprintf("Please input address:%v password", fromAddress)
		password = utils.GetPassPhrase(promptText, false)
	}
	ks := keystore.NewKeyStore(keystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)
	account := accounts.Account{Address: fromAddress}

	signedTx, err := ks.SignTxWithPassphrase(account, password, tx, chainID)
	if err != nil {
		utils.Fatalf("SignTx error: %v\n", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		utils.Fatalf("Send Transaction error: %v\n", err)
	}
	fmt.Printf("Transaction send success, hash: %s\n", signedTx.Hash())
	return nil
}

func updateValidatorInfo(ctx *cli.Context, method string) error {
	validatorInfo := makeValidatorInfo(ctx)
	stakingAmount := big.NewInt(0)
	if method == ValidatorCreateMethod && ctx.GlobalIsSet(utils.ValidatorStakingFlag.Name) {
		stakingAmount.SetString(strings.TrimSpace(ctx.GlobalString(utils.ValidatorStakingFlag.Name)), 10)
	}

	validatorABIstr := senatus.ValidatorContractABI()
	valABI, err := abi.JSON(strings.NewReader(validatorABIstr))
	if err != nil {
		utils.Fatalf("validator abi load error: %v\n", err)
	}

	data, err := valABI.Pack(
		method,
		validatorInfo.rewardAddr,
		validatorInfo.moniker,
		validatorInfo.website,
		validatorInfo.email,
		validatorInfo.details,
	)
	if err != nil {
		utils.Fatalf("validator contract create pack error: %v\n", err)
	}
	validatorContractAddr := senatus.ValidatorContratAddress()
	msg := ethereum.CallMsg{
		To:    &validatorContractAddr,
		Value: stakingAmount,
		Data:  data,
	}
	return transactionHandler(ctx, &msg)
}

// createValidator create a validator
func createValidator(ctx *cli.Context) error {
	return updateValidatorInfo(ctx, ValidatorCreateMethod)
}

// editValidator edit a validator
func editValidator(ctx *cli.Context) error {
	return updateValidatorInfo(ctx, ValidatorEditMethod)
}

func stakeHander(ctx *cli.Context, method string) error {
	if !ctx.GlobalIsSet(utils.ValidatorAddressFlag.Name) {
		utils.Fatalf("validator address must set")
	}

	validatorAddress := common.HexToAddress(strings.TrimSpace(ctx.GlobalString(utils.ValidatorAddressFlag.Name)))

	stakingAmount := big.NewInt(0)
	if method == StakeMethod && ctx.GlobalIsSet(utils.ValidatorStakingFlag.Name) {
		stakingAmount.SetString(strings.TrimSpace(ctx.GlobalString(utils.ValidatorStakingFlag.Name)), 10)
	}

	validatorABIstr := senatus.ValidatorContractABI()
	valABI, err := abi.JSON(strings.NewReader(validatorABIstr))
	if err != nil {
		utils.Fatalf("validator abi load error: %v\n", err)
	}

	data, err := valABI.Pack(
		method,
		validatorAddress,
	)

	validatorContractAddr := senatus.ValidatorContratAddress()
	msg := ethereum.CallMsg{
		To:    &validatorContractAddr,
		Value: stakingAmount,
		Data:  data,
	}
	return transactionHandler(ctx, &msg)
}

// staking some token to a validator
func staking(ctx *cli.Context) error {
	return stakeHander(ctx, StakeMethod)
}

// unstaking from a validator
func unstaking(ctx *cli.Context) error {
	return stakeHander(ctx, UnstakeMethod)
}

func withdrawHandler(ctx *cli.Context, method string) error {
	if !ctx.GlobalIsSet(utils.ValidatorAddressFlag.Name) {
		utils.Fatalf("validator address must set")
	}
	validatorAddress := common.HexToAddress(strings.TrimSpace(ctx.GlobalString(utils.ValidatorAddressFlag.Name)))

	validatorABIstr := senatus.ValidatorContractABI()
	valABI, err := abi.JSON(strings.NewReader(validatorABIstr))
	if err != nil {
		utils.Fatalf("validator abi load error: %v\n", err)
	}

	data, err := valABI.Pack(
		method,
		validatorAddress,
	)

	value := big.NewInt(0)
	validatorContractAddr := senatus.ValidatorContratAddress()
	msg := ethereum.CallMsg{
		To:    &validatorContractAddr,
		Value: value,
		Data:  data,
	}
	return transactionHandler(ctx, &msg)
}

// withdrawReward from a validator
func withdrawReward(ctx *cli.Context) error {
	return withdrawHandler(ctx, WithdrawRewardsMethod)
}

// withdrawStaking from a validator
func withdrawStaking(ctx *cli.Context) error {
	return withdrawHandler(ctx, WithdrawStakingMethod)
}

// unjailed a validator
func unjail(ctx *cli.Context) error {
	validatorABIstr := senatus.ValidatorContractABI()
	valABI, err := abi.JSON(strings.NewReader(validatorABIstr))
	if err != nil {
		utils.Fatalf("validator abi load error: %v\n", err)
	}

	data, err := valABI.Pack(
		UnjailedMethod,
	)

	if err != nil {
		utils.Fatalf("validator pack error: %v\n", err)
	}

	value := big.NewInt(0)
	validatorContractAddr := senatus.ValidatorContratAddress()
	msg := ethereum.CallMsg{
		To:    &validatorContractAddr,
		Value: value,
		Data:  data,
	}
	return transactionHandler(ctx, &msg)
}

func queryHandler(ctx *cli.Context, msg *ethereum.CallMsg) ([]byte, error) {
	var nodeHost string
	if ctx.GlobalIsSet(utils.NodeEndpointFlag.Name) {
		nodeHost = strings.TrimSpace(ctx.GlobalString(utils.NodeEndpointFlag.Name))
	} else {
		nodeHost = defaultNodeHttpHost
	}

	clientCtx, cancel := context.WithTimeout(context.Background(), nodeTimeout)
	defer cancel()

	client, err := ethclient.Dial(nodeHost)
	if err != nil {
		utils.Fatalf("connect to %s error: %v\n", nodeHost, err)
	}
	return client.CallContract(clientCtx, *msg, nil)
}

func queryValidatorDescription(ctx *cli.Context) error {
	if !ctx.GlobalIsSet(utils.ValidatorAddressFlag.Name) {
		utils.Fatalf("validator address must be set")
	}
	validatorAddress := common.HexToAddress(strings.TrimSpace(ctx.GlobalString(utils.ValidatorAddressFlag.Name)))

	validatorABIstr := senatus.ValidatorContractABI()
	valABI, err := abi.JSON(strings.NewReader(validatorABIstr))
	if err != nil {
		utils.Fatalf("validator abi load error: %v\n", err)
	}

	validatorContractAddr := senatus.ValidatorContratAddress()
	data, err := valABI.Pack(
		GetValidatorDespMethod,
		validatorAddress,
	)
	if err != nil {
		utils.Fatalf("query validator description pack err: %v\n", err)
	}
	msg := ethereum.CallMsg{
		To:   &validatorContractAddr,
		Data: data,
	}
	result, err := queryHandler(ctx, &msg)
	if err != nil {
		utils.Fatalf("Get validator's description errr: %v\n", err)
	}
	var (
		moniker = new(string)
		website = new(string)
		email   = new(string)
		details = new(string)
	)
	out := &[]interface{}{
		moniker,
		website,
		email,
		details,
	}
	err = valABI.UnpackIntoInterface(out, GetValidatorDespMethod, result)
	if err != nil {
		utils.Fatalf("Get validator's description err: %v\n", err)
	}
	fmt.Printf("validator %v description: \n", validatorAddress)
	fmt.Printf("\tmoniker: %s\n", *moniker)
	fmt.Printf("\twebsite: %s\n", *website)
	fmt.Printf("\temail: %s\n", *email)
	fmt.Printf("\tdetails: %s\n", *details)
	return nil
}

func queryValidatorInfo(ctx *cli.Context) error {
	if !ctx.GlobalIsSet(utils.ValidatorAddressFlag.Name) {
		utils.Fatalf("validator address must be set")
	}
	validatorAddress := common.HexToAddress(strings.TrimSpace(ctx.GlobalString(utils.ValidatorAddressFlag.Name)))

	validatorABIstr := senatus.ValidatorContractABI()
	valABI, err := abi.JSON(strings.NewReader(validatorABIstr))
	if err != nil {
		utils.Fatalf("validator abi load error: %v\n", err)
	}

	validatorContractAddr := senatus.ValidatorContratAddress()
	data, err := valABI.Pack(
		GetValidatorInfoMethod,
		validatorAddress,
	)
	if err != nil {
		utils.Fatalf("query validator info pack err: %v\n", err)
	}
	msg := ethereum.CallMsg{
		To:   &validatorContractAddr,
		Data: data,
	}
	result, err := queryHandler(ctx, &msg)
	if err != nil {
		utils.Fatalf("Get validator's info errr: %v\n", err)
	}
	var (
		rewardAddr              = new(common.Address)
		status                  = new(uint8)
		stakingAmount           = new(*big.Int)
		rewardAmount            = new(*big.Int)
		slashAmount             = new(*big.Int)
		lastWithdrawRewardBlock = new(*big.Int)
		stakers                 = new([]common.Address)
	)
	out := &[]interface{}{
		rewardAddr,
		status,
		stakingAmount,
		rewardAmount,
		slashAmount,
		lastWithdrawRewardBlock,
		stakers,
	}
	err = valABI.UnpackIntoInterface(out, GetValidatorInfoMethod, result)
	if err != nil {
		utils.Fatalf("Unpack validator's info err: %v\n", err)
	}
	fmt.Printf("validator's info:\n")
	fmt.Printf("\tvalidator address: %v\n", validatorAddress)
	fmt.Printf("\trewardAddress: %v\n", rewardAddr)
	fmt.Printf("\tstatus: %v\n", validatorStatus((*status)))
	fmt.Printf("\tstakingAmount: %v\n", (*stakingAmount).String())
	fmt.Printf("\trewardAmount: %v\n", (*rewardAmount).String())
	fmt.Printf("\tslashAmount: %v\n", (*slashAmount).String())
	fmt.Printf("\tlastWithdrawRewardBlock: %v\n", (*lastWithdrawRewardBlock).String())
	fmt.Printf("\tstakers: %v\n", stakers)
	return nil
}

func queryActivatedValidators(ctx *cli.Context) error {
	validatorABIstr := senatus.ValidatorContractABI()
	valABI, err := abi.JSON(strings.NewReader(validatorABIstr))
	if err != nil {
		utils.Fatalf("validator abi load error: %v\n", err)
	}

	validatorContractAddr := senatus.ValidatorContratAddress()
	data, err := valABI.Pack(GetActivatedValidatorsMethod)
	if err != nil {
		utils.Fatalf("query activated validator pack err: %v\n", err)
	}
	msg := ethereum.CallMsg{
		To:   &validatorContractAddr,
		Data: data,
	}
	result, err := queryHandler(ctx, &msg)
	if err != nil {
		utils.Fatalf("Get activated validators errr: %v\n", err)
	}
	var validators []common.Address
	err = valABI.UnpackIntoInterface(&validators, GetActivatedValidatorsMethod, result)
	if err != nil {
		utils.Fatalf("Unpack activated validators err: %v\n", err)
	}
	fmt.Printf("current activated validators:\n")
	fmt.Printf("\tvalidators: %v\n", validators)
	return nil
}

func queryValidatorCandidators(ctx *cli.Context) error {
	validatorABIstr := senatus.ValidatorContractABI()
	valABI, err := abi.JSON(strings.NewReader(validatorABIstr))
	if err != nil {
		utils.Fatalf("validator abi load error: %v\n", err)
	}

	validatorContractAddr := senatus.ValidatorContratAddress()
	data, err := valABI.Pack(GetValidatorCandidatesMethod)
	if err != nil {
		utils.Fatalf("query validator candidators pack err: %v\n", err)
	}
	msg := ethereum.CallMsg{
		To:   &validatorContractAddr,
		Data: data,
	}
	result, err := queryHandler(ctx, &msg)
	if err != nil {
		utils.Fatalf("Get validator candidators errr: %v\n", err)
	}
	var (
		candidates     = new([]common.Address)
		stakingAmounts = new([]*big.Int)
		candidateSize  = new(*big.Int)
	)

	out := &[]interface{}{
		candidates,
		stakingAmounts,
		candidateSize,
	}
	err = valABI.UnpackIntoInterface(out, GetValidatorCandidatesMethod, result)
	if err != nil {
		utils.Fatalf("Unpack validator candidators err: %v\n", err)
	}
	candidatieSizeInt := int((*candidateSize).Int64())
	for i := 0; i < candidatieSizeInt; i++ {
		fmt.Printf("\tcandidator: %v, stakingAmount; %v\n", (*candidates)[i], (*stakingAmounts)[i])
	}

	return nil
}

func queryStakingInfo(ctx *cli.Context) error {
	if !ctx.GlobalIsSet(utils.ValidatorAddressFlag.Name) {
		utils.Fatalf("validator address must be set")
	}
	validatorAddress := common.HexToAddress(strings.TrimSpace(ctx.GlobalString(utils.ValidatorAddressFlag.Name)))

	if !ctx.GlobalIsSet(utils.StakerAddressFlag.Name) {
		utils.Fatalf("staker address must be set")
	}
	stakerAddress := common.HexToAddress(strings.TrimSpace(ctx.GlobalString(utils.StakerAddressFlag.Name)))

	validatorABIstr := senatus.ValidatorContractABI()
	valABI, err := abi.JSON(strings.NewReader(validatorABIstr))
	if err != nil {
		utils.Fatalf("validator abi load error: %v\n", err)
	}

	validatorContractAddr := senatus.ValidatorContratAddress()
	data, err := valABI.Pack(
		GetStakingInfoMethod,
		stakerAddress,
		validatorAddress,
	)
	if err != nil {
		utils.Fatalf("query staking info pack err: %v\n", err)
	}
	msg := ethereum.CallMsg{
		To:   &validatorContractAddr,
		Data: data,
	}
	result, err := queryHandler(ctx, &msg)
	if err != nil {
		utils.Fatalf("Get staking info errr: %v\n", err)
	}
	var (
		stakingAmount = new(*big.Int)
		unstakeBlock  = new(*big.Int)
		stakerIndex   = new(*big.Int)
	)

	out := &[]interface{}{
		stakingAmount,
		unstakeBlock,
		stakerIndex,
	}
	err = valABI.UnpackIntoInterface(out, GetStakingInfoMethod, result)
	if err != nil {
		utils.Fatalf("Unpack staking info err: %v\n", err)
	}
	fmt.Printf("staker: %v, staking to validator: %v\n", stakerAddress, validatorAddress)
	fmt.Printf("\tstaking amount: %s\n", (*stakingAmount).String())
	fmt.Printf("\tunstake block: %s\n", (*unstakeBlock).String())
	fmt.Printf("\tstaker index: %s\n", (*stakerIndex).String())
	return nil
}

func querySlashRecord(ctx *cli.Context) error {
	if !ctx.GlobalIsSet(utils.ValidatorAddressFlag.Name) {
		utils.Fatalf("validator address must be set")
	}
	validatorAddress := common.HexToAddress(strings.TrimSpace(ctx.GlobalString(utils.ValidatorAddressFlag.Name)))
	slashABIstr := senatus.SlashContractABI()
	slashABI, err := abi.JSON(strings.NewReader(slashABIstr))
	if err != nil {
		utils.Fatalf("validator abi load error: %v\n", err)
	}

	slashContractAddr := senatus.SlashContractAddress()
	data, err := slashABI.Pack(
		GetValidatorSlashRecordMethod,
		validatorAddress,
	)
	if err != nil {
		utils.Fatalf("query slash record info pack err: %v\n", err)
	}

	msg := ethereum.CallMsg{
		To:   &slashContractAddr,
		Data: data,
	}
	result, err := queryHandler(ctx, &msg)
	if err != nil {
		utils.Fatalf("Get slash record info errr: %v\n", err)
	}
	var missedBlocksCounter *big.Int

	err = slashABI.UnpackIntoInterface(&missedBlocksCounter, GetValidatorSlashRecordMethod, result)
	if err != nil {
		utils.Fatalf("Unpack staking info err: %v\n", err)
	}
	fmt.Printf("Validator: %v, missed block counter: %v\n", validatorAddress, missedBlocksCounter.String())
	return nil
}
