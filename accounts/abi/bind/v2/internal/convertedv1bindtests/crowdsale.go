// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package convertedv1bindtests

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

// TODO: convert this type to value type after everything works.
// CrowdsaleMetaData contains all meta data concerning the Crowdsale contract.
var CrowdsaleMetaData = &bind.MetaData{
	ABI:     "[{\"constant\":false,\"inputs\":[],\"name\":\"checkGoalReached\",\"outputs\":[],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"deadline\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"beneficiary\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"tokenReward\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"fundingGoal\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"amountRaised\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"price\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"funders\",\"outputs\":[{\"name\":\"addr\",\"type\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"inputs\":[{\"name\":\"ifSuccessfulSendTo\",\"type\":\"address\"},{\"name\":\"fundingGoalInEthers\",\"type\":\"uint256\"},{\"name\":\"durationInMinutes\",\"type\":\"uint256\"},{\"name\":\"etherCostOfEachToken\",\"type\":\"uint256\"},{\"name\":\"addressOfTokenUsedAsReward\",\"type\":\"address\"}],\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"backer\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"isContribution\",\"type\":\"bool\"}],\"name\":\"FundTransfer\",\"type\":\"event\"}]",
	Pattern: "84d7e935785c5c648282d326307bb8fa0d",
	Bin:     "0x606060408190526007805460ff1916905560a0806105a883396101006040529051608051915160c05160e05160008054600160a060020a03199081169095178155670de0b6b3a7640000958602600155603c9093024201600355930260045560058054909216909217905561052f90819061007990396000f36060604052361561006c5760e060020a600035046301cb3b20811461008257806329dcb0cf1461014457806338af3eed1461014d5780636e66f6e91461015f5780637a3a0e84146101715780637b3e5e7b1461017a578063a035b1fe14610183578063dc0d3dff1461018c575b61020060075460009060ff161561032357610002565b61020060035460009042106103205760025460015490106103cb576002548154600160a060020a0316908290606082818181858883f150915460025460408051600160a060020a039390931683526020830191909152818101869052517fe842aea7a5f1b01049d752008c53c52890b1a6daf660cf39e8eec506112bbdf6945090819003909201919050a15b60405160008054600160a060020a039081169230909116319082818181858883f150506007805460ff1916600117905550505050565b6103a160035481565b6103ab600054600160a060020a031681565b6103ab600554600160a060020a031681565b6103a160015481565b6103a160025481565b6103a160045481565b6103be60043560068054829081101561000257506000526002027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f8101547ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d409190910154600160a060020a03919091169082565b005b505050815481101561000257906000526020600020906002020160005060008201518160000160006101000a815481600160a060020a030219169083021790555060208201518160010160005055905050806002600082828250540192505081905550600560009054906101000a9004600160a060020a0316600160a060020a031663a9059cbb3360046000505484046040518360e060020a0281526004018083600160a060020a03168152602001828152602001925050506000604051808303816000876161da5a03f11561000257505060408051600160a060020a03331681526020810184905260018183015290517fe842aea7a5f1b01049d752008c53c52890b1a6daf660cf39e8eec506112bbdf692509081900360600190a15b50565b5060a0604052336060908152346080819052600680546001810180835592939282908280158290116102025760020281600202836000526020600020918201910161020291905b8082111561039d57805473ffffffffffffffffffffffffffffffffffffffff19168155600060019190910190815561036a565b5090565b6060908152602090f35b600160a060020a03166060908152602090f35b6060918252608052604090f35b5b60065481101561010e576006805482908110156100025760009182526002027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f0190600680549254600160a060020a0316928490811015610002576002027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d40015460405190915082818181858883f19350505050507fe842aea7a5f1b01049d752008c53c52890b1a6daf660cf39e8eec506112bbdf660066000508281548110156100025760008290526002027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f01548154600160a060020a039190911691908490811015610002576002027ff652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d40015460408051600160a060020a0394909416845260208401919091526000838201525191829003606001919050a16001016103cc56",
}

// Crowdsale is an auto generated Go binding around an Ethereum contract.
type Crowdsale struct {
	abi abi.ABI
}

// NewCrowdsale creates a new instance of Crowdsale.
func NewCrowdsale() (*Crowdsale, error) {
	parsed, err := CrowdsaleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Crowdsale{abi: *parsed}, nil
}

func (_Crowdsale *Crowdsale) PackConstructor(ifSuccessfulSendTo common.Address, fundingGoalInEthers *big.Int, durationInMinutes *big.Int, etherCostOfEachToken *big.Int, addressOfTokenUsedAsReward common.Address) []byte {
	res, _ := _Crowdsale.abi.Pack("", ifSuccessfulSendTo, fundingGoalInEthers, durationInMinutes, etherCostOfEachToken, addressOfTokenUsedAsReward)
	return res
}

// AmountRaised is a free data retrieval call binding the contract method 0x7b3e5e7b.
//
// Solidity: function amountRaised() returns(uint256)
func (_Crowdsale *Crowdsale) PackAmountRaised() ([]byte, error) {
	return _Crowdsale.abi.Pack("amountRaised")
}

func (_Crowdsale *Crowdsale) UnpackAmountRaised(data []byte) (*big.Int, error) {
	out, err := _Crowdsale.abi.Unpack("amountRaised", data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Beneficiary is a free data retrieval call binding the contract method 0x38af3eed.
//
// Solidity: function beneficiary() returns(address)
func (_Crowdsale *Crowdsale) PackBeneficiary() ([]byte, error) {
	return _Crowdsale.abi.Pack("beneficiary")
}

func (_Crowdsale *Crowdsale) UnpackBeneficiary(data []byte) (common.Address, error) {
	out, err := _Crowdsale.abi.Unpack("beneficiary", data)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CheckGoalReached is a free data retrieval call binding the contract method 0x01cb3b20.
//
// Solidity: function checkGoalReached() returns()
func (_Crowdsale *Crowdsale) PackCheckGoalReached() ([]byte, error) {
	return _Crowdsale.abi.Pack("checkGoalReached")
}

// Deadline is a free data retrieval call binding the contract method 0x29dcb0cf.
//
// Solidity: function deadline() returns(uint256)
func (_Crowdsale *Crowdsale) PackDeadline() ([]byte, error) {
	return _Crowdsale.abi.Pack("deadline")
}

func (_Crowdsale *Crowdsale) UnpackDeadline(data []byte) (*big.Int, error) {
	out, err := _Crowdsale.abi.Unpack("deadline", data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Funders is a free data retrieval call binding the contract method 0xdc0d3dff.
//
// Solidity: function funders(uint256 ) returns(address addr, uint256 amount)
func (_Crowdsale *Crowdsale) PackFunders(Arg0 *big.Int) ([]byte, error) {
	return _Crowdsale.abi.Pack("funders", Arg0)
}

type FundersOutput struct {
	Addr   common.Address
	Amount *big.Int
}

func (_Crowdsale *Crowdsale) UnpackFunders(data []byte) (FundersOutput, error) {
	out, err := _Crowdsale.abi.Unpack("funders", data)

	outstruct := new(FundersOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Addr = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Amount = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// FundingGoal is a free data retrieval call binding the contract method 0x7a3a0e84.
//
// Solidity: function fundingGoal() returns(uint256)
func (_Crowdsale *Crowdsale) PackFundingGoal() ([]byte, error) {
	return _Crowdsale.abi.Pack("fundingGoal")
}

func (_Crowdsale *Crowdsale) UnpackFundingGoal(data []byte) (*big.Int, error) {
	out, err := _Crowdsale.abi.Unpack("fundingGoal", data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Price is a free data retrieval call binding the contract method 0xa035b1fe.
//
// Solidity: function price() returns(uint256)
func (_Crowdsale *Crowdsale) PackPrice() ([]byte, error) {
	return _Crowdsale.abi.Pack("price")
}

func (_Crowdsale *Crowdsale) UnpackPrice(data []byte) (*big.Int, error) {
	out, err := _Crowdsale.abi.Unpack("price", data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokenReward is a free data retrieval call binding the contract method 0x6e66f6e9.
//
// Solidity: function tokenReward() returns(address)
func (_Crowdsale *Crowdsale) PackTokenReward() ([]byte, error) {
	return _Crowdsale.abi.Pack("tokenReward")
}

func (_Crowdsale *Crowdsale) UnpackTokenReward(data []byte) (common.Address, error) {
	out, err := _Crowdsale.abi.Unpack("tokenReward", data)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CrowdsaleFundTransfer represents a FundTransfer event raised by the Crowdsale contract.
type CrowdsaleFundTransfer struct {
	Backer         common.Address
	Amount         *big.Int
	IsContribution bool
	Raw            *types.Log // Blockchain specific contextual infos
}

const CrowdsaleFundTransferEventName = "FundTransfer"

func (_Crowdsale *Crowdsale) UnpackFundTransferEvent(log *types.Log) (*CrowdsaleFundTransfer, error) {
	event := "FundTransfer"
	if log.Topics[0] != _Crowdsale.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(CrowdsaleFundTransfer)
	if len(log.Data) > 0 {
		if err := _Crowdsale.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range _Crowdsale.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}
