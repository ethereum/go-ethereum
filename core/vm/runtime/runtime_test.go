// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package runtime

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestDefaults(t *testing.T) {
	cfg := new(Config)
	setDefaults(cfg)

	if cfg.Difficulty == nil {
		t.Error("expected difficulty to be non nil")
	}

	if cfg.Time == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.GasLimit == 0 {
		t.Error("didn't expect gaslimit to be zero")
	}
	if cfg.GasPrice == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.Value == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.GetHashFn == nil {
		t.Error("expected time to be non nil")
	}
	if cfg.BlockNumber == nil {
		t.Error("expected block number to be non nil")
	}
}

func TestEVM(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("crashed with: %v", r)
		}
	}()

	Execute([]byte{
		byte(vm.DIFFICULTY),
		byte(vm.TIMESTAMP),
		byte(vm.GASLIMIT),
		byte(vm.PUSH1),
		byte(vm.ORIGIN),
		byte(vm.BLOCKHASH),
		byte(vm.COINBASE),
	}, nil, nil)
}

func TestExecute(t *testing.T) {
	ret, _, _, err := Execute([]byte{
		byte(vm.PUSH1), 10,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}, nil, nil)
	if err != nil {
		t.Fatal("didn't expect error", err)
	}

	num := new(big.Int).SetBytes(ret)
	if num.Cmp(big.NewInt(10)) != 0 {
		t.Error("Expected 10, got", num)
	}
}

func TestCall(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	state, _ := state.New(common.Hash{}, state.NewDatabase(db))
	address := common.HexToAddress("0x0a")
	state.SetCode(address, []byte{
		byte(vm.PUSH1), 10,
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	})

	ret, _, _, err := Call(address, nil, &Config{State: state})
	if err != nil {
		t.Fatal("didn't expect error", err)
	}

	num := new(big.Int).SetBytes(ret)
	if num.Cmp(big.NewInt(10)) != 0 {
		t.Error("Expected 10, got", num)
	}
}

func TestTransactionReverted(t *testing.T) {
	/*
	 	   	Contract Source Code
	 	   	```
	 	   	contract Demo {
	 	       		function Demo() {}
	 	       		function IllegalDivision() returns(int) {
	 	           		var dividend = 0;
	 	           		return 1 / dividend;
	 	       		}
	 	       		function LegalDivision() returns(int) {
	 	           		var dividend = 1;
	 	           		return 1 / dividend;
	        		}
	 	   	}
	 	   	```
	*/
	var definition = `[{"constant":false,"inputs":[],"name":"LegalDivision","outputs":[{"name":"","type":"int256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[],"name":"IllegalDivision","outputs":[{"name":"","type":"int256"}],"payable":false,"type":"function"},{"inputs":[],"payable":false,"type":"constructor"}]`
	var rawcode = common.Hex2Bytes("6060604052341561000c57fe5b5b5b5b60f88061001d6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680632d256662146044578063522de105146067575bfe5b3415604b57fe5b6051608a565b6040518082815260200191505060405180910390f35b3415606e57fe5b607460ab565b6040518082815260200191505060405180910390f35b60006000600190508060ff16600181151560a057fe5b0460ff1691505b5090565b60006000600090508060ff16600181151560c157fe5b0460ff1691505b50905600a165627a7a7230582091585859014c4644d0427bf34abc65433b5874993ddea00cac57bcb87ee4cb6b0029")

	abi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	legalDivision, err := abi.Pack("LegalDivision")
	if err != nil {
		t.Fatal(err)
	}

	illegalDivision, err := abi.Pack("IllegalDivision")
	if err != nil {
		t.Fatal(err)
	}
	var failed bool
	// deploy
	cfg := &Config{
		Origin: common.HexToAddress("sender"),
	}
	code, _, _, _, err := Create(rawcode, cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, _, failed, _ = Execute(code, legalDivision, cfg)
	if failed != false {
		t.Fatal("Expect false, got true")
	}
	_, _, failed, _ = Execute(code, illegalDivision, cfg)
	if failed != true {
		t.Fatal("Expect true, got false")
	}
}

func TestDelegateReverted(t *testing.T) {
	/*
		 		Contract Source Code
		 		```
		 		contract Relay {
		 		    address public currentVersion;
		 		    address public owner;

				    modifier onlyOwner() {
		 			if (msg.sender != owner) {
		 			    throw;
		 			}
		 			_;
		 		    }
		 		    function Relay(address _address) {
		 			currentVersion = _address;
		 			owner = msg.sender;
		 		    }
		 		    function changeContract(address newVersion) public
		 		    onlyOwner()
		 		    {
		 			currentVersion = newVersion;
		 		    }
		 		    function() {
		 			if(!currentVersion.delegatecall(msg.data)) throw;
		 		    }
				}

		 		contract Demo {
		 		    function Demo() {
		 		    }
		 		    function IllegalDivision() returns(int) {
		 			var dividend = 0;
		 			return 1 / dividend;
		 		    }
		 		    function LegalDivision() returns(int) {
		 			var dividend = 1;
		 			return 1 / dividend;
		 		    }
		 		}
		 		```
	*/
	var definition = `[{"constant":false,"inputs":[],"name":"LegalDivision","outputs":[{"name":"","type":"int256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[],"name":"IllegalDivision","outputs":[{"name":"","type":"int256"}],"payable":false,"type":"function"},{"inputs":[],"payable":false,"type":"constructor"}]`
	var rawcode1 = common.Hex2Bytes("6060604052341561000c57fe5b5b5b5b60f88061001d6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680632d256662146044578063522de105146067575bfe5b3415604b57fe5b6051608a565b6040518082815260200191505060405180910390f35b3415606e57fe5b607460ab565b6040518082815260200191505060405180910390f35b60006000600190508060ff16600181151560a057fe5b0460ff1691505b5090565b60006000600090508060ff16600181151560c157fe5b0460ff1691505b50905600a165627a7a7230582091585859014c4644d0427bf34abc65433b5874993ddea00cac57bcb87ee4cb6b0029")
	var rawcode2 = common.Hex2Bytes("6060604052341561000c57fe5b60405160208061039c833981016040528080519060200190919050505b80600060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555033600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505b505b6102df806100bd6000396000f30060606040523615610055576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680633d71c3af146100ea5780638da5cb5b146101205780639d888e8614610172575b341561005d57fe5b6100e85b600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600036600060405160200152604051808383808284378201915050925050506020604051808303818560325a03f415156100d057fe5b50506040518051905015156100e55760006000fd5b5b565b005b34156100f257fe5b61011e600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919050506101c4565b005b341561012857fe5b610130610267565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b341561017a57fe5b61018261028d565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156102215760006000fd5b80600060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505b5b50565b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16815600a165627a7a723058208496aa0ef6d67e2e0423b76e8fd61b92e0047fe2e20ad20f887caf2b378dfa300029")

	abi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	legalDivision, err := abi.Pack("LegalDivision")
	if err != nil {
		t.Fatal(err)
	}

	illegalDivision, err := abi.Pack("IllegalDivision")
	if err != nil {
		t.Fatal(err)
	}
	var failed bool
	// deploy
	cfg := &Config{
		Origin: common.HexToAddress("sender"),
	}
	_, addr, _, _, err := Create(rawcode1, cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, addr2, _, _, err := Create(append(rawcode2, common.LeftPadBytes(addr.Bytes(), 32)...), cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, _, failed, _ = Call(addr2, legalDivision, cfg)
	if failed != false {
		t.Fatal("Expect false, got true")
	}
	_, _, failed, _ = Call(addr2, illegalDivision, cfg)
	if failed != true {
		t.Fatal("Expect true, got false")
	}
}

func BenchmarkCall(b *testing.B) {
	var definition = `[{"constant":true,"inputs":[],"name":"seller","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":false,"inputs":[],"name":"abort","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"value","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[],"name":"refund","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"buyer","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":false,"inputs":[],"name":"confirmReceived","outputs":[],"type":"function"},{"constant":true,"inputs":[],"name":"state","outputs":[{"name":"","type":"uint8"}],"type":"function"},{"constant":false,"inputs":[],"name":"confirmPurchase","outputs":[],"type":"function"},{"inputs":[],"type":"constructor"},{"anonymous":false,"inputs":[],"name":"Aborted","type":"event"},{"anonymous":false,"inputs":[],"name":"PurchaseConfirmed","type":"event"},{"anonymous":false,"inputs":[],"name":"ItemReceived","type":"event"},{"anonymous":false,"inputs":[],"name":"Refunded","type":"event"}]`

	var code = common.Hex2Bytes("6060604052361561006c5760e060020a600035046308551a53811461007457806335a063b4146100865780633fa4f245146100a6578063590e1ae3146100af5780637150d8ae146100cf57806373fac6f0146100e1578063c19d93fb146100fe578063d696069714610112575b610131610002565b610133600154600160a060020a031681565b610131600154600160a060020a0390811633919091161461015057610002565b61014660005481565b610131600154600160a060020a039081163391909116146102d557610002565b610133600254600160a060020a031681565b610131600254600160a060020a0333811691161461023757610002565b61014660025460ff60a060020a9091041681565b61013160025460009060ff60a060020a9091041681146101cc57610002565b005b600160a060020a03166060908152602090f35b6060908152602090f35b60025460009060a060020a900460ff16811461016b57610002565b600154600160a060020a03908116908290301631606082818181858883f150506002805460a060020a60ff02191660a160020a179055506040517f72c874aeff0b183a56e2b79c71b46e1aed4dee5e09862134b8821ba2fddbf8bf9250a150565b80546002023414806101dd57610002565b6002805460a060020a60ff021973ffffffffffffffffffffffffffffffffffffffff1990911633171660a060020a1790557fd5d55c8a68912e9a110618df8d5e2e83b8d83211c57a8ddd1203df92885dc881826060a15050565b60025460019060a060020a900460ff16811461025257610002565b60025460008054600160a060020a0390921691606082818181858883f150508354604051600160a060020a0391821694503090911631915082818181858883f150506002805460a060020a60ff02191660a160020a179055506040517fe89152acd703c9d8c7d28829d443260b411454d45394e7995815140c8cbcbcf79250a150565b60025460019060a060020a900460ff1681146102f057610002565b6002805460008054600160a060020a0390921692909102606082818181858883f150508354604051600160a060020a0391821694503090911631915082818181858883f150506002805460a060020a60ff02191660a160020a179055506040517f8616bbbbad963e4e65b1366f1d75dfb63f9e9704bbbf91fb01bec70849906cf79250a15056")

	abi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		b.Fatal(err)
	}

	cpurchase, err := abi.Pack("confirmPurchase")
	if err != nil {
		b.Fatal(err)
	}
	creceived, err := abi.Pack("confirmReceived")
	if err != nil {
		b.Fatal(err)
	}
	refund, err := abi.Pack("refund")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 400; j++ {
			Execute(code, cpurchase, nil)
			Execute(code, creceived, nil)
			Execute(code, refund, nil)
		}
	}
}
