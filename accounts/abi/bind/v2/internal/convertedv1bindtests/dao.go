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
// DAOMetaData contains all meta data concerning the DAO contract.
var DAOMetaData = &bind.MetaData{
	ABI:     "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"proposals\",\"outputs\":[{\"name\":\"recipient\",\"type\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\"},{\"name\":\"description\",\"type\":\"string\"},{\"name\":\"votingDeadline\",\"type\":\"uint256\"},{\"name\":\"executed\",\"type\":\"bool\"},{\"name\":\"proposalPassed\",\"type\":\"bool\"},{\"name\":\"numberOfVotes\",\"type\":\"uint256\"},{\"name\":\"currentResult\",\"type\":\"int256\"},{\"name\":\"proposalHash\",\"type\":\"bytes32\"}],\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"proposalNumber\",\"type\":\"uint256\"},{\"name\":\"transactionBytecode\",\"type\":\"bytes\"}],\"name\":\"executeProposal\",\"outputs\":[{\"name\":\"result\",\"type\":\"int256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"memberId\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"numProposals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"members\",\"outputs\":[{\"name\":\"member\",\"type\":\"address\"},{\"name\":\"canVote\",\"type\":\"bool\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"memberSince\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"debatingPeriodInMinutes\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"minimumQuorum\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"targetMember\",\"type\":\"address\"},{\"name\":\"canVote\",\"type\":\"bool\"},{\"name\":\"memberName\",\"type\":\"string\"}],\"name\":\"changeMembership\",\"outputs\":[],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"majorityMargin\",\"outputs\":[{\"name\":\"\",\"type\":\"int256\"}],\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"beneficiary\",\"type\":\"address\"},{\"name\":\"etherAmount\",\"type\":\"uint256\"},{\"name\":\"JobDescription\",\"type\":\"string\"},{\"name\":\"transactionBytecode\",\"type\":\"bytes\"}],\"name\":\"newProposal\",\"outputs\":[{\"name\":\"proposalID\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"minimumQuorumForProposals\",\"type\":\"uint256\"},{\"name\":\"minutesForDebate\",\"type\":\"uint256\"},{\"name\":\"marginOfVotesForMajority\",\"type\":\"int256\"}],\"name\":\"changeVotingRules\",\"outputs\":[],\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"proposalNumber\",\"type\":\"uint256\"},{\"name\":\"supportsProposal\",\"type\":\"bool\"},{\"name\":\"justificationText\",\"type\":\"string\"}],\"name\":\"vote\",\"outputs\":[{\"name\":\"voteID\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"proposalNumber\",\"type\":\"uint256\"},{\"name\":\"beneficiary\",\"type\":\"address\"},{\"name\":\"etherAmount\",\"type\":\"uint256\"},{\"name\":\"transactionBytecode\",\"type\":\"bytes\"}],\"name\":\"checkProposalCode\",\"outputs\":[{\"name\":\"codeChecksOut\",\"type\":\"bool\"}],\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"type\":\"function\"},{\"inputs\":[{\"name\":\"minimumQuorumForProposals\",\"type\":\"uint256\"},{\"name\":\"minutesForDebate\",\"type\":\"uint256\"},{\"name\":\"marginOfVotesForMajority\",\"type\":\"int256\"},{\"name\":\"congressLeader\",\"type\":\"address\"}],\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"proposalID\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"description\",\"type\":\"string\"}],\"name\":\"ProposalAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"proposalID\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"position\",\"type\":\"bool\"},{\"indexed\":false,\"name\":\"voter\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"justification\",\"type\":\"string\"}],\"name\":\"Voted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"proposalID\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"result\",\"type\":\"int256\"},{\"indexed\":false,\"name\":\"quorum\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"active\",\"type\":\"bool\"}],\"name\":\"ProposalTallied\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"member\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"isMember\",\"type\":\"bool\"}],\"name\":\"MembershipChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"minimumQuorum\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"debatingPeriodInMinutes\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"majorityMargin\",\"type\":\"int256\"}],\"name\":\"ChangeOfRules\",\"type\":\"event\"}]",
	Pattern: "d0a4ad96d49edb1c33461cebc6fb260919",
	Bin:     "0x606060405260405160808061145f833960e06040529051905160a05160c05160008054600160a060020a03191633179055600184815560028490556003839055600780549182018082558280158290116100b8576003028160030283600052602060002091820191016100b891906101c8565b50506060919091015160029190910155600160a060020a0381166000146100a65760008054600160a060020a031916821790555b505050506111f18061026e6000396000f35b505060408051608081018252600080825260208281018290528351908101845281815292820192909252426060820152600780549194509250811015610002579081527fa66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c6889050815181546020848101517401000000000000000000000000000000000000000002600160a060020a03199290921690921760a060020a60ff021916178255604083015180516001848101805460008281528690209195600293821615610100026000190190911692909204601f9081018390048201949192919091019083901061023e57805160ff19168380011785555b50610072929150610226565b5050600060028201556001015b8082111561023a578054600160a860020a031916815560018181018054600080835592600290821615610100026000190190911604601f81901061020c57506101bb565b601f0160209004906000526020600020908101906101bb91905b8082111561023a5760008155600101610226565b5090565b828001600101855582156101af579182015b828111156101af57825182600050559160200191906001019061025056606060405236156100b95760e060020a6000350463013cf08b81146100bb578063237e9492146101285780633910682114610281578063400e3949146102995780635daf08ca146102a257806369bd34361461032f5780638160f0b5146103385780638da5cb5b146103415780639644fcbd14610353578063aa02a90f146103be578063b1050da5146103c7578063bcca1fd3146104b5578063d3c0715b146104dc578063eceb29451461058d578063f2fde38b1461067b575b005b61069c6004356004805482908110156100025790600052602060002090600a02016000506005810154815460018301546003840154600485015460068601546007870154600160a060020a03959095169750929560020194919360ff828116946101009093041692919089565b60408051602060248035600481810135601f81018590048502860185019096528585526107759581359591946044949293909201918190840183828082843750949650505050505050600060006004600050848154811015610002575090527f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19e600a8402908101547f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b909101904210806101e65750600481015460ff165b8061026757508060000160009054906101000a9004600160a060020a03168160010160005054846040518084600160a060020a0316606060020a0281526014018381526020018280519060200190808383829060006004602084601f0104600f02600301f15090500193505050506040518091039020816007016000505414155b8061027757506001546005820154105b1561109257610002565b61077560043560066020526000908152604090205481565b61077560055481565b61078760043560078054829081101561000257506000526003026000805160206111d18339815191528101547fa66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c68a820154600160a060020a0382169260a060020a90920460ff16917fa66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c689019084565b61077560025481565b61077560015481565b610830600054600160a060020a031681565b604080516020604435600481810135601f81018490048402850184019095528484526100b9948135946024803595939460649492939101918190840183828082843750949650505050505050600080548190600160a060020a03908116339091161461084d57610002565b61077560035481565b604080516020604435600481810135601f8101849004840285018401909552848452610775948135946024803595939460649492939101918190840183828082843750506040805160209735808a0135601f81018a90048a0283018a019093528282529698976084979196506024909101945090925082915084018382808284375094965050505050505033600160a060020a031660009081526006602052604081205481908114806104ab5750604081205460078054909190811015610002579082526003026000805160206111d1833981519152015460a060020a900460ff16155b15610ce557610002565b6100b960043560243560443560005433600160a060020a03908116911614610b1857610002565b604080516020604435600481810135601f810184900484028501840190955284845261077594813594602480359593946064949293910191819084018382808284375094965050505050505033600160a060020a031660009081526006602052604081205481908114806105835750604081205460078054909190811015610002579082526003026000805160206111d18339815191520181505460a060020a900460ff16155b15610f1d57610002565b604080516020606435600481810135601f81018490048402850184019095528484526107759481359460248035956044359560849492019190819084018382808284375094965050505050505060006000600460005086815481101561000257908252600a027f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b01815090508484846040518084600160a060020a0316606060020a0281526014018381526020018280519060200190808383829060006004602084601f0104600f02600301f150905001935050505060405180910390208160070160005054149150610cdc565b6100b960043560005433600160a060020a03908116911614610f0857610002565b604051808a600160a060020a031681526020018981526020018060200188815260200187815260200186815260200185815260200184815260200183815260200182810382528981815460018160011615610100020316600290048152602001915080546001816001161561010002031660029004801561075e5780601f106107335761010080835404028352916020019161075e565b820191906000526020600020905b81548152906001019060200180831161074157829003601f168201915b50509a505050505050505050505060405180910390f35b60408051918252519081900360200190f35b60408051600160a060020a038616815260208101859052606081018390526080918101828152845460026001821615610100026000190190911604928201839052909160a08301908590801561081e5780601f106107f35761010080835404028352916020019161081e565b820191906000526020600020905b81548152906001019060200180831161080157829003601f168201915b50509550505050505060405180910390f35b60408051600160a060020a03929092168252519081900360200190f35b600160a060020a03851660009081526006602052604081205414156108a957604060002060078054918290556001820180825582801582901161095c5760030281600302836000526020600020918201910161095c9190610a4f565b600160a060020a03851660009081526006602052604090205460078054919350908390811015610002575060005250600381026000805160206111d183398151915201805474ff0000000000000000000000000000000000000000191660a060020a85021781555b60408051600160a060020a03871681526020810186905281517f27b022af4a8347100c7a041ce5ccf8e14d644ff05de696315196faae8cd50c9b929181900390910190a15050505050565b505050915081506080604051908101604052808681526020018581526020018481526020014281526020015060076000508381548110156100025790600052602060002090600302016000508151815460208481015160a060020a02600160a060020a03199290921690921774ff00000000000000000000000000000000000000001916178255604083015180516001848101805460008281528690209195600293821615610100026000190190911692909204601f90810183900482019491929190910190839010610ad357805160ff19168380011785555b50610b03929150610abb565b5050600060028201556001015b80821115610acf57805474ffffffffffffffffffffffffffffffffffffffffff1916815560018181018054600080835592600290821615610100026000190190911604601f819010610aa15750610a42565b601f016020900490600052602060002090810190610a4291905b80821115610acf5760008155600101610abb565b5090565b82800160010185558215610a36579182015b82811115610a36578251826000505591602001919060010190610ae5565b50506060919091015160029190910155610911565b600183905560028290556003819055604080518481526020810184905280820183905290517fa439d3fa452be5e0e1e24a8145e715f4fd8b9c08c96a42fd82a855a85e5d57de9181900360600190a1505050565b50508585846040518084600160a060020a0316606060020a0281526014018381526020018280519060200190808383829060006004602084601f0104600f02600301f150905001935050505060405180910390208160070160005081905550600260005054603c024201816003016000508190555060008160040160006101000a81548160ff0219169083021790555060008160040160016101000a81548160ff02191690830217905550600081600501600050819055507f646fec02522b41e7125cfc859a64fd4f4cefd5dc3b6237ca0abe251ded1fa881828787876040518085815260200184600160a060020a03168152602001838152602001806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f168015610cc45780820380516001836020036101000a031916815260200191505b509550505050505060405180910390a1600182016005555b50949350505050565b6004805460018101808355909190828015829011610d1c57600a0281600a028360005260206000209182019101610d1c9190610db8565b505060048054929450918491508110156100025790600052602060002090600a02016000508054600160a060020a031916871781556001818101879055855160028381018054600082815260209081902096975091959481161561010002600019011691909104601f90810182900484019391890190839010610ed857805160ff19168380011785555b50610b6c929150610abb565b50506001015b80821115610acf578054600160a060020a03191681556000600182810182905560028381018054848255909281161561010002600019011604601f819010610e9c57505b5060006003830181905560048301805461ffff191690556005830181905560068301819055600783018190556008830180548282559082526020909120610db2916002028101905b80821115610acf57805474ffffffffffffffffffffffffffffffffffffffffff1916815560018181018054600080835592600290821615610100026000190190911604601f819010610eba57505b5050600101610e44565b601f016020900490600052602060002090810190610dfc9190610abb565b601f016020900490600052602060002090810190610e929190610abb565b82800160010185558215610da6579182015b82811115610da6578251826000505591602001919060010190610eea565b60008054600160a060020a0319168217905550565b600480548690811015610002576000918252600a027f8a35acfbc15ff81a39ae7d344fd709f28e8600b4aa8c65c6b64bfe7fe36bd19b01905033600160a060020a0316600090815260098201602052604090205490915060ff1660011415610f8457610002565b33600160a060020a031660009081526009820160205260409020805460ff1916600190811790915560058201805490910190558315610fcd576006810180546001019055610fda565b6006810180546000190190555b7fc34f869b7ff431b034b7b9aea9822dac189a685e0b015c7d1be3add3f89128e8858533866040518085815260200184815260200183600160a060020a03168152602001806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f16801561107a5780820380516001836020036101000a031916815260200191505b509550505050505060405180910390a1509392505050565b6006810154600354901315611158578060000160009054906101000a9004600160a060020a0316600160a060020a03168160010160005054670de0b6b3a76400000284604051808280519060200190808383829060006004602084601f0104600f02600301f150905090810190601f1680156111225780820380516001836020036101000a031916815260200191505b5091505060006040518083038185876185025a03f15050505060048101805460ff191660011761ff00191661010017905561116d565b60048101805460ff191660011761ff00191690555b60068101546005820154600483015460408051888152602081019490945283810192909252610100900460ff166060830152517fd220b7272a8b6d0d7d6bcdace67b936a8f175e6d5c1b3ee438b72256b32ab3af9181900360800190a1509291505056a66cc928b5edb82af9bd49922954155ab7b0942694bea4ce44661d9a8736c688",
}

// DAO is an auto generated Go binding around an Ethereum contract.
type DAO struct {
	abi abi.ABI
}

// NewDAO creates a new instance of DAO.
func NewDAO() (*DAO, error) {
	parsed, err := DAOMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &DAO{abi: *parsed}, nil
}

func (dAO *DAO) PackConstructor(minimumQuorumForProposals *big.Int, minutesForDebate *big.Int, marginOfVotesForMajority *big.Int, congressLeader common.Address) []byte {
	res, _ := dAO.abi.Pack("", minimumQuorumForProposals, minutesForDebate, marginOfVotesForMajority, congressLeader)
	return res
}

// ChangeMembership is a free data retrieval call binding the contract method 0x9644fcbd.
//
// Solidity: function changeMembership(address targetMember, bool canVote, string memberName) returns()
func (dAO *DAO) PackChangeMembership(TargetMember common.Address, CanVote bool, MemberName string) ([]byte, error) {
	return dAO.abi.Pack("changeMembership", TargetMember, CanVote, MemberName)
}

// ChangeVotingRules is a free data retrieval call binding the contract method 0xbcca1fd3.
//
// Solidity: function changeVotingRules(uint256 minimumQuorumForProposals, uint256 minutesForDebate, int256 marginOfVotesForMajority) returns()
func (dAO *DAO) PackChangeVotingRules(MinimumQuorumForProposals *big.Int, MinutesForDebate *big.Int, MarginOfVotesForMajority *big.Int) ([]byte, error) {
	return dAO.abi.Pack("changeVotingRules", MinimumQuorumForProposals, MinutesForDebate, MarginOfVotesForMajority)
}

// CheckProposalCode is a free data retrieval call binding the contract method 0xeceb2945.
//
// Solidity: function checkProposalCode(uint256 proposalNumber, address beneficiary, uint256 etherAmount, bytes transactionBytecode) returns(bool codeChecksOut)
func (dAO *DAO) PackCheckProposalCode(ProposalNumber *big.Int, Beneficiary common.Address, EtherAmount *big.Int, TransactionBytecode []byte) ([]byte, error) {
	return dAO.abi.Pack("checkProposalCode", ProposalNumber, Beneficiary, EtherAmount, TransactionBytecode)
}

func (dAO *DAO) UnpackCheckProposalCode(data []byte) (bool, error) {
	out, err := dAO.abi.Unpack("checkProposalCode", data)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// DebatingPeriodInMinutes is a free data retrieval call binding the contract method 0x69bd3436.
//
// Solidity: function debatingPeriodInMinutes() returns(uint256)
func (dAO *DAO) PackDebatingPeriodInMinutes() ([]byte, error) {
	return dAO.abi.Pack("debatingPeriodInMinutes")
}

func (dAO *DAO) UnpackDebatingPeriodInMinutes(data []byte) (*big.Int, error) {
	out, err := dAO.abi.Unpack("debatingPeriodInMinutes", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// ExecuteProposal is a free data retrieval call binding the contract method 0x237e9492.
//
// Solidity: function executeProposal(uint256 proposalNumber, bytes transactionBytecode) returns(int256 result)
func (dAO *DAO) PackExecuteProposal(ProposalNumber *big.Int, TransactionBytecode []byte) ([]byte, error) {
	return dAO.abi.Pack("executeProposal", ProposalNumber, TransactionBytecode)
}

func (dAO *DAO) UnpackExecuteProposal(data []byte) (*big.Int, error) {
	out, err := dAO.abi.Unpack("executeProposal", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// MajorityMargin is a free data retrieval call binding the contract method 0xaa02a90f.
//
// Solidity: function majorityMargin() returns(int256)
func (dAO *DAO) PackMajorityMargin() ([]byte, error) {
	return dAO.abi.Pack("majorityMargin")
}

func (dAO *DAO) UnpackMajorityMargin(data []byte) (*big.Int, error) {
	out, err := dAO.abi.Unpack("majorityMargin", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// MemberId is a free data retrieval call binding the contract method 0x39106821.
//
// Solidity: function memberId(address ) returns(uint256)
func (dAO *DAO) PackMemberId(Arg0 common.Address) ([]byte, error) {
	return dAO.abi.Pack("memberId", Arg0)
}

func (dAO *DAO) UnpackMemberId(data []byte) (*big.Int, error) {
	out, err := dAO.abi.Unpack("memberId", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// Members is a free data retrieval call binding the contract method 0x5daf08ca.
//
// Solidity: function members(uint256 ) returns(address member, bool canVote, string name, uint256 memberSince)
func (dAO *DAO) PackMembers(Arg0 *big.Int) ([]byte, error) {
	return dAO.abi.Pack("members", Arg0)
}

type MembersOutput struct {
	Member      common.Address
	CanVote     bool
	Name        string
	MemberSince *big.Int
}

func (dAO *DAO) UnpackMembers(data []byte) (MembersOutput, error) {
	out, err := dAO.abi.Unpack("members", data)

	outstruct := new(MembersOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Member = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	outstruct.CanVote = *abi.ConvertType(out[1], new(bool)).(*bool)

	outstruct.Name = *abi.ConvertType(out[2], new(string)).(*string)

	outstruct.MemberSince = abi.ConvertType(out[3], new(big.Int)).(*big.Int)

	return *outstruct, err

}

// MinimumQuorum is a free data retrieval call binding the contract method 0x8160f0b5.
//
// Solidity: function minimumQuorum() returns(uint256)
func (dAO *DAO) PackMinimumQuorum() ([]byte, error) {
	return dAO.abi.Pack("minimumQuorum")
}

func (dAO *DAO) UnpackMinimumQuorum(data []byte) (*big.Int, error) {
	out, err := dAO.abi.Unpack("minimumQuorum", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// NewProposal is a free data retrieval call binding the contract method 0xb1050da5.
//
// Solidity: function newProposal(address beneficiary, uint256 etherAmount, string JobDescription, bytes transactionBytecode) returns(uint256 proposalID)
func (dAO *DAO) PackNewProposal(Beneficiary common.Address, EtherAmount *big.Int, JobDescription string, TransactionBytecode []byte) ([]byte, error) {
	return dAO.abi.Pack("newProposal", Beneficiary, EtherAmount, JobDescription, TransactionBytecode)
}

func (dAO *DAO) UnpackNewProposal(data []byte) (*big.Int, error) {
	out, err := dAO.abi.Unpack("newProposal", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// NumProposals is a free data retrieval call binding the contract method 0x400e3949.
//
// Solidity: function numProposals() returns(uint256)
func (dAO *DAO) PackNumProposals() ([]byte, error) {
	return dAO.abi.Pack("numProposals")
}

func (dAO *DAO) UnpackNumProposals(data []byte) (*big.Int, error) {
	out, err := dAO.abi.Unpack("numProposals", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() returns(address)
func (dAO *DAO) PackOwner() ([]byte, error) {
	return dAO.abi.Pack("owner")
}

func (dAO *DAO) UnpackOwner(data []byte) (common.Address, error) {
	out, err := dAO.abi.Unpack("owner", data)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Proposals is a free data retrieval call binding the contract method 0x013cf08b.
//
// Solidity: function proposals(uint256 ) returns(address recipient, uint256 amount, string description, uint256 votingDeadline, bool executed, bool proposalPassed, uint256 numberOfVotes, int256 currentResult, bytes32 proposalHash)
func (dAO *DAO) PackProposals(Arg0 *big.Int) ([]byte, error) {
	return dAO.abi.Pack("proposals", Arg0)
}

type ProposalsOutput struct {
	Recipient      common.Address
	Amount         *big.Int
	Description    string
	VotingDeadline *big.Int
	Executed       bool
	ProposalPassed bool
	NumberOfVotes  *big.Int
	CurrentResult  *big.Int
	ProposalHash   [32]byte
}

func (dAO *DAO) UnpackProposals(data []byte) (ProposalsOutput, error) {
	out, err := dAO.abi.Unpack("proposals", data)

	outstruct := new(ProposalsOutput)
	if err != nil {
		return *outstruct, err
	}

	outstruct.Recipient = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	outstruct.Amount = abi.ConvertType(out[1], new(big.Int)).(*big.Int)

	outstruct.Description = *abi.ConvertType(out[2], new(string)).(*string)

	outstruct.VotingDeadline = abi.ConvertType(out[3], new(big.Int)).(*big.Int)

	outstruct.Executed = *abi.ConvertType(out[4], new(bool)).(*bool)

	outstruct.ProposalPassed = *abi.ConvertType(out[5], new(bool)).(*bool)

	outstruct.NumberOfVotes = abi.ConvertType(out[6], new(big.Int)).(*big.Int)

	outstruct.CurrentResult = abi.ConvertType(out[7], new(big.Int)).(*big.Int)

	outstruct.ProposalHash = *abi.ConvertType(out[8], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// TransferOwnership is a free data retrieval call binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (dAO *DAO) PackTransferOwnership(NewOwner common.Address) ([]byte, error) {
	return dAO.abi.Pack("transferOwnership", NewOwner)
}

// Vote is a free data retrieval call binding the contract method 0xd3c0715b.
//
// Solidity: function vote(uint256 proposalNumber, bool supportsProposal, string justificationText) returns(uint256 voteID)
func (dAO *DAO) PackVote(ProposalNumber *big.Int, SupportsProposal bool, JustificationText string) ([]byte, error) {
	return dAO.abi.Pack("vote", ProposalNumber, SupportsProposal, JustificationText)
}

func (dAO *DAO) UnpackVote(data []byte) (*big.Int, error) {
	out, err := dAO.abi.Unpack("vote", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// DAOChangeOfRules represents a ChangeOfRules event raised by the DAO contract.
type DAOChangeOfRules struct {
	MinimumQuorum           *big.Int
	DebatingPeriodInMinutes *big.Int
	MajorityMargin          *big.Int
	Raw                     *types.Log // Blockchain specific contextual infos
}

const DAOChangeOfRulesEventName = "ChangeOfRules"

func (dAO *DAO) UnpackChangeOfRulesEvent(log *types.Log) (*DAOChangeOfRules, error) {
	event := "ChangeOfRules"
	if log.Topics[0] != dAO.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(DAOChangeOfRules)
	if len(log.Data) > 0 {
		if err := dAO.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range dAO.abi.Events[event].Inputs {
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

// DAOMembershipChanged represents a MembershipChanged event raised by the DAO contract.
type DAOMembershipChanged struct {
	Member   common.Address
	IsMember bool
	Raw      *types.Log // Blockchain specific contextual infos
}

const DAOMembershipChangedEventName = "MembershipChanged"

func (dAO *DAO) UnpackMembershipChangedEvent(log *types.Log) (*DAOMembershipChanged, error) {
	event := "MembershipChanged"
	if log.Topics[0] != dAO.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(DAOMembershipChanged)
	if len(log.Data) > 0 {
		if err := dAO.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range dAO.abi.Events[event].Inputs {
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

// DAOProposalAdded represents a ProposalAdded event raised by the DAO contract.
type DAOProposalAdded struct {
	ProposalID  *big.Int
	Recipient   common.Address
	Amount      *big.Int
	Description string
	Raw         *types.Log // Blockchain specific contextual infos
}

const DAOProposalAddedEventName = "ProposalAdded"

func (dAO *DAO) UnpackProposalAddedEvent(log *types.Log) (*DAOProposalAdded, error) {
	event := "ProposalAdded"
	if log.Topics[0] != dAO.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(DAOProposalAdded)
	if len(log.Data) > 0 {
		if err := dAO.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range dAO.abi.Events[event].Inputs {
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

// DAOProposalTallied represents a ProposalTallied event raised by the DAO contract.
type DAOProposalTallied struct {
	ProposalID *big.Int
	Result     *big.Int
	Quorum     *big.Int
	Active     bool
	Raw        *types.Log // Blockchain specific contextual infos
}

const DAOProposalTalliedEventName = "ProposalTallied"

func (dAO *DAO) UnpackProposalTalliedEvent(log *types.Log) (*DAOProposalTallied, error) {
	event := "ProposalTallied"
	if log.Topics[0] != dAO.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(DAOProposalTallied)
	if len(log.Data) > 0 {
		if err := dAO.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range dAO.abi.Events[event].Inputs {
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

// DAOVoted represents a Voted event raised by the DAO contract.
type DAOVoted struct {
	ProposalID    *big.Int
	Position      bool
	Voter         common.Address
	Justification string
	Raw           *types.Log // Blockchain specific contextual infos
}

const DAOVotedEventName = "Voted"

func (dAO *DAO) UnpackVotedEvent(log *types.Log) (*DAOVoted, error) {
	event := "Voted"
	if log.Topics[0] != dAO.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(DAOVoted)
	if len(log.Data) > 0 {
		if err := dAO.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range dAO.abi.Events[event].Inputs {
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