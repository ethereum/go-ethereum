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

package statediff_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/statediff"
	"github.com/ethereum/go-ethereum/statediff/testhelpers"
	sdtypes "github.com/ethereum/go-ethereum/statediff/types"
)

var (
	db                                                         ethdb.Database
	genesisBlock, block0, block1, block2, block3               *types.Block
	block1CoinbaseAddr, block2CoinbaseAddr, block3CoinbaseAddr common.Address
	block1CoinbaseHash, block2CoinbaseHash, block3CoinbaseHash common.Hash
	builder                                                    statediff.Builder
	emptyStorage                                               = make([]sdtypes.StorageNode, 0)

	// block 1 data
	block1CoinbaseAccount, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  big.NewInt(5000000000000000000),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	block1CoinbaseLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("38251692195afc818c92b485fcb8a4691af89cbe5a2ab557b83a4261be2a9a"),
		block1CoinbaseAccount,
	})
	block1CoinbaseLeafNodeHash = crypto.Keccak256(block1CoinbaseLeafNode)
	block1x040bBranchNode, _   = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("cc947d5ebb80600bad471f12c6ad5e4981e3525ecf8a2d982cc032536ae8b66d"),
		common.Hex2Bytes("e80e52462e635a834e90e86ccf7673a6430384aac17004d626f4db831f0624bc"),
		common.Hex2Bytes("59a8f11f60cb0a8488831f242da02944a26fd269d0608a44b8b873ded9e59e1b"),
		common.Hex2Bytes("1ffb51e987e3cbd2e1dc1a64508d2e2b265477e21698b0d10fdf137f35027f40"),
		[]byte{},
		common.Hex2Bytes("ce5077f49a13ff8199d0e77715fdd7bfd6364774effcd5499bd93cba54b3c644"),
		common.Hex2Bytes("f5146783c048e66ce1a776ae990b4255e5fba458ece77fcb83ff6e91d6637a88"),
		common.Hex2Bytes("6a0558b6c38852e985cf01c2156517c1c6a1e64c787a953c347825f050b236c6"),
		common.Hex2Bytes("56b6e93958b99aaae158cc2329e71a1865ba6f39c67b096922c5cf3ed86b0ae5"),
		[]byte{},
		common.Hex2Bytes("50d317a89a3405367d66668902f2c9f273a8d0d7d5d790dc516bca142f4a84af"),
		common.Hex2Bytes("c72ca72750fdc1af3e6da5c7c5d82c54e4582f15b488a8aa1674058a99825dae"),
		common.Hex2Bytes("e1a489df7b18cde818da6d38e235b026c2e61bcd3d34880b3ed0d67e0e4f0159"),
		common.Hex2Bytes("b58d5062f2609fd2d68f00d14ab33fef2b373853877cf40bf64729e85b8fdc54"),
		block1CoinbaseLeafNodeHash,
		[]byte{},
		[]byte{},
	})
	block1x040bBranchNodeHash = crypto.Keccak256(block1x040bBranchNode)
	block1x04BranchNode, _    = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("a9317a59365ca09cefcd384018696590afffc432e35a97e8f85aa48907bf3247"),
		common.Hex2Bytes("e0bc229254ce7a6a736c3953e570ab18b4a7f5f2a9aa3c3057b5f17d250a1cad"),
		common.Hex2Bytes("a2484ec8884dbe0cf24ece99d67df0d1fe78992d67cc777636a817cb2ef205aa"),
		common.Hex2Bytes("12b78d4078c607747f06bb88bd08f839eaae0e3ac6854e5f65867d4f78abb84e"),
		common.Hex2Bytes("359a51862df5462e4cd302f69cb338512f21eb37ce0791b9a562e72ec48b7dbf"),
		common.Hex2Bytes("13f8d617b6a734da9235b6ac80bdd7aeaff6120c39aa223638d88f22d4ba4007"),
		common.Hex2Bytes("02055c6400e0ec3440a8bb8fdfd7d6b6c57b7bf83e37d7e4e983d416fdd8314e"),
		common.Hex2Bytes("4b1cca9eb3e47e805e7f4c80671a9fcd589fd6ddbe1790c3f3e177e8ede01b9e"),
		common.Hex2Bytes("70c3815efb23b986018089e009a38e6238b8850b3efd33831913ca6fa9240249"),
		common.Hex2Bytes("7084699d2e72a193fd75bb6108ae797b4661696eba2d631d521fc94acc7b3247"),
		common.Hex2Bytes("b2b3cd9f1e46eb583a6185d9a96b4e80125e3d75e6191fdcf684892ef52935cb"),
		block1x040bBranchNodeHash,
		common.Hex2Bytes("34d9ff0fee6c929424e52268dedbc596d10786e909c5a68d6466c2aba17387ce"),
		common.Hex2Bytes("7484d5e44b6ee6b10000708c37e035b42b818475620f9316beffc46531d1eebf"),
		common.Hex2Bytes("30c8a283adccf2742272563cd3d6710c89ba21eac0118bf5310cfb231bcca77f"),
		common.Hex2Bytes("4bae8558d2385b8d3bc6e6ede20bdbc5dbb0b5384c316ba8985682f88d2e506d"),
		[]byte{},
	})
	block1x04BranchNodeHash = crypto.Keccak256(block1x04BranchNode)
	block1RootBranchNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("90dcaf88c40c7bbc95a912cbdde67c175767b31173df9ee4b0d733bfdd511c43"),
		common.Hex2Bytes("babe369f6b12092f49181ae04ca173fb68d1a5456f18d20fa32cba73954052bd"),
		common.Hex2Bytes("473ecf8a7e36a829e75039a3b055e51b8332cbf03324ab4af2066bbd6fbf0021"),
		common.Hex2Bytes("bbda34753d7aa6c38e603f360244e8f59611921d9e1f128372fec0d586d4f9e0"),
		block1x04BranchNodeHash,
		common.Hex2Bytes("a5f3f2f7542148c973977c8a1e154c4300fec92f755f7846f1b734d3ab1d90e7"),
		common.Hex2Bytes("e823850f50bf72baae9d1733a36a444ab65d0a6faaba404f0583ce0ca4dad92d"),
		common.Hex2Bytes("f7a00cbe7d4b30b11faea3ae61b7f1f2b315b61d9f6bd68bfe587ad0eeceb721"),
		common.Hex2Bytes("7117ef9fc932f1a88e908eaead8565c19b5645dc9e5b1b6e841c5edbdfd71681"),
		common.Hex2Bytes("69eb2de283f32c11f859d7bcf93da23990d3e662935ed4d6b39ce3673ec84472"),
		common.Hex2Bytes("203d26456312bbc4da5cd293b75b840fc5045e493d6f904d180823ec22bfed8e"),
		common.Hex2Bytes("9287b5c21f2254af4e64fca76acc5cd87399c7f1ede818db4326c98ce2dc2208"),
		common.Hex2Bytes("6fc2d754e304c48ce6a517753c62b1a9c1d5925b89707486d7fc08919e0a94ec"),
		common.Hex2Bytes("7b1c54f15e299bd58bdfef9741538c7828b5d7d11a489f9c20d052b3471df475"),
		common.Hex2Bytes("51f9dd3739a927c89e357580a4c97b40234aa01ed3d5e0390dc982a7975880a0"),
		common.Hex2Bytes("89d613f26159af43616fd9455bb461f4869bfede26f2130835ed067a8b967bfb"),
		[]byte{},
	})

	// block 2 data
	block2CoinbaseAccount, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  big.NewInt(5000000000000000000),
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	block2CoinbaseLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("20679cbcf198c1741a6f4e4473845659a30caa8b26f8d37a0be2e2bc0d8892"),
		block2CoinbaseAccount,
	})
	block2CoinbaseLeafNodeHash   = crypto.Keccak256(block2CoinbaseLeafNode)
	block2MovedPremineBalance, _ = new(big.Int).SetString("4000000000000000000000", 10)
	block2MovedPremineAccount, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  block2MovedPremineBalance,
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	block2MovedPremineLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("20f2e24db7943eab4415f99e109698863b0fecca1cf9ffc500f38cefbbe29e"),
		block2MovedPremineAccount,
	})
	block2MovedPremineLeafNodeHash = crypto.Keccak256(block2MovedPremineLeafNode)
	block2x00080dBranchNode, _     = rlp.EncodeToBytes([]interface{}{
		block2MovedPremineLeafNodeHash,
		[]byte{},
		[]byte{},
		[]byte{},
		block2CoinbaseLeafNodeHash,
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
	})
	block2x00080dBranchNodeHash = crypto.Keccak256(block2x00080dBranchNode)
	block2x0008BranchNode, _    = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("def97a26f824fc3911cf7f8c41dfc9bc93cc36ae2248de22ecae01d6950b2dc9"),
		common.Hex2Bytes("234a575e2c5badab8de0f6515b6723195323a0562fbe1316255888637043f1c1"),
		common.Hex2Bytes("29659740af1c23306ee8f8294c71a5632ace8c80b1eb61cfdf7022f47ff52305"),
		common.Hex2Bytes("cf2681d23bb666d89dec8123bce9e626240a7e2ce7a1e8316b1ee88181c9471c"),
		common.Hex2Bytes("18d8de6967fe34b9fd411c74fecc45f8a737961791e70d8ece967bb07cf4d4dc"),
		common.Hex2Bytes("7cad60c7cbca8c79c2db5a8fc1baa9381484d43d6c37dfb97718c3a109d47dfc"),
		common.Hex2Bytes("2138f5a9062b750b6320e5fac5b134da90a9edbda06ef3e1ae64fb1366ca998c"),
		common.Hex2Bytes("532826502a9661fcae7c0f5d2a4c8cb287dfc521e828349543c5a461a9d591ed"),
		common.Hex2Bytes("30543537413dd086d4b1560f46b90e8da0f43de5584a138ab036d74e84657523"),
		common.Hex2Bytes("c98042928af640bfa1142aca895cd76e146332dce94ddad3426e74ed519ca1e0"),
		common.Hex2Bytes("43de3e62cc3148193899d018dff813c04c5b636ce95bd7e828416204292d9ff9"),
		[]byte{},
		common.Hex2Bytes("78d533b9182bb42f6c16e9ebd5734f0d280179ba1c9b6316c2c1df73f7dd8a54"),
		block2x00080dBranchNodeHash,
		common.Hex2Bytes("934b736b57a892aaa15a03c7e37746bb096313727135f9841cb64c263785cf81"),
		common.Hex2Bytes("38ce97150e90dfd7258901a0ddee72d8e30760a3d0419dbb80135c66588739a2"),
		[]byte{},
	})
	block2x0008BranchNodeHash = crypto.Keccak256(block2x0008BranchNode)
	block2x00BranchNode, _    = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("e45a9e85cab1b6eb18b30df2c6acc448bbac6a30d81646823b31223e16e5063e"),
		common.Hex2Bytes("33bd7171d556b981f6849064eb09412b24fedc0812127db936067043f53db1b9"),
		common.Hex2Bytes("ca56945f074da4f15587404593faf3a50d17ea0e21a418ad6ec99bdf4bf3f914"),
		common.Hex2Bytes("da23e9004f782df128eea1adff77952dc85f91b7f7ca4893aac5f21d24c3a1c9"),
		common.Hex2Bytes("ba5ec61fa780ee02af19db99677c37560fc4f0df5c278d9dfa2837f30f72bc6b"),
		common.Hex2Bytes("8310ad91625c2e3429a74066b7e2e0c958325e4e7fa3ec486b73b7c8300cfef7"),
		common.Hex2Bytes("732e5c103bf4d5adfef83773026809d9405539b67e93293a02342e83ad2fb766"),
		common.Hex2Bytes("30d14ff0c2aab57d1fbaf498ab14519b4e9d94f149a3dc15f0eec5adf8df25e1"),
		block2x0008BranchNodeHash,
		common.Hex2Bytes("5a43bd92e55aa78df60e70b6b53b6366c4080fd6a5bdd7b533b46aff4a75f6f2"),
		common.Hex2Bytes("a0c410aa59efe416b1213166fab680ce330bd46c3ebf877ff14609ee6a383600"),
		common.Hex2Bytes("2f41e918786e557293068b1eda9b3f9f86ed4e65a6a5363ee3262109f6e08b17"),
		common.Hex2Bytes("01f42a40f02f6f24bb97b09c4d3934e8b03be7cfbb902acc1c8fd67a7a5abace"),
		common.Hex2Bytes("0acbdce2787a6ea177209bd13bfc9d0779d7e2b5249e0211a2974164e14312f5"),
		common.Hex2Bytes("dadbe113e4132e0c0c3cd4867e0a2044d0e5a3d44b350677ed42fc9244d004d4"),
		common.Hex2Bytes("aa7441fefc17d76aedfcaf692fe71014b94c1547b6d129562b34fc5995ca0d1a"),
		[]byte{},
	})
	block2x00BranchNodeHash = crypto.Keccak256(block2x00BranchNode)
	block2RootBranchNode, _ = rlp.EncodeToBytes([]interface{}{
		block2x00BranchNodeHash,
		common.Hex2Bytes("babe369f6b12092f49181ae04ca173fb68d1a5456f18d20fa32cba73954052bd"),
		common.Hex2Bytes("473ecf8a7e36a829e75039a3b055e51b8332cbf03324ab4af2066bbd6fbf0021"),
		common.Hex2Bytes("bbda34753d7aa6c38e603f360244e8f59611921d9e1f128372fec0d586d4f9e0"),
		block1x04BranchNodeHash,
		common.Hex2Bytes("a5f3f2f7542148c973977c8a1e154c4300fec92f755f7846f1b734d3ab1d90e7"),
		common.Hex2Bytes("e823850f50bf72baae9d1733a36a444ab65d0a6faaba404f0583ce0ca4dad92d"),
		common.Hex2Bytes("f7a00cbe7d4b30b11faea3ae61b7f1f2b315b61d9f6bd68bfe587ad0eeceb721"),
		common.Hex2Bytes("7117ef9fc932f1a88e908eaead8565c19b5645dc9e5b1b6e841c5edbdfd71681"),
		common.Hex2Bytes("69eb2de283f32c11f859d7bcf93da23990d3e662935ed4d6b39ce3673ec84472"),
		common.Hex2Bytes("203d26456312bbc4da5cd293b75b840fc5045e493d6f904d180823ec22bfed8e"),
		common.Hex2Bytes("9287b5c21f2254af4e64fca76acc5cd87399c7f1ede818db4326c98ce2dc2208"),
		common.Hex2Bytes("6fc2d754e304c48ce6a517753c62b1a9c1d5925b89707486d7fc08919e0a94ec"),
		common.Hex2Bytes("7b1c54f15e299bd58bdfef9741538c7828b5d7d11a489f9c20d052b3471df475"),
		common.Hex2Bytes("51f9dd3739a927c89e357580a4c97b40234aa01ed3d5e0390dc982a7975880a0"),
		common.Hex2Bytes("89d613f26159af43616fd9455bb461f4869bfede26f2130835ed067a8b967bfb"),
		[]byte{},
	})

	// block3 data
	// path 060e0f
	blcok3CoinbaseBalance, _ = new(big.Int).SetString("5156250000000000000", 10)
	block3CoinbaseAccount, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  blcok3CoinbaseBalance,
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	block3CoinbaseLeafNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3a174f00e64521a535f35e67c1aa241951c791639b2f3d060f49c5d9fa8b9e"),
		block3CoinbaseAccount,
	})
	block3CoinbaseLeafNodeHash = crypto.Keccak256(block3CoinbaseLeafNode)
	// path 0c0e050703
	block3MovedPremineBalance1, _ = new(big.Int).SetString("3750000000000000000", 10)
	block3MovedPremineAccount1, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  block3MovedPremineBalance1,
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	block3MovedPremineLeafNode1, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("3ced93917e658d10e2d9009470dad72b63c898d173721194a12f2ae5e190"), // ce573ced93917e658d10e2d9009470dad72b63c898d173721194a12f2ae5e190
		block3MovedPremineAccount1,
	})
	block3MovedPremineLeafNodeHash1 = crypto.Keccak256(block3MovedPremineLeafNode1)
	// path 0c0e050708
	block3MovedPremineBalance2, _ = new(big.Int).SetString("1999944000000000000000", 10)
	block3MovedPremineAccount2, _ = rlp.EncodeToBytes(state.Account{
		Nonce:    0,
		Balance:  block3MovedPremineBalance2,
		CodeHash: testhelpers.NullCodeHash.Bytes(),
		Root:     testhelpers.EmptyContractRoot,
	})
	block3MovedPremineLeafNode2, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("33bc1e69eedf90f402e11f6862da14ed8e50156635a04d6393bbae154012"), // ce5783bc1e69eedf90f402e11f6862da14ed8e50156635a04d6393bbae154012
		block3MovedPremineAccount2,
	})
	block3MovedPremineLeafNodeHash2 = crypto.Keccak256(block3MovedPremineLeafNode2)

	block3x0c0e0507BranchNode, _ = rlp.EncodeToBytes([]interface{}{
		[]byte{},
		[]byte{},
		[]byte{},
		block3MovedPremineLeafNodeHash1,
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		block3MovedPremineLeafNodeHash2,
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
	})
	block3x0c0e0507BranchNodeHash = crypto.Keccak256(block3x0c0e0507BranchNode)

	block3x0c0e05BranchNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("452e3beb503b1d87ae7c672b98a8e3fd043a671405502562ae1043dc97151a50"),
		[]byte{},
		common.Hex2Bytes("2f5bb16f77086f67ce8c4258cb9061cb299e597b2ad4ad6d7ccc474d6d88e85e"),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		block3x0c0e0507BranchNodeHash,
		[]byte{},
		common.Hex2Bytes("44623e5a9319f83870db0ea4611a25fca1e1da3eeea2be4a091dfc15ab45689e"),
		common.Hex2Bytes("b41e047a97f44fa4cb8146467b88c8f4705811029d9e170abb0aba7d0af9f0da"),
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
	})
	block3x0c0e05BranchNodeHash = crypto.Keccak256(block3x0c0e05BranchNode)

	block3x060eBranchNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("94d77c7c30b88829c9989948b206cda5e532b38b49534261c517aebf4a3e6fdb"),
		common.Hex2Bytes("a5cf57a50da8204964e834a12a53f9bed7afc9b700a4a81b440122d60c7603a7"),
		[]byte{},
		common.Hex2Bytes("3730ec0571f34b6c3b178dc26ccb31a3f50c29da9b1921e41b9477ddab41b0fe"),
		[]byte{},
		common.Hex2Bytes("543952bb9566c2018cf8d7b90d6a7903cdfff3d79ac36189be5322de42fc3fc0"),
		[]byte{},
		common.Hex2Bytes("c4a49b66f0bcc08531e50cdea5577a281d111fa542eaefd9a9aead8febb0735e"),
		common.Hex2Bytes("362ad58916c71463b98c079649fc486c5f082c4f548bd4ab501515f0c5641cb4"),
		common.Hex2Bytes("36aae109f6f55f0bd05eb05bb365af2332dfe5f06d3d17903e88534c319eb709"),
		common.Hex2Bytes("430dcfc5cc49a6b490dd54138920e8f94e427239c2bccc14705cfd4ff6cc4383"),
		common.Hex2Bytes("73ed77563dfed2fdb38900b474db88b2270f449167e0d877fda9e2229f119fe8"),
		common.Hex2Bytes("5dfe06013f2a41f1779194ceb07769d019f518b2a694a82fa1661e60fd973eaa"),
		common.Hex2Bytes("80bdfd85fbb6b45850bad6e34136aaa1b04711e47469fa2f0d19eca52089efb5"),
		[]byte{},
		block3CoinbaseLeafNodeHash,
		[]byte{},
	})
	block3x060eBranchNodeHash = crypto.Keccak256(block3x060eBranchNode)

	block3x0c0eBranchNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("70647f11b2b995d718f9e8aceb44c8839e0055641930d216fa6090280a9d63d5"),
		common.Hex2Bytes("fdfb17cd2fba2a14219981cb7886a1977cd85dbef5c767c562f4a5f547febff0"),
		common.Hex2Bytes("ff87313253ec6f860142b7bf62efb4cb07ea668c57aa90cbe9ef22b72fee15c7"),
		common.Hex2Bytes("3a77b3c26a54ad37bdf4e19c1bce93493ec0f79d9ad90190b70bc840b54918e1"),
		common.Hex2Bytes("af1b3b14324561b68f2e24dbcc28673ab35ce3fd0230fe2bc86b3d1931745195"),
		block3x0c0e05BranchNodeHash,
		common.Hex2Bytes("647dcbfe6aabcd9d219ff40422af4326bfc1ec66703195a78eb48618ddef248d"),
		common.Hex2Bytes("2d2bf06159cc8928283c3419a03f08ea34c493a9d002a0ec76d5c429508ccaf4"),
		common.Hex2Bytes("d7147251b3f48f25e1e4c6d8f83a00b1eca66e99a4ea0d238942ce72d0ba6414"),
		common.Hex2Bytes("cb859370869967594fb29f4e2904413310146733d7fcbd11407f3e47626e0e34"),
		common.Hex2Bytes("b93ab9b0bd83963860fbe0b7d543879cfde756ea1618d2a40d85483058cc5a26"),
		common.Hex2Bytes("45aee096499d209931457ce251c5c7e5543f22524f67785ff8f0f3f02588b0ed"),
		[]byte{},
		common.Hex2Bytes("aa2ae9379797c5066bba646108074ae8677e82c923d584b6d1c1268ca3708c5c"),
		common.Hex2Bytes("e6eb055f0d8e194c083471479a3de87fa0f90c0f4aaa518416ec1e469ec32e3a"),
		common.Hex2Bytes("0cc9c50fc7eba162fb17f2e04e3599c13abbf210d9781864d0edec401ecaebba"),
		[]byte{},
	})
	block3x0c0eBranchNodeHash = crypto.Keccak256(block3x0c0eBranchNode)

	block3x06BranchNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("68f7ff8c074d6e4cccd55b5b1c2116a6dd7047d4332090e6db8839362991b0ae"),
		common.Hex2Bytes("c446eb4377c750701374c56e50759e6ba68b7adf4d543e718c8b28a99ae3b6ad"),
		common.Hex2Bytes("ef2c49ec64cb65eae0d99684e74c8af2bd0206c9a0214d9d3eddf0881dd8412a"),
		common.Hex2Bytes("7096c4cc7e8125f0b142d8644ad681f8a8142e210c806f33f3f7004f0e9d6002"),
		common.Hex2Bytes("bc9a8ae647b234cd6607b6b0245e3b3d5ec4f7ea006e7eda1f92d02f0ea91116"),
		common.Hex2Bytes("a87720deb92ff2f899e809befab9970a61c86148c4fa09d04b77505ee4a5bda5"),
		common.Hex2Bytes("2460e5b6ded7c0001de29c15db124614432fef6486370cc9970f63b0d95fd5e2"),
		common.Hex2Bytes("ed1c447d4a32bc31e9e32259dc63da10df91231e786332e3df122b301b1f8fc3"),
		common.Hex2Bytes("0d27dfc201d995c2323b792860dbca087da7cc56d1698c39b7c4b9277729c5ca"),
		common.Hex2Bytes("f6d2be168d9c17643c9ea80c29322b364604cdfd36eef40123d83fad364e43fa"),
		common.Hex2Bytes("004bf1c30a5730f464de1a0ba4ac5b5618df66d6106073d08742166e33a7eeb5"),
		common.Hex2Bytes("7298d019a57a1b04ac31ed874d654ba0d3c249704c5d9efa1d08959fc89e0779"),
		common.Hex2Bytes("fb3d50b7af6f839e371ff8ebd0322e94e6b6fb7888416737f88cf55bcf5859ec"),
		common.Hex2Bytes("4e7a2618fa1fc560a73c24839657adf7e48d600ecfb12333678115936597a913"),
		block3x060eBranchNodeHash,
		common.Hex2Bytes("1909706c5db040f54c19f4050659ad484982145b02474653917de379f15ebb36"),
		[]byte{},
	})
	block3x06BranchNodeHash = crypto.Keccak256(block3x06BranchNode)

	block3x0cBranchNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("dae48f5b47930c28bb116fbd55e52cd47242c71bf55373b55eb2805ee2e4a929"),
		common.Hex2Bytes("0f1f37f337ec800e2e5974e2e7355f10f1a4832b39b846d916c3597a460e0676"),
		common.Hex2Bytes("da8f627bb8fbeead17b318e0a8e4f528db310f591bb6ab2deda4a9f7ca902ab5"),
		common.Hex2Bytes("971c662648d58295d0d0aa4b8055588da0037619951217c22052802549d94a2f"),
		common.Hex2Bytes("ccc701efe4b3413fd6a61a6c9f40e955af774649a8d9fd212d046a5a39ddbb67"),
		common.Hex2Bytes("d607cdb32e2bd635ee7f2f9e07bc94ddbd09b10ec0901b66628e15667aec570b"),
		common.Hex2Bytes("5b89203dc940e6fa70ec19ad4e01d01849d3a5baa0a8f9c0525256ed490b159f"),
		common.Hex2Bytes("b84227d48df68aecc772939a59afa9e1a4ab578f7b698bdb1289e29b6044668e"),
		common.Hex2Bytes("fd1c992070b94ace57e48cbf6511a16aa770c645f9f5efba87bbe59d0a042913"),
		common.Hex2Bytes("e16a7ccea6748ae90de92f8aef3b3dc248a557b9ac4e296934313f24f7fced5f"),
		common.Hex2Bytes("42373cf4a00630d94de90d0a23b8f38ced6b0f7cb818b8925fee8f0c2a28a25a"),
		common.Hex2Bytes("5f89d2161c1741ff428864f7889866484cef622de5023a46e795dfdec336319f"),
		common.Hex2Bytes("7597a017664526c8c795ce1da27b8b72455c49657113e0455552dbc068c5ba31"),
		common.Hex2Bytes("d5be9089012fda2c585a1b961e988ea5efcd3a06988e150a8682091f694b37c5"),
		block3x0c0eBranchNodeHash,
		common.Hex2Bytes("49bf6e8df0acafd0eff86defeeb305568e44d52d2235cf340ae15c6034e2b241"),
		[]byte{},
	})
	block3x0cBranchNodeHash = crypto.Keccak256(block3x0cBranchNode)

	block3RootBranchNode, _ = rlp.EncodeToBytes([]interface{}{
		common.Hex2Bytes("f646da473c426e79f1c796b00d4873f47de1dbe1c9d19d63993a05eeb8b4041d"),
		common.Hex2Bytes("babe369f6b12092f49181ae04ca173fb68d1a5456f18d20fa32cba73954052bd"),
		common.Hex2Bytes("473ecf8a7e36a829e75039a3b055e51b8332cbf03324ab4af2066bbd6fbf0021"),
		common.Hex2Bytes("bbda34753d7aa6c38e603f360244e8f59611921d9e1f128372fec0d586d4f9e0"),
		common.Hex2Bytes("d9cff5d5f2418afd16a4da5c221fdc8bd47520c5927922f69a68177b64da6ac0"),
		common.Hex2Bytes("a5f3f2f7542148c973977c8a1e154c4300fec92f755f7846f1b734d3ab1d90e7"),
		block3x06BranchNodeHash,
		common.Hex2Bytes("f7a00cbe7d4b30b11faea3ae61b7f1f2b315b61d9f6bd68bfe587ad0eeceb721"),
		common.Hex2Bytes("7117ef9fc932f1a88e908eaead8565c19b5645dc9e5b1b6e841c5edbdfd71681"),
		common.Hex2Bytes("69eb2de283f32c11f859d7bcf93da23990d3e662935ed4d6b39ce3673ec84472"),
		common.Hex2Bytes("203d26456312bbc4da5cd293b75b840fc5045e493d6f904d180823ec22bfed8e"),
		common.Hex2Bytes("9287b5c21f2254af4e64fca76acc5cd87399c7f1ede818db4326c98ce2dc2208"),
		block3x0cBranchNodeHash,
		common.Hex2Bytes("7b1c54f15e299bd58bdfef9741538c7828b5d7d11a489f9c20d052b3471df475"),
		common.Hex2Bytes("51f9dd3739a927c89e357580a4c97b40234aa01ed3d5e0390dc982a7975880a0"),
		common.Hex2Bytes("89d613f26159af43616fd9455bb461f4869bfede26f2130835ed067a8b967bfb"),
		[]byte{},
	})
)

func init() {
	db = rawdb.NewMemoryDatabase()
	genesisBlock = core.DefaultGenesisBlock().MustCommit(db)
	genBy, err := rlp.EncodeToBytes(genesisBlock)
	if err != nil {
		log.Fatal(err)
	}
	var block0RLP []byte
	block0, block0RLP, err = loadBlockFromRLPFile("./block0_rlp")
	if err != nil {
		log.Fatal(err)
	}
	if !bytes.Equal(genBy, block0RLP) {
		log.Fatal("mainnet genesis blocks do not match")
	}
	block1, _, err = loadBlockFromRLPFile("./block1_rlp")
	if err != nil {
		log.Fatal(err)
	}
	block1CoinbaseAddr = block1.Coinbase()
	block1CoinbaseHash = crypto.Keccak256Hash(block1CoinbaseAddr.Bytes())
	block2, _, err = loadBlockFromRLPFile("./block2_rlp")
	if err != nil {
		log.Fatal(err)
	}
	block2CoinbaseAddr = block2.Coinbase()
	block2CoinbaseHash = crypto.Keccak256Hash(block2CoinbaseAddr.Bytes())
	block3, _, err = loadBlockFromRLPFile("./block3_rlp")
	if err != nil {
		log.Fatal(err)
	}
	block3CoinbaseAddr = block3.Coinbase()
	block3CoinbaseHash = crypto.Keccak256Hash(block3CoinbaseAddr.Bytes())
}

func loadBlockFromRLPFile(filename string) (*types.Block, []byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	blockRLP, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, nil, err
	}
	block := new(types.Block)
	return block, blockRLP, rlp.DecodeBytes(blockRLP, block)
}

func TestBuilderOnMainnetBlocks(t *testing.T) {
	chain, _ := core.NewBlockChain(db, nil, params.MainnetChainConfig, ethash.NewFaker(), vm.Config{}, nil)
	_, err := chain.InsertChain([]*types.Block{block1, block2, block3})
	if err != nil {
		t.Error(err)
	}
	params := statediff.Params{
		IntermediateStateNodes: true,
	}
	builder = statediff.NewBuilder(chain.StateCache())

	var tests = []struct {
		name              string
		startingArguments statediff.Args
		expected          *statediff.StateObject
	}{
		// note that block0 (genesis) has over 1000 nodes due to the pre-allocation for the crowd-sale
		// it is not feasible to write a unit test of that size at this time
		{
			"testBlock1",
			//10000 transferred from testBankAddress to account1Addr
			statediff.Args{
				OldStateRoot: block0.Root(),
				NewStateRoot: block1.Root(),
				BlockNumber:  block1.Number(),
				BlockHash:    block1.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block1.Number(),
				BlockHash:   block1.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block1RootBranchNode,
					},
					{
						Path:         []byte{'\x04'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block1x04BranchNode,
					},
					{
						Path:         []byte{'\x04', '\x0b'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block1x040bBranchNode,
					},
					{
						Path:         []byte{'\x04', '\x0b', '\x0e'},
						NodeType:     sdtypes.Leaf,
						LeafKey:      block1CoinbaseHash.Bytes(),
						NodeValue:    block1CoinbaseLeafNode,
						StorageNodes: emptyStorage,
					},
				},
			},
		},
		{
			"testBlock2",
			// 1000 transferred from testBankAddress to account1Addr
			// 1000 transferred from account1Addr to account2Addr
			// account1addr creates a new contract
			statediff.Args{
				OldStateRoot: block1.Root(),
				NewStateRoot: block2.Root(),
				BlockNumber:  block2.Number(),
				BlockHash:    block2.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block2.Number(),
				BlockHash:   block2.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block2RootBranchNode,
					},
					{
						Path:         []byte{'\x00'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block2x00BranchNode,
					},
					{
						Path:         []byte{'\x00', '\x08'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block2x0008BranchNode,
					},
					{
						Path:         []byte{'\x00', '\x08', '\x0d'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block2x00080dBranchNode,
					},
					// this new leaf at x00 x08 x0d x00 was "created" when a premine account (leaf) was moved from path x00 x08 x0d
					// this occurred because of the creation of the new coinbase receiving account (leaf) at x00 x08 x0d x04
					// which necessitates we create a branch at x00 x08 x0d (as shown in the below UpdateAccounts)
					{
						Path:         []byte{'\x00', '\x08', '\x0d', '\x00'},
						NodeType:     sdtypes.Leaf,
						StorageNodes: emptyStorage,
						LeafKey:      common.HexToHash("08d0f2e24db7943eab4415f99e109698863b0fecca1cf9ffc500f38cefbbe29e").Bytes(),
						NodeValue:    block2MovedPremineLeafNode,
					},
					{
						Path:         []byte{'\x00', '\x08', '\x0d', '\x04'},
						NodeType:     sdtypes.Leaf,
						StorageNodes: emptyStorage,
						LeafKey:      block2CoinbaseHash.Bytes(),
						NodeValue:    block2CoinbaseLeafNode,
					},
				},
			},
		},
		{
			"testBlock3",
			//the contract's storage is changed
			//and the block is mined by account 2
			statediff.Args{
				OldStateRoot: block2.Root(),
				NewStateRoot: block3.Root(),
				BlockNumber:  block3.Number(),
				BlockHash:    block3.Hash(),
			},
			&statediff.StateObject{
				BlockNumber: block3.Number(),
				BlockHash:   block3.Hash(),
				Nodes: []sdtypes.StateNode{
					{
						Path:         []byte{},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block3RootBranchNode,
					},
					{
						Path:         []byte{'\x06'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block3x06BranchNode,
					},
					{
						Path:         []byte{'\x06', '\x0e'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block3x060eBranchNode,
					},
					{
						Path:         []byte{'\x0c'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block3x0cBranchNode,
					},
					{
						Path:         []byte{'\x0c', '\x0e'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block3x0c0eBranchNode,
					},
					{
						Path:         []byte{'\x0c', '\x0e', '\x05'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block3x0c0e05BranchNode,
					},
					{
						Path:         []byte{'\x0c', '\x0e', '\x05', '\x07'},
						NodeType:     sdtypes.Branch,
						StorageNodes: emptyStorage,
						NodeValue:    block3x0c0e0507BranchNode,
					},
					{ // How was this account created???
						Path:         []byte{'\x0c', '\x0e', '\x05', '\x07', '\x03'},
						NodeType:     sdtypes.Leaf,
						StorageNodes: emptyStorage,
						LeafKey:      common.HexToHash("ce573ced93917e658d10e2d9009470dad72b63c898d173721194a12f2ae5e190").Bytes(),
						NodeValue:    block3MovedPremineLeafNode1,
					},
					{ // This account (leaf) used to be at 0c 0e 05 07, likely moves because of the new account above
						Path:         []byte{'\x0c', '\x0e', '\x05', '\x07', '\x08'},
						NodeType:     sdtypes.Leaf,
						StorageNodes: emptyStorage,
						LeafKey:      common.HexToHash("ce5783bc1e69eedf90f402e11f6862da14ed8e50156635a04d6393bbae154012").Bytes(),
						NodeValue:    block3MovedPremineLeafNode2,
					},
					{ // this is the new account created due to the coinbase mining a block, it's creation shouldn't affect 0x 0e 05 07
						Path:         []byte{'\x06', '\x0e', '\x0f'},
						NodeType:     sdtypes.Leaf,
						StorageNodes: emptyStorage,
						LeafKey:      block3CoinbaseHash.Bytes(),
						NodeValue:    block3CoinbaseLeafNode,
					},
				},
			},
		},
	}

	for _, test := range tests {
		diff, err := builder.BuildStateDiffObject(test.startingArguments, params)
		if err != nil {
			t.Error(err)
		}
		receivedStateDiffRlp, err := rlp.EncodeToBytes(diff)
		if err != nil {
			t.Error(err)
		}
		expectedStateDiffRlp, err := rlp.EncodeToBytes(test.expected)
		if err != nil {
			t.Error(err)
		}
		sort.Slice(receivedStateDiffRlp, func(i, j int) bool { return receivedStateDiffRlp[i] < receivedStateDiffRlp[j] })
		sort.Slice(expectedStateDiffRlp, func(i, j int) bool { return expectedStateDiffRlp[i] < expectedStateDiffRlp[j] })
		if !bytes.Equal(receivedStateDiffRlp, expectedStateDiffRlp) {
			t.Logf("Test failed: %s", test.name)
			t.Errorf("actual state diff: %+v\nexpected state diff: %+v", diff, test.expected)
		}
	}
}
