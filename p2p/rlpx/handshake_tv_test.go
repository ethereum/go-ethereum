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

package rlpx

import "crypto/ecdsa"

type handshakeTest struct {
	// Inputs
	initiator, recipient                         *Config
	initiatorEphemeralKey, recipientEphemeralKey *ecdsa.PrivateKey
	initiatorNonce, recipientNonce               []byte

	// Derived Values: These must match exactly in all Test*TV
	// functions and do not depend on any random values apart from the
	// ones above.
	negotiatedVersion                               uint
	initiatorEgressSecrets, initiatorIngressSecrets secrets

	// Encrypted Packets: We can't check them directly because both
	// encryption and signing introduce random values.
	// TestHandshakePacketsRecipientTV and
	// TestHandshakePacketsInitiatorTV check that each 'side' accepts
	// the other packet and computes the right secrets.
	encAuth, encAuthResp []byte

	// Digests of the empty string created with each MAC hash. These
	// are checked TestHandshakeDeriveMacTV with the packets above
	// because RLPx V4 includes the ciphertext in the hash.
	initiatorIngressMacDigest, initiatorEgressMacDigest []byte
}

var handshakeTV = []handshakeTest{
	// initiator V5, recipient V5
	{
		initiator: &Config{
			Key:     hexkey("5e173f6ac3c669587538e7727cf19b782a4f2fda07c1eaa662c593e5e85e3051"),
			ForceV4: false,
		},
		recipient: &Config{
			Key:     hexkey("c45f950382d542169ea207959ee0220ec1491755abe405cd7498d6b16adb6df8"),
			ForceV4: false,
		},
		initiatorEphemeralKey: hexkey("19c2185f4f40634926ebed3af09070ca9e029f2edd5fae6253074896205f5f6c"),
		recipientEphemeralKey: hexkey("d25688cf0ab10afa1a0e2dba7853ed5f1e5bf1c631757ed4e103b593ff3f5620"),
		initiatorNonce:        hexb("cd26fecb93657d1cd9e9eaf4f8be720b56dd1d39f190c4e10000000000000005"),
		recipientNonce:        hexb("f37ec61d84cea03dcc5e8385db93248584e8af4b4d1c832d0000000000000005"),

		encAuth: hexb(`
			04f0849817e9483c39b1eead6f5a0dfb09cbc4c43151172c2549c5b07c9364f7
			853cbae79e7d2a3b9a79d042ddbdf1d95db1f8c7428989123afb02fcad93bfac
			413ad9a9f2bf9b2a621a09fe804229f31f9a71ae6776f10bc8d13bb372fa4af5
			9a39fd0ae546cb5fce9fd55d59b13e6cbcc234421ab089f1f08932d4622e460c
			b0f63b3375b8388e25f84db55a415c764386e00bc19da675baf8e643f48d14c9
			89432062ed3495943bb6b3f46e8a5011edc3648a0396bccbaa0fe164bd2b8919
			df542da5fd34f24e2d5b84082a9be5fce2e17625d90078eafa8d3125314553e7
			008dedd9fd5d9d082811a08581d596f7605eba7500aafd2f3c5c8b8cfdce2ac3
			417559c3d52d6d326a832f0c43077d0697c06db24a5a28c1d67033ecda5d4ff8
			74831427bcda3ef422b8cc9f34b1eabf39607a
		`),
		encAuthResp: hexb(`
			0496117783823744f0f58cd952ed34a1866e4ade1a2d66cdfd041f877c4e4216
			d45645ad1dedee1d54d5a767d87231fbadbc6dcb2b48c75b3ab46cb18b224a7f
			cd3d9619e03d24813aaa0cc37adb32aa2fa7fbc13aa1fbc01d24d402715fe213
			62a457a986ec649983f0ff81f5f207799849bce8061dab17b491ac7f0090c426
			f7e63c31f11917e8e33c65d74bd2094435e73ffab1dfbaff368de079244d4ebd
			7f8b542f7081756a1e94b4ed26fb1c3bddabbd642064a15ad597a4f63894ea31
			13cab7533eec3b8ae163f8ebd61d7bac71e4
		`),

		negotiatedVersion: 5,
		initiatorIngressSecrets: secrets{
			encKey: hexb("5d268dbeede1c3ce4e7cd1f900543f671467284d53c6f6fd6b284789652bd1f6"),
			encIV:  hexb("e5703e8952a6eafcdb2c1940c7615843"),
			macKey: hexb("09726cd8b6414cb1f5858b0339badeeed377a48cbe5f3d28a4f74ae41e610c4d"),
		},
		initiatorEgressSecrets: secrets{
			encKey: hexb("40bfcb0da6d30512f57187f61b4816fcdc9aaaf107184e29467fe6f6ccefe4a7"),
			encIV:  hexb("0e906055c0ca86940626d0fde4f3a9c2"),
			macKey: hexb("c676534122bd3a555ca8f2d924b63222b1b5b5efccb7b37a52795c1c49450fdc"),
		},
		initiatorIngressMacDigest: hexb("de72d7161bc7a9ddda4a70a48d08eda55d6fc4d90ef80a4b6645f81d6373e66b"),
		initiatorEgressMacDigest:  hexb("47bb77bff168de73c7ae34473f4d085abdf97cce7e01cab6ee3a4f69a021645f"),
	},

	// initiator V5, recipient V4
	{
		initiator: &Config{
			Key:     hexkey("5e173f6ac3c669587538e7727cf19b782a4f2fda07c1eaa662c593e5e85e3051"),
			ForceV4: false,
		},
		recipient: &Config{
			Key:     hexkey("c45f950382d542169ea207959ee0220ec1491755abe405cd7498d6b16adb6df8"),
			ForceV4: true,
		},
		initiatorEphemeralKey: hexkey("19c2185f4f40634926ebed3af09070ca9e029f2edd5fae6253074896205f5f6c"),
		recipientEphemeralKey: hexkey("d25688cf0ab10afa1a0e2dba7853ed5f1e5bf1c631757ed4e103b593ff3f5620"),
		initiatorNonce:        hexb("cd26fecb93657d1cd9e9eaf4f8be720b56dd1d39f190c4e10000000000000005"),
		recipientNonce:        hexb("f37ec61d84cea03dcc5e8385db93248584e8af4b4d1c832d8c7453c0089687a7"),

		encAuth: hexb(`
			04e96a95f95ca7188aedd8abcf58f5f99efc7efed406923084655ce9b197cfe1
			0921159e105d1cf335e2f4755813598b868784130f4f0ae0bf1776b70d9e3061
			9205869af0963e0503133dabd7e5ed8ddf531ac3ac4d243f131252b8fd5a1638
			7296236b9b6b23432dcf02064284522690b8a07cf7e8cd9935409693203f31b9
			4fe5b2fba7e49a90711597e13ab318230a5fa3c89965569c9da21aa20686df25
			a39bd3393c87ae0f9d11bd6b270480f3544be771fdca8fd2cebb161cc508ee38
			f1eedb793d75f7e081fdf837be699fee2af1f00e2e8924d2ec5c64dabe445d0f
			14caae3c20583252fa8adace052a5f5832ebb957d3324cf12d27232934193806
			a4bfdf9adc3a0921e0bdc7c7457cf35b5d99e729d7e0fd9aa09ab1217ff5ceca
			768fb1fc636499eecab58bb01f46eea652ccbd
		`),
		encAuthResp: hexb(`
			04d0f8e56113ee9402ab4fed101fe03842f265e13e9bb76af2ac1ffba11d8892
			524e59f1906eb2e6e35f774ccb3449d2f5084b96063e668fb73d90a94b0114dc
			c14364a087f270adc65421741d87c492eafe1ca3b86a76313d026564e1abbb48
			6d03727c9baf8d0314af54296ea829aa086174a7836113f1dd420750af98f0b8
			802940ab16af05421d6b812054b285fdf1ae82ae0c08f1dbee3d60691979e8bd
			0b31599ac47138ba24d404699ae4558fec8bb94f120e63362e4b94a50894021e
			70e69101820018472823a48bc0d61c617c21
		`),

		negotiatedVersion: 4,
		initiatorIngressSecrets: secrets{
			encKey: hexb("3ca5db8d7d13af7bb3763fee9cef628925a4abda5961d7392fae731c02278377"),
			encIV:  hexb("00000000000000000000000000000000"),
			macKey: hexb("cb2cd684639c1b64b80687b977c4140bea8c953a1f3975aca6f1589a850879ba"),
		},
		initiatorEgressSecrets: secrets{
			encKey: hexb("3ca5db8d7d13af7bb3763fee9cef628925a4abda5961d7392fae731c02278377"),
			encIV:  hexb("00000000000000000000000000000000"),
			macKey: hexb("cb2cd684639c1b64b80687b977c4140bea8c953a1f3975aca6f1589a850879ba"),
		},
		initiatorIngressMacDigest: hexb("d835ba6c6ac42d4c686a2e3cee6b2ee0190a7da79d6275f2b0b4bdc71fb66709"),
		initiatorEgressMacDigest:  hexb("1901e950288f010d005ccafede47d1dc177c442605a9702fcd6c5a0e717dc130"),
	},

	// initiator V4, recipient V5
	{
		initiator: &Config{
			Key:     hexkey("5e173f6ac3c669587538e7727cf19b782a4f2fda07c1eaa662c593e5e85e3051"),
			ForceV4: true,
		},
		recipient: &Config{
			Key:     hexkey("c45f950382d542169ea207959ee0220ec1491755abe405cd7498d6b16adb6df8"),
			ForceV4: false,
		},
		initiatorEphemeralKey: hexkey("19c2185f4f40634926ebed3af09070ca9e029f2edd5fae6253074896205f5f6c"),
		recipientEphemeralKey: hexkey("d25688cf0ab10afa1a0e2dba7853ed5f1e5bf1c631757ed4e103b593ff3f5620"),
		initiatorNonce:        hexb("cd26fecb93657d1cd9e9eaf4f8be720b56dd1d39f190c4e1c6b7ec66f077bb11"),
		recipientNonce:        hexb("f37ec61d84cea03dcc5e8385db93248584e8af4b4d1c832d0000000000000005"),

		encAuth: hexb(`
			0461109a208261e0bedca9b3d474e991409fe0f5be7fd757d33c3a2934be7819
			269509b86229f36677f23aebe239c21e0c2b244811f9ee0d5370bc4cef1510a0
			77e1d4d295a6219a9393d7493a52b9e98ccca17a196b43efb30cdfaf2ea99fff
			bd29c44b5b417924ad3afad6c436e85a0024e24f55f27bb8f86735a76586093f
			4d496348363cb2fb9905e62786c18030a8b4e3fe4a621439ae1c10598a09f9f5
			e61315cea0cd09fb0438d9c76b1d516e183a8df2bdc2d7e3a51f4011a990999c
			b5fb737b9dd16181f44253b313811286004efe9298d4ff49eefd28dc095ed362
			8b56551f052c4d94c0c0108d08656a0201eb29d66702acb06e2894fab08a684a
			394258171fc3f099c4f58a075ecde74c8084731639e19b194ff9eef6824a1330
			59bd884d81ef14541fd9475b9f3d8bcb613eb6
		`),
		encAuthResp: hexb(`
			04c77decb1d4abe500fb924a5972d495b6ca6782122e1e795d8282575302ed32
			85b0d4bae57ac07c2f455e67cf73d2c77b9dcd295252ca146a65ec3a7e9d6336
			a1ed212843cafac42831c5a785fe7fc6e18a1ce7ae3d9603c439cdd991ec2f7a
			5197838efbd8c0ad68e31559c8a711ca3368bb6f4ed6de53db86df7a56eb2897
			bbb251a2c2e86af3198e87bd98af9f3d96ae7f0777656a3a0c9dec4718f49377
			0f78b6fecc51f398c0e36e696da3b482584c00581b6b7e596b71580777972bf4
			e158ce46acf49c893a509c00415996948df2
		`),

		negotiatedVersion: 4,
		initiatorIngressSecrets: secrets{
			encKey: hexb("fc8e46d37d756d53af5f6cebb35d94118bf305fe1c73fc9e672350cbc1dedd75"),
			encIV:  hexb("00000000000000000000000000000000"),
			macKey: hexb("274693e239751ff505004ed5ab680fd80823d49a12139554bffce549b32d048c"),
		},
		initiatorEgressSecrets: secrets{
			encKey: hexb("fc8e46d37d756d53af5f6cebb35d94118bf305fe1c73fc9e672350cbc1dedd75"),
			encIV:  hexb("00000000000000000000000000000000"),
			macKey: hexb("274693e239751ff505004ed5ab680fd80823d49a12139554bffce549b32d048c"),
		},
		initiatorIngressMacDigest: hexb("395908f0f3da7e588aad3e6ec04fab504f0a65664bf8fe135b67ebaf4cc41daf"),
		initiatorEgressMacDigest:  hexb("09cc2e837d5cf048f41ea39dccc4b07814a125dc5d47e416ae13dac8d4523239"),
	},

	// old V4 test vector from https://gist.github.com/fjl/3a78780d17c755d22df2
	{
		initiator: &Config{
			Key:     hexkey("5e173f6ac3c669587538e7727cf19b782a4f2fda07c1eaa662c593e5e85e3051"),
			ForceV4: true,
		},
		recipient: &Config{
			Key:     hexkey("c45f950382d542169ea207959ee0220ec1491755abe405cd7498d6b16adb6df8"),
			ForceV4: true,
		},
		initiatorEphemeralKey: hexkey("19c2185f4f40634926ebed3af09070ca9e029f2edd5fae6253074896205f5f6c"),
		recipientEphemeralKey: hexkey("d25688cf0ab10afa1a0e2dba7853ed5f1e5bf1c631757ed4e103b593ff3f5620"),
		initiatorNonce:        hexb("cd26fecb93657d1cd9e9eaf4f8be720b56dd1d39f190c4e1c6b7ec66f077bb11"),
		recipientNonce:        hexb("f37ec61d84cea03dcc5e8385db93248584e8af4b4d1c832d8c7453c0089687a7"),

		encAuth: hexb(`
			04a0274c5951e32132e7f088c9bdfdc76c9d91f0dc6078e848f8e3361193dbdc
			43b94351ea3d89e4ff33ddcefbc80070498824857f499656c4f79bbd97b6c51a
			514251d69fd1785ef8764bd1d262a883f780964cce6a14ff206daf1206aa073a
			2d35ce2697ebf3514225bef186631b2fd2316a4b7bcdefec8d75a1025ba2c540
			4a34e7795e1dd4bc01c6113ece07b0df13b69d3ba654a36e35e69ff9d482d88d
			2f0228e7d96fe11dccbb465a1831c7d4ad3a026924b182fc2bdfe016a6944312
			021da5cc459713b13b86a686cf34d6fe6615020e4acf26bf0d5b7579ba813e77
			23eb95b3cef9942f01a58bd61baee7c9bdd438956b426a4ffe238e61746a8c93
			d5e10680617c82e48d706ac4953f5e1c4c4f7d013c87d34a06626f498f34576d
			c017fdd3d581e83cfd26cf125b6d2bda1f1d56
		`),
		encAuthResp: hexb(`
			049934a7b2d7f9af8fd9db941d9da281ac9381b5740e1f64f7092f3588d4f87f
			5ce55191a6653e5e80c1c5dd538169aa123e70dc6ffc5af1827e546c0e958e42
			dad355bcc1fcb9cdf2cf47ff524d2ad98cbf275e661bf4cf00960e74b5956b79
			9771334f426df007350b46049adb21a6e78ab1408d5e6ccde6fb5e69f0f4c92b
			b9c725c02f99fa72b9cdc8dd53cff089e0e73317f61cc5abf6152513cb7d833f
			09d2851603919bf0fbe44d79a09245c6e8338eb502083dc84b846f2fee1cc310
			d2cc8b1b9334728f97220bb799376233e113
		`),

		negotiatedVersion: 4,
		initiatorIngressSecrets: secrets{
			encKey: hexb("c0458fa97a5230830e05f4f20b7c755c1d4e54b1ce5cf43260bb191eef4e418d"),
			encIV:  hexb("00000000000000000000000000000000"),
			macKey: hexb("48c938884d5067a1598272fcddaa4b833cd5e7d92e8228c0ecdfabbe68aef7f1"),
		},
		initiatorEgressSecrets: secrets{
			encKey: hexb("c0458fa97a5230830e05f4f20b7c755c1d4e54b1ce5cf43260bb191eef4e418d"),
			encIV:  hexb("00000000000000000000000000000000"),
			macKey: hexb("48c938884d5067a1598272fcddaa4b833cd5e7d92e8228c0ecdfabbe68aef7f1"),
		},
		initiatorIngressMacDigest: hexb("75823d96e23136c89666ee025fb21a432be906512b3dd4a3049e898adb433847"),
		initiatorEgressMacDigest:  hexb("09771e93b1a6109e97074cbe2d2b0cf3d3878efafe68f53c41bb60c0ec49097e"),
	},
}
