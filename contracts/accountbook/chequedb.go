// Copyright 2019 The go-ethereum Authors
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

package accountbook

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	// Database schema definitions
	//                                                                            +-----------+
	//                                             >   Drawer1(client) ---------> |  cheque1  |
	//        Cheque Drawee(Les server)          -/                               +-----------+
	//                                         -/
	//   +------------+      +------------+  -/                                   +-----------+
	//   |server addr1|----->|  Contract1 |-/------->  Drawer2(client) ---------> |  cheque2  |
	//   +------------+      +------------+  -\                                    +-----------+
	//                                         -\
	//                                           -\                                +-----------+
	//                                             >   Drawer3(client) ---------> |  cheque3  |
	//                                                                            +-----------+
	//                  ...
	//   +------------+      +------------+
	//   |server addrn|----->|  Contractn |
	//   +------------+      +------------+
	//
	//       Cheque Drawer(Light client)
	//                                                   +---------------+
	//                     -> Drawee1(server) ---------> |  last issued  |
	//   +------------+  -/                              +---------------+
	//   |client addr1|-/---> Drawee2(server) ---------> |  last issued  |
	//   +------------+  -\                              +---------------+
	//                     -> Drawee3(server) ---------> |  last issued  |
	//                                                   +---------------+
	//        ...
	//   +------------+
	//   |client addrn|
	//   +------------+
	contractAddrPrefix = []byte("-a") // contractAddrPrefix + deployer(20bytes) -> contract_addr
	chequePrefix       = []byte("-c") // chequePrefix + contract_addr(20bytes) + drawer_id(20bytes) -> cheque
	issuedPrefix       = []byte("-i") // issuedPrefix + drawer_id(20bytes) + contract_addr(20bytes) -> big-endian number
)

// chequeDB keeps all signed cheques issued by customers. It's very important
// to save the cheques properly, otherwise the owner of accountbook can't claim
// the money back.
//
// Cheques are cumulatively confirmed, so only the latest version needs to be stored.
type chequeDB struct {
	db ethdb.Database
}

// newChequeDB intiailises the chequedb with given db handler.
func newChequeDB(db ethdb.Database) *chequeDB { return &chequeDB{db: db} }

// readContractAddr returns the contract address deployed by specified deployer.
func (db *chequeDB) readContractAddr(deployer common.Address) *common.Address {
	blob, err := db.db.Get(append(contractAddrPrefix, deployer.Bytes()...))
	if err != nil {
		return nil
	}
	if len(blob) != common.AddressLength {
		return nil
	}
	addr := common.BytesToAddress(blob)
	return &addr
}

// writeContractAddr writes the contract address which deployed by current address.
func (db *chequeDB) writeContractAddr(deployer, contractAddr common.Address) {
	if err := db.db.Put(append(contractAddrPrefix, deployer.Bytes()...), contractAddr.Bytes()); err != nil {
		log.Crit("Failed to write contract address", "err", err)
	}
}

// readCheque returns the last issued cheque for the specified drawer.
// If there is no local_addr => contract_addr mapping, it means we haven't
// deployed the contract yet.
func (db *chequeDB) readCheque(contractAddr, drawer common.Address) *Cheque {
	blob, err := db.db.Get(append(append(chequePrefix, contractAddr.Bytes()...), drawer.Bytes()...))
	if err != nil {
		return nil
	}
	var cheque Cheque
	if err := rlp.DecodeBytes(blob, &cheque); err != nil {
		return nil
	}
	return &cheque
}

// writeCheque writes the last issued cheque from the specific drawer
// into disk.
func (db *chequeDB) writeCheque(contractAddr, drawer common.Address, cheque *Cheque) {
	blob, err := rlp.EncodeToBytes(cheque)
	if err != nil {
		log.Crit("Failed to encode cheque", "error", err)
	}
	err = db.db.Put(append(append(chequePrefix, contractAddr.Bytes()...), drawer.Bytes()...), blob)
	if err != nil {
		log.Crit("Failed to store cheque", "error", err)
	}
}

// readLastIssued returns the last issued amount by local address to
// specified contract address.
func (db *chequeDB) readLastIssued(drawer, contractAddr common.Address) *big.Int {
	blob, err := db.db.Get(append(append(issuedPrefix, drawer.Bytes()...), contractAddr.Bytes()...))
	if err != nil {
		return nil
	}
	return new(big.Int).SetBytes(blob)
}

// writeLastIssued writes the last issued amount by local address to
// specified contract address into the disk.
func (db *chequeDB) writeLastIssued(drawer, contractAddr common.Address, amount *big.Int) {
	if err := db.db.Put(append(append(issuedPrefix, drawer.Bytes()...), contractAddr.Bytes()...), amount.Bytes()); err != nil {
		log.Crit("Failed to store last issue amount", "error", err)
	}
}

// allCheques returns all received cheques from different drawers.
func (db *chequeDB) allCheques(contractAddr common.Address) (cheques []*Cheque) {
	iter := db.db.NewIteratorWithPrefix(append(chequePrefix, contractAddr.Bytes()...))
	defer iter.Release()
	for iter.Next() {
		var cheque Cheque
		if err := rlp.DecodeBytes(iter.Value(), &cheque); err != nil {
			continue
		}
		cheques = append(cheques, &cheque)
	}
	return
}

// allIssued returns all issued amount from local address to different
// contracts.
func (db *chequeDB) allIssued(drawer common.Address) (addresses []common.Address, amounts []*big.Int) {
	iter := db.db.NewIteratorWithPrefix(append(issuedPrefix, drawer.Bytes()...))
	defer iter.Release()
	for iter.Next() {
		var addr common.Address
		if len(iter.Key()) != len(issuedPrefix)+common.AddressLength+common.AddressLength {
			continue
		}
		amount := new(big.Int).SetBytes(iter.Value())
		copy(addr[:], iter.Key()[len(iter.Key())-common.AddressLength:])
		addresses = append(addresses, addr)
		amounts = append(amounts, amount)
	}
	return
}
