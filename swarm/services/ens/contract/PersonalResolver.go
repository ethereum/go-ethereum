// This file is an automatically generated Go binding. Do not modify as any
// change will likely be lost upon the next re-generation!

package contract

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// PersonalResolverABI is the input ABI used to generate the binding from.
const PersonalResolverABI = `[{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"setOwner","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"bytes32[]"}],"name":"deletePrivateRR","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"isPersonalResolver","outputs":[{"name":"","type":"bool"}],"type":"function"},{"constant":true,"inputs":[{"name":"id","type":"bytes32"}],"name":"getExtended","outputs":[{"name":"data","type":"bytes"}],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"string"},{"name":"rtype","type":"bytes16"},{"name":"ttl","type":"uint32"},{"name":"len","type":"uint16"},{"name":"data","type":"bytes32"}],"name":"setRR","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"bytes32[]"},{"name":"rtype","type":"bytes16"},{"name":"ttl","type":"uint32"},{"name":"len","type":"uint16"},{"name":"data","type":"bytes32"}],"name":"setPrivateRR","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"nodeId","type":"bytes12"},{"name":"qtype","type":"bytes32"},{"name":"index","type":"uint16"}],"name":"resolve","outputs":[{"name":"rcode","type":"uint16"},{"name":"rtype","type":"bytes16"},{"name":"ttl","type":"uint32"},{"name":"len","type":"uint16"},{"name":"data","type":"bytes32"}],"type":"function"},{"constant":false,"inputs":[{"name":"rootNodeId","type":"bytes12"},{"name":"name","type":"string"}],"name":"deleteRR","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"nodeId","type":"bytes12"},{"name":"label","type":"bytes32"}],"name":"findResolver","outputs":[{"name":"rcode","type":"uint16"},{"name":"ttl","type":"uint32"},{"name":"rnode","type":"bytes12"},{"name":"raddress","type":"address"}],"type":"function"},{"inputs":[],"type":"constructor"}]`

// PersonalResolverBin is the compiled bytecode used for deploying new contracts.
const PersonalResolverBin = `60606040525b33600060006101000a81548173ffffffffffffffffffffffffffffffffffffffff02191690830217905550600160016000506000600074010000000000000000000000000000000000000000028152602001908152602001600020600050600201600050819055505b611cb58061007c6000396000f3606060405236156100a0576000357c01000000000000000000000000000000000000000000000000000000009004806313af4035146100a25780631b370194146100ba5780633f5665e7146101165780638021061c1461013b5780638bba944d146101bf5780638da5cb5b1461024257806391c8e7b91461027b578063a16fdafa146102fb578063bc06183d1461037a578063edc0277c146103d9576100a0565b005b6100b8600480803590602001909190505061045a565b005b610114600480803590602001909190803590602001908201803590602001919190808060200260200160405190810160405280939291908181526020018383602002808284378201915050505050509090919050506104e5565b005b61012360048050506105c1565b60405180821515815260200191505060405180910390f35b61015160048080359060200190919050506105cf565b60405180806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f1680156101b15780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b6102406004808035906020019091908035906020019082018035906020019191908080601f016020809104026020016040519081016040528093929190818152602001838380828437820191505050505050909091908035906020019091908035906020019091908035906020019091908035906020019091905050610603565b005b61024f6004805050610699565b604051808273ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6102f9600480803590602001909190803590602001908201803590602001919190808060200260200160405190810160405280939291908181526020018383602002808284378201915050505050509090919080359060200190919080359060200190919080359060200190919080359060200190919050506106bf565b005b610323600480803590602001909190803590602001909190803590602001909190505061074f565b604051808661ffff168152602001856fffffffffffffffffffffffffffffffff191681526020018463ffffffff1681526020018361ffff168152602001826000191681526020019550505050505060405180910390f35b6103d76004808035906020019091908035906020019082018035906020019191908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509090919050506108b0565b005b6103f86004808035906020019091908035906020019091905050610998565b604051808561ffff1681526020018463ffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff191681526020018273ffffffffffffffffffffffffffffffffffffffff16815260200194505050505060405180910390f35b600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156104b657610002565b80600060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908302179055505b50565b60006000600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561054557610002565b610553848460008651610a14565b9150915060008160020160005054141561056c57610002565b6001816002016000505411156105ab576105a681600001600050600070010000000000000000000000000000000002600060006000610b41565b6105ba565b6105b9848460008651610bde565b5b5b50505050565b6000600190506105cc565b90565b6020604051908101604052806000815260200150602060405190810160405280600081526020015090506105fe565b919050565b60006000600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561066357610002565b6106778861067289610dfd9090565b610e47565b9150915061068e8160000160005087878787610b41565b5b5050505050505050565b600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60006000600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561071f57610002565b61072d888860008a51610a14565b915091506107448160000160005087878787610b41565b5b5050505050505050565b600060006000600060006000600060008861ffff16111561076f576108a3565b600160005060008b73ffffffffffffffffffffffffffffffffffffffff1916815260200190815260200160002060005091506000826002016000505414156107bc576003965086506108a3565b8160000160005090507f2a00000000000000000000000000000000000000000000000000000000000000600019168960001916148061083257508060000160009054906101000a9004700100000000000000000000000000000000026fffffffffffffffffffffffffffffffff19168960001916145b1561089e578060000160009054906101000a900470010000000000000000000000000000000002955085508060000160109054906101000a900463ffffffff16945084508060000160149054906101000a900461ffff16935083508060010160005054925082506108a3565b6108a3565b5050939792965093509350565b60006000600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561091057610002565b6109248461091f85610dfd9090565b610e47565b9150915060008160020160005054141561093d57610002565b60018160020160005054111561097c5761097781600001600050600070010000000000000000000000000000000002600060006000610b41565b610991565b6109908461098b85610dfd9090565b611197565b5b5b50505050565b600060006000600060006109ac87876116db565b92508250600160005060008473ffffffffffffffffffffffffffffffffffffffff1916815260200190815260200160002060005090506000816002016000505414156109fd57600394508450610a0a565b610e109350835030915081505b5092959194509250565b6000600060006000600060018688010392505b8683121515610af857610a4e898985815181101561000257906020019060200201516116db565b9150600160005060008373ffffffffffffffffffffffffffffffffffffffff191681526020019081526020016000206000509050600081600201600050541415610ae457600181600201600050819055506001600160005060008b73ffffffffffffffffffffffffffffffffffffffff191681526020019081526020016000206000506002016000828282505401925050819055505b81985088505b828060019003935050610a27565b88600160005060008b73ffffffffffffffffffffffffffffffffffffffff1916815260200190815260200160002060005080905094509450610b35565b50505094509492505050565b60208150602060ff161115610b5557610002565b838560000160006101000a8154816fffffffffffffffffffffffffffffffff021916908370010000000000000000000000000000000090040217905550828560000160106101000a81548163ffffffff02191690830217905550818560000160146101000a81548161ffff021916908302179055508085600101600050819055505b5050505050565b60006000610bf486866001870160018703610a14565b9150915060016000506000610c1e84886000815181101561000257906020019060200201516116db565b73ffffffffffffffffffffffffffffffffffffffff1916815260200190815260200160002060006000820160006000820160006101000a8154906fffffffffffffffffffffffffffffffff02191690556000820160106101000a81549063ffffffff02191690556000820160146101000a81549061ffff02191690556001820160005060009055505060028201600050600090556003820160005080546000825590600052602060002090810190610d539190610cd6565b80821115610d4f576000818150805460018160011615610100020316600290046000825580601f10610d085750610d45565b601f016020900490600052602060002090810190610d449190610d26565b80821115610d405760008181506000905550600101610d26565b5090565b5b5050600101610cd6565b5090565b5b50505060018160020160008282825054039250508190555060018160020160005054148015610dd057506000700100000000000000000000000000000000028160000160005060000160009054906101000a9004700100000000000000000000000000000000026fffffffffffffffffffffffffffffffff1916145b8015610ddc5750600183115b15610df457610df386866001870160018703610bde565b5b5b505050505050565b604060405190810160405280600081526020016000815260200150600060208301905060406040519081016040528084518152602001828152602001509150610e41565b50919050565b6000600060406040519081016040528060008152602001600081526020015060406040519081016040528060008152602001600081526020015060006000610ec9604060405190810160405280600181526020017f2e00000000000000000000000000000000000000000000000000000000000000815260200150610dfd9090565b93505b610ed7876117219090565b151561114f57610eea8488611737909190565b9250610ef7836117219090565b15610f0157610002565b610f1588610f10856117659090565b6116db565b9150600160005060008373ffffffffffffffffffffffffffffffffffffffff19168152602001908152602001600020600050905060008160020160005054141561114557600181600301600050818180549050019150818154818355818115116110105781836000526020600020918201910161100f9190610f92565b8082111561100b576000818150805460018160011615610100020316600290046000825580601f10610fc45750611001565b601f0160209004906000526020600020908101906110009190610fe2565b80821115610ffc5760008181506000905550600101610fe2565b5090565b5b5050600101610f92565b5090565b5b5050505061101f836117779090565b816003016000506001836003016000508054905003815481101561000257906000526020600020900160005b509080519060200190828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061109557805160ff19168380011785556110c6565b828001600101855582156110c6579182015b828111156110c55782518260005055916020019190600101906110a7565b5b5090506110f191906110d3565b808211156110ed57600081815060009055506001016110d3565b5090565b5050600181600201600050819055506001600160005060008a73ffffffffffffffffffffffffffffffffffffffff191681526020019081526020016000206000506002016000828282505401925050819055505b8197508750610ecc565b87600160005060008a73ffffffffffffffffffffffffffffffffffffffff191681526020019081526020016000206000508090509550955061118c565b505050509250929050565b6040604051908101604052806000815260200160008152602001506000600060006000600061120c611203604060405190810160405280600181526020017f2e00000000000000000000000000000000000000000000000000000000000000815260200150610dfd9090565b886117ee909190565b95506112228861121d8961181c9090565b610e47565b9450945083600301600050805490509250600091505b828210156114dc5761130f866113078660030160005085815481101561000257906000526020600020900160005b508054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156112fb5780601f106112d0576101008083540402835291602001916112fb565b820191906000526020600020905b8154815290600101906020018083116112de57829003601f168201915b5050505050610dfd9090565b611864909190565b156114ce578360030160005060018403815481101561000257906000526020600020900160005b508460030160005083815481101561000257906000526020600020900160005b509080546001816001161561010002031660029004828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106113a457805485556113e1565b828001600101855582156113e157600052602060002091601f016020900482015b828111156113e05782548255916001019190600101906113c5565b5b50905061140c91906113ee565b8082111561140857600081815060009055506001016113ee565b5090565b5050600184600301600050818180549050039150818154818355818115116114c5578183600052602060002091820191016114c49190611447565b808211156114c0576000818150805460018160011615610100020316600290046000825580601f1061147957506114b6565b601f0160209004906000526020600020908101906114b59190611497565b808211156114b15760008181506000905550600101611497565b5090565b5b5050600101611447565b5090565b5b505050506114dc565b5b8180600101925050611238565b6114e7866117659090565b9050600160005060006114fa87846116db565b73ffffffffffffffffffffffffffffffffffffffff1916815260200190815260200160002060006000820160006000820160006101000a8154906fffffffffffffffffffffffffffffffff02191690556000820160106101000a81549063ffffffff02191690556000820160146101000a81549061ffff0219169055600182016000506000905550506002820160005060009055600382016000508054600082559060005260206000209081019061162f91906115b2565b8082111561162b576000818150805460018160011615610100020316600290046000825580601f106115e45750611621565b601f0160209004906000526020600020908101906116209190611602565b8082111561161c5760008181506000905550600101611602565b5090565b5b50506001016115b2565b5090565b5b505050600184600201600082828250540392505081905550600184600201600050541480156116ac57506000700100000000000000000000000000000000028460000160005060000160009054906101000a9004700100000000000000000000000000000000026fffffffffffffffffffffffffffffffff1916145b80156116c057506116be876117219090565b155b156116d0576116cf8888611197565b5b5b5050505050505050565b60008282604051808373ffffffffffffffffffffffffffffffffffffffff19168152600c0182600019168152602001925050506040518091039020905080505b92915050565b600060008260000151149050611732565b919050565b60406040519081016040528060008152602001600081526020015061175d838383611880565b505b92915050565b6000815160208301512090505b919050565b60206040519081016040528060008152602001506020604051908101604052806000815260200150600083600001516040518059106117b35750595b908082528060200260200182016040525091506020820190506117df8185602001518660000151611936565b8192506117e7565b5050919050565b604060405190810160405280600081526020016000815260200150611814838383611989565b505b92915050565b604060405190810160405280600081526020016000815260200150604060405190810160405280836000015181526020018360200151815260200150905061185f565b919050565b600060006118728484611a58565b14905061187a565b92915050565b60406040519081016040528060008152602001600081526020015060006118b98560000151866020015186600001518760200151611b23565b905080836020019090818152602001505084602001518103856000015103836000019090818152602001505084602001518114156119065760008560000190908181526020015050611926565b836000015183600001510185600001818151039150909081815260200150505b82915061192e565b509392505050565b60005b6020821015156119655782518452602084019350835060208301925082505b6020820391508150611939565b6001826020036101000a039050801983511681855116818117865250505b50505050565b60406040519081016040528060008152602001600081526020015060006119c28560000151866020015186600001518760200151611be7565b9050846020015183602001909081815260200150508460200151810383600001909081815260200150508460000151856020015101811415611a135760008560000190908181526020015050611a48565b836000015183600001510185600001818151039150909081815260200150508360000151810185602001909081815260200150505b829150611a50565b509392505050565b6000600060006000600060006000600060008a6000015197508a600001518a600001511015611a8b578960000151975087505b8a60200151965089602001519550600094505b87851015611b035786519350855192508284141515611ae557600185896020030160080260020a03199150818316828516039050600081141515611ae457809850611b15565b5b602087019650865060208601955085505b6020850194508450611a9e565b89600001518b60000151039850611b15565b505050505050505092915050565b60006000600060008786111515611bd457602086111515611b8e5760018660200360080260020a031980865116878a03890194505b808286511614611b7a57600185039450886001860111611b5857889450611b80565b87850194505b5050829350611bdc56611bd3565b85852091508588038701925082505b8683101515611bd2578583209050806000191682600019161415611bc5578583019350611bdc565b6001830392508250611b9d565b5b5b869350611bdc565b505050949350505050565b600060006000600060008887111515611c9f57602087111515611c4e5760018760200360080260020a031980875116888b038a018a96505b818388511614611c3f57600187019650806001880310611c1f578b8b0196505b505050839450611ca956611c9e565b868620915087935083506000925082505b86890383111515611c9d578684209050806000191682600019161415611c8757839450611ca9565b60018401935083505b8280600101935050611c5f565b5b5b8888019450611ca9565b5050505094935050505056`

// DeployPersonalResolver deploys a new Ethereum contract, binding an instance of PersonalResolver to it.
func DeployPersonalResolver(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *PersonalResolver, error) {
	parsed, err := abi.JSON(strings.NewReader(PersonalResolverABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(PersonalResolverBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PersonalResolver{PersonalResolverCaller: PersonalResolverCaller{contract: contract}, PersonalResolverTransactor: PersonalResolverTransactor{contract: contract}}, nil
}

// PersonalResolver is an auto generated Go binding around an Ethereum contract.
type PersonalResolver struct {
	PersonalResolverCaller     // Read-only binding to the contract
	PersonalResolverTransactor // Write-only binding to the contract
}

// PersonalResolverCaller is an auto generated read-only Go binding around an Ethereum contract.
type PersonalResolverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PersonalResolverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PersonalResolverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PersonalResolverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PersonalResolverSession struct {
	Contract     *PersonalResolver // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PersonalResolverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PersonalResolverCallerSession struct {
	Contract *PersonalResolverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// PersonalResolverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PersonalResolverTransactorSession struct {
	Contract     *PersonalResolverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// PersonalResolverRaw is an auto generated low-level Go binding around an Ethereum contract.
type PersonalResolverRaw struct {
	Contract *PersonalResolver // Generic contract binding to access the raw methods on
}

// PersonalResolverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PersonalResolverCallerRaw struct {
	Contract *PersonalResolverCaller // Generic read-only contract binding to access the raw methods on
}

// PersonalResolverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PersonalResolverTransactorRaw struct {
	Contract *PersonalResolverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPersonalResolver creates a new instance of PersonalResolver, bound to a specific deployed contract.
func NewPersonalResolver(address common.Address, backend bind.ContractBackend) (*PersonalResolver, error) {
	contract, err := bindPersonalResolver(address, backend.(bind.ContractCaller), backend.(bind.ContractTransactor))
	if err != nil {
		return nil, err
	}
	return &PersonalResolver{PersonalResolverCaller: PersonalResolverCaller{contract: contract}, PersonalResolverTransactor: PersonalResolverTransactor{contract: contract}}, nil
}

// NewPersonalResolverCaller creates a new read-only instance of PersonalResolver, bound to a specific deployed contract.
func NewPersonalResolverCaller(address common.Address, caller bind.ContractCaller) (*PersonalResolverCaller, error) {
	contract, err := bindPersonalResolver(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &PersonalResolverCaller{contract: contract}, nil
}

// NewPersonalResolverTransactor creates a new write-only instance of PersonalResolver, bound to a specific deployed contract.
func NewPersonalResolverTransactor(address common.Address, transactor bind.ContractTransactor) (*PersonalResolverTransactor, error) {
	contract, err := bindPersonalResolver(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &PersonalResolverTransactor{contract: contract}, nil
}

// bindPersonalResolver binds a generic wrapper to an already deployed contract.
func bindPersonalResolver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PersonalResolverABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PersonalResolver *PersonalResolverRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _PersonalResolver.Contract.PersonalResolverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PersonalResolver *PersonalResolverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PersonalResolver.Contract.PersonalResolverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PersonalResolver *PersonalResolverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PersonalResolver.Contract.PersonalResolverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PersonalResolver *PersonalResolverCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _PersonalResolver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PersonalResolver *PersonalResolverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PersonalResolver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PersonalResolver *PersonalResolverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PersonalResolver.Contract.contract.Transact(opts, method, params...)
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_PersonalResolver *PersonalResolverCaller) FindResolver(opts *bind.CallOpts, nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	ret := new(struct {
		Rcode    uint16
		Ttl      uint32
		Rnode    [12]byte
		Raddress common.Address
	})
	out := ret
	err := _PersonalResolver.contract.Call(opts, out, "findResolver", nodeId, label)
	return *ret, err
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_PersonalResolver *PersonalResolverSession) FindResolver(nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	return _PersonalResolver.Contract.FindResolver(&_PersonalResolver.CallOpts, nodeId, label)
}

// FindResolver is a free data retrieval call binding the contract method 0xedc0277c.
//
// Solidity: function findResolver(nodeId bytes12, label bytes32) constant returns(rcode uint16, ttl uint32, rnode bytes12, raddress address)
func (_PersonalResolver *PersonalResolverCallerSession) FindResolver(nodeId [12]byte, label [32]byte) (struct {
	Rcode    uint16
	Ttl      uint32
	Rnode    [12]byte
	Raddress common.Address
}, error) {
	return _PersonalResolver.Contract.FindResolver(&_PersonalResolver.CallOpts, nodeId, label)
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_PersonalResolver *PersonalResolverCaller) GetExtended(opts *bind.CallOpts, id [32]byte) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _PersonalResolver.contract.Call(opts, out, "getExtended", id)
	return *ret0, err
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_PersonalResolver *PersonalResolverSession) GetExtended(id [32]byte) ([]byte, error) {
	return _PersonalResolver.Contract.GetExtended(&_PersonalResolver.CallOpts, id)
}

// GetExtended is a free data retrieval call binding the contract method 0x8021061c.
//
// Solidity: function getExtended(id bytes32) constant returns(data bytes)
func (_PersonalResolver *PersonalResolverCallerSession) GetExtended(id [32]byte) ([]byte, error) {
	return _PersonalResolver.Contract.GetExtended(&_PersonalResolver.CallOpts, id)
}

// IsPersonalResolver is a free data retrieval call binding the contract method 0x3f5665e7.
//
// Solidity: function isPersonalResolver() constant returns(bool)
func (_PersonalResolver *PersonalResolverCaller) IsPersonalResolver(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _PersonalResolver.contract.Call(opts, out, "isPersonalResolver")
	return *ret0, err
}

// IsPersonalResolver is a free data retrieval call binding the contract method 0x3f5665e7.
//
// Solidity: function isPersonalResolver() constant returns(bool)
func (_PersonalResolver *PersonalResolverSession) IsPersonalResolver() (bool, error) {
	return _PersonalResolver.Contract.IsPersonalResolver(&_PersonalResolver.CallOpts)
}

// IsPersonalResolver is a free data retrieval call binding the contract method 0x3f5665e7.
//
// Solidity: function isPersonalResolver() constant returns(bool)
func (_PersonalResolver *PersonalResolverCallerSession) IsPersonalResolver() (bool, error) {
	return _PersonalResolver.Contract.IsPersonalResolver(&_PersonalResolver.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_PersonalResolver *PersonalResolverCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _PersonalResolver.contract.Call(opts, out, "owner")
	return *ret0, err
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_PersonalResolver *PersonalResolverSession) Owner() (common.Address, error) {
	return _PersonalResolver.Contract.Owner(&_PersonalResolver.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_PersonalResolver *PersonalResolverCallerSession) Owner() (common.Address, error) {
	return _PersonalResolver.Contract.Owner(&_PersonalResolver.CallOpts)
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_PersonalResolver *PersonalResolverCaller) Resolve(opts *bind.CallOpts, nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	ret := new(struct {
		Rcode uint16
		Rtype [16]byte
		Ttl   uint32
		Len   uint16
		Data  [32]byte
	})
	out := ret
	err := _PersonalResolver.contract.Call(opts, out, "resolve", nodeId, qtype, index)
	return *ret, err
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_PersonalResolver *PersonalResolverSession) Resolve(nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	return _PersonalResolver.Contract.Resolve(&_PersonalResolver.CallOpts, nodeId, qtype, index)
}

// Resolve is a free data retrieval call binding the contract method 0xa16fdafa.
//
// Solidity: function resolve(nodeId bytes12, qtype bytes32, index uint16) constant returns(rcode uint16, rtype bytes16, ttl uint32, len uint16, data bytes32)
func (_PersonalResolver *PersonalResolverCallerSession) Resolve(nodeId [12]byte, qtype [32]byte, index uint16) (struct {
	Rcode uint16
	Rtype [16]byte
	Ttl   uint32
	Len   uint16
	Data  [32]byte
}, error) {
	return _PersonalResolver.Contract.Resolve(&_PersonalResolver.CallOpts, nodeId, qtype, index)
}

// DeletePrivateRR is a paid mutator transaction binding the contract method 0x1b370194.
//
// Solidity: function deletePrivateRR(rootNodeId bytes12, name bytes32[]) returns()
func (_PersonalResolver *PersonalResolverTransactor) DeletePrivateRR(opts *bind.TransactOpts, rootNodeId [12]byte, name [][32]byte) (*types.Transaction, error) {
	return _PersonalResolver.contract.Transact(opts, "deletePrivateRR", rootNodeId, name)
}

// DeletePrivateRR is a paid mutator transaction binding the contract method 0x1b370194.
//
// Solidity: function deletePrivateRR(rootNodeId bytes12, name bytes32[]) returns()
func (_PersonalResolver *PersonalResolverSession) DeletePrivateRR(rootNodeId [12]byte, name [][32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.DeletePrivateRR(&_PersonalResolver.TransactOpts, rootNodeId, name)
}

// DeletePrivateRR is a paid mutator transaction binding the contract method 0x1b370194.
//
// Solidity: function deletePrivateRR(rootNodeId bytes12, name bytes32[]) returns()
func (_PersonalResolver *PersonalResolverTransactorSession) DeletePrivateRR(rootNodeId [12]byte, name [][32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.DeletePrivateRR(&_PersonalResolver.TransactOpts, rootNodeId, name)
}

// DeleteRR is a paid mutator transaction binding the contract method 0xbc06183d.
//
// Solidity: function deleteRR(rootNodeId bytes12, name string) returns()
func (_PersonalResolver *PersonalResolverTransactor) DeleteRR(opts *bind.TransactOpts, rootNodeId [12]byte, name string) (*types.Transaction, error) {
	return _PersonalResolver.contract.Transact(opts, "deleteRR", rootNodeId, name)
}

// DeleteRR is a paid mutator transaction binding the contract method 0xbc06183d.
//
// Solidity: function deleteRR(rootNodeId bytes12, name string) returns()
func (_PersonalResolver *PersonalResolverSession) DeleteRR(rootNodeId [12]byte, name string) (*types.Transaction, error) {
	return _PersonalResolver.Contract.DeleteRR(&_PersonalResolver.TransactOpts, rootNodeId, name)
}

// DeleteRR is a paid mutator transaction binding the contract method 0xbc06183d.
//
// Solidity: function deleteRR(rootNodeId bytes12, name string) returns()
func (_PersonalResolver *PersonalResolverTransactorSession) DeleteRR(rootNodeId [12]byte, name string) (*types.Transaction, error) {
	return _PersonalResolver.Contract.DeleteRR(&_PersonalResolver.TransactOpts, rootNodeId, name)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(newOwner address) returns()
func (_PersonalResolver *PersonalResolverTransactor) SetOwner(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _PersonalResolver.contract.Transact(opts, "setOwner", newOwner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(newOwner address) returns()
func (_PersonalResolver *PersonalResolverSession) SetOwner(newOwner common.Address) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetOwner(&_PersonalResolver.TransactOpts, newOwner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(newOwner address) returns()
func (_PersonalResolver *PersonalResolverTransactorSession) SetOwner(newOwner common.Address) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetOwner(&_PersonalResolver.TransactOpts, newOwner)
}

// SetPrivateRR is a paid mutator transaction binding the contract method 0x91c8e7b9.
//
// Solidity: function setPrivateRR(rootNodeId bytes12, name bytes32[], rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverTransactor) SetPrivateRR(opts *bind.TransactOpts, rootNodeId [12]byte, name [][32]byte, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.contract.Transact(opts, "setPrivateRR", rootNodeId, name, rtype, ttl, len, data)
}

// SetPrivateRR is a paid mutator transaction binding the contract method 0x91c8e7b9.
//
// Solidity: function setPrivateRR(rootNodeId bytes12, name bytes32[], rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverSession) SetPrivateRR(rootNodeId [12]byte, name [][32]byte, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetPrivateRR(&_PersonalResolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}

// SetPrivateRR is a paid mutator transaction binding the contract method 0x91c8e7b9.
//
// Solidity: function setPrivateRR(rootNodeId bytes12, name bytes32[], rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverTransactorSession) SetPrivateRR(rootNodeId [12]byte, name [][32]byte, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetPrivateRR(&_PersonalResolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}

// SetRR is a paid mutator transaction binding the contract method 0x8bba944d.
//
// Solidity: function setRR(rootNodeId bytes12, name string, rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverTransactor) SetRR(opts *bind.TransactOpts, rootNodeId [12]byte, name string, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.contract.Transact(opts, "setRR", rootNodeId, name, rtype, ttl, len, data)
}

// SetRR is a paid mutator transaction binding the contract method 0x8bba944d.
//
// Solidity: function setRR(rootNodeId bytes12, name string, rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverSession) SetRR(rootNodeId [12]byte, name string, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetRR(&_PersonalResolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}

// SetRR is a paid mutator transaction binding the contract method 0x8bba944d.
//
// Solidity: function setRR(rootNodeId bytes12, name string, rtype bytes16, ttl uint32, len uint16, data bytes32) returns()
func (_PersonalResolver *PersonalResolverTransactorSession) SetRR(rootNodeId [12]byte, name string, rtype [16]byte, ttl uint32, len uint16, data [32]byte) (*types.Transaction, error) {
	return _PersonalResolver.Contract.SetRR(&_PersonalResolver.TransactOpts, rootNodeId, name, rtype, ttl, len, data)
}
