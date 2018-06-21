// Copyright 2018 The go-ethereum Authors
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

package encryption

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
)

var expectedTransformedHex = "470c6c67ba1820d5cb4c23ccef22aa2417a323dc97e5f5dced930d74f2932fd178df80ddf129f4f8a4ccec0225c1c2e6765cbbd92bbad8413c50d93d53f7b2fec975d6f29468eccdaf6458f7a3306a7bc207211ea7f9ee5e6951ce6874aef09eb7ff9bed0aa00920b4dcc105e5f1f8dfbdb0564751311d6ceaca1e4e6d988097582638106404c03cfaef8db0e46674d9e8192c1b62d1cd952389cab3a8dee9329fdb059bafd9bae3df3a6b5a9a10961a0333016c99ee3d65cf18ea8ea7b4c386d59dcbf317460d517d5b55504b5992ffb9c3d7c49ebe1fe3ab7b00ad84b1d76d95f8165141260e6980b2ffe8a0000c36dd77ace4c7a781887f831f740d92d8b4848f1b9e237877b988b5a3e23d7b07b2c8eda0ea9fa748ab10bb6bfd7447b66f44e528d24eff31defdecddaf2bb5e6b8e2aa3d2f1a83de44e0ee3789b75dadba1375b7f3a8f7f1eb2d9b78ad1844d425fb76ea6ba45eabe88f1e062d7552d8c7d44c9c66c2753c41892cb675fd1d564b5f4746e76aa4b24ad121a907b8786715483067f46f1cd4a3dad5125f58c3d348db776e99e8a562ebe603e0e98509c72118ee81259b43f1f770dc9220dddbe4d12d2ffabdad76663d1cf30304eacbb43e9bbe0e4f95aae8609bf8a0786e56e0d20d4875d17cec644ca16824424c6c0700e5082ad5288d1a9fec637b75d24c270086a3606b97cc3240314ac123c04ff4d67ef547504f0eeea5e6252f9b5b75e47d32e583c4dee95ac84b6f03fef412e24c09697d6f7bc8408c2118c524f8e277b82658a5869d6d69d7fde469be27ea33fa22b47f3276f60580f2a05b0fbceb67949d2c8e6d8b097ceef6781d962866b23bc71b54597dd05caafbd533ff4e0013f0fc7e202782ba1e6ccd11242a5bc823af9ad0d5bcb8b78d0ce150fc6f8b97c66f9337adaf3878cbb70733044fd14f318d8e389940d07e0593e35f46dc84e7a9a75dadda153bbe5755af0683cfd262ce3797b1bc648a4e5cef5d9aa47d08cf78c6c9dd872096deb856680d5d932abfd5f2024f908ab84d19b762fdcd31209b1d9adf9cbab904bc58f5b00d5a3e0eb2ade52c7c9a678aadedb1b9257263e6145d34ebcc250d0867e2fdc2ed23b4a8e8aa580424308b543b76595f7e09127e8c301d3f6c228e656df2b5e63a89d634db2f87249025ad3b53d2703d6b93c4ab6b1dee4f9b4d2a0b383ed87458e0e1b2a3c04f9a08b885d856ea7ca9abda9a9c8bd28ecce78675b16829493dbeb66963a3665a8386bc4b97d13cfacacfa80b12e36b598d4f09ab7847481a95b35a937bd4f0107598844fc518c4134e8ca55cec024773dcf0a17c71f602406363dcc03912f23cffad9c611e34605fb03fea3d84aac33886efed6c53d455c786231436c01524cd41e2b6c35b339feaa94d7df0b3572db5595c05a1d4a96e1a17e814cea965be81786784af5697b5c7909fd7701d0f5d991f469e20fbc39536239c7a53278c0b1189253d3d978d54b51d33e0c8ac03ca1f08cfd440130944ae9d353b0b7f263a3a27d15290f35c0ad7cf62e8429ebfa10f5471987c53c5d574c9aebb935bdb8339fee9e1030b3f2845c36272642233afba71f81f10322057b51a33b434a4bc45530581c4a636895fc773e20cf6b8a6cef060920310f4cb79f735fc14e758f646b0cf3bf0ce1f1479788f5b73507203d54022c7afd805d46f45e7ede96b2660869939440d7d575fbf372a7f63421728343dfd88c5fa29ea3f15b68fe360f54afa0681093f2c82888d6babf9558159383c1a76264deec2d7fccb20dbd3ab4b3f137fb3dcf253fb6953df2f663d08e8b39909c9dbff98d49377ce0ce03c580b5e1bca6fcf58aceda42b00b1f478cff2995d47c82f2d5a3dc0e9718a0a46d8330353f2039e68d9565d29e77efea91e058ae03140a4b58b297ae664c9f0e7af78f3633f4aeabd95fd367ac5c67eb63fa2bd7035f4adfe3ab8502952d1675b47118b2006b4a5a1b2a3d03aa670862adc5f1ad1b39f0e4a08e2412128e0ef5fa84366b4b50992cc139e7837f8d65f8eae3c8dc2730f2ef1e1585ce95b0fe354c6c853526558a1732171b6017254ba0f0c22273b55caa2ee680791fb34e22bf897a0998156083c83d03310956b1f947ef474c90b2d7e20beae27f5d33b79ad6d9b1b0188e3cf850108f3a02f9386a314b97fb6c946e572e40024da3e4784c523cf2a70e45f6c0f8f35e9b279aecee183ccb30477ee4f5d57c246ab9918495159307935340a92bd46d6519bcc51af4a785a7eb7fa6eae9def89e2411efd2cb2c33f4d605b77af1ae298967fd846453e8b0a55c57706e79d7badd93268d93be27790bbb051496552b094f37bf96843cdf7604dc7f696976bebbe3561c7bd4b3f2843bd07e34a3c3acf0d6c755014a3e9d922dabdbc892511b6af3214958b3531baceab9082c2b4e19ef802b99db2cfa4076ba681efae8a9546582e4cbf07000fb3903727261b93d1bd08809ddc41b2a61b0bfa6b9210b8731394fd9084bc1406bcdd0992410e90f749a087edafb76f6617209855269bb5c89711d2830ea99f2a0e548991ef8dd3e62ced239dff4e9d8c1e834fb57078dc2a4322acc1ea1dbe64064db4cc2a7cdc32884cb7c31aa3c95634356b4d32f1c92fa039e5ed18c1af6605e2f66e5383fa1fcc610a3cc1eef5fe6296ac378f2440d0bd3c77d458af0bfa6b64cedd0301b116efafbbb8f9e88d4bedf56d11cfe0967fc06932e1f232c7daf2c73b58c52daa82dc22b2290147e358348f991f1473c4e63ac943cabc429f5689da8aee6801fd941778966da87fe137b033a0231a90a2ac759efbcb7c52de8e8f27bcc4e5bcb560524d17b0345727f8092e22b6a0e03ceee355085d4fe81568b5b8b84b69870dd333c9b3bdca45db7e43b876a217819928fec31f3ca4c44c4f03a91d578f9ed883bf1247a10eefe42279484af6f70719e193b07a14fc7d93fd6cee9e883a81cc51b53044fa2f1b2525c10e23fb988b31559e1fae4c082d905486d984d2f1e87c40976a571f92e4fdf09e23cad561d9cd3d1ca94e778d08b82852dbe9bab2c7f147c6078504c30fc3fe749fca95ed23791dcf6b935ab0ed08645c481cd2b7969258405f183593617c0d24ebe79bc6c87df65ddc25f933b12a641cc95cf321c1b2edf5db5813a98d99fd1760b5fac19c2b47c2a750e96da49c97276cf31cbd73e1e93ef6990fdf1b08a3e7963dffcf65aa6496727f6be5d9225cfbbadfd6a3a06a454acbb80e761699be97b9caeb71dc3c20a2c253f349ee190b9d8c1ff95751a281ed88b098e4a78fae8249232d4280daf46580ca6c14411f8c4d0e8fa44f8808a5a2dda36d35f79aa77c1c4c615216ec309aa5f3caff3a449b9740baad11e525a5651416a12dd4688897b5ee5e1e8361a87269e9be0789ea7564f64752788f3ac57bda4aa45582d8dabc4d44759639984514d9b151004bf0d9d3140becde41603ffd9d412ec08a1b2f7262862a761d7ce5b91a239d68cdfe7c615b62217422c5a224e652cffdbbe7b41064c9375bbf2c23b63130afedd93987c1f1e4a1623a77d3ef4960fd232afc7d12159c8f1245984e4bd390685668f10da77bf5f6130a5d405a42fb02d0048d63ca48cd4901f3704975b0493714c78de33c1a969993529bac13e310ee098a7b1e8e93b4fcb269773a1521defef98f403f79a8e24e1594b8dc6055223b79b4d7009498c02790a37dd3a9ac7168b11568877ee852395165f87d881355327e57ba48be948b1325824244a552ca80c2ecb79a04a199f0b6a58b455327d998a67ebae414394928161a311de9d3d7f1fc99c0bc35b0a814e7821d2d6958129775c56fdd9b28b6e43a3132a95eb64805f9941b0c5ca2078c37bc7a4d43ddcbc6e8c66de2ef51c714a81913bebeb8f9c6aa14da7e71854870fba9f093aa77abeb23e8d37f3a50135a3f894d1cdf647d3dceb3f16ad4b4f01dffe99d74cd1f18d37735ac69287df0a8b49b2fc3773bf426c881b64a1241c1143e39b6dca59e28c116947aac39585dd81f6fe3832f97d873884738982976f9f29aa6463662cc81c2abbd5eafbb3d9800ba811410a521065d61ad01cfde2c2d98f5fa0ec63e8b01ec6a2d738a1f6cf8c99725079aa51ed3dc7630deeea0b27ee8edbfa19bdb8504cf80a1ca74ea6777fa8786fd76d4ab019c00da35278bf49be5a42aff264875b69bab9e7f3f9299f2ea6a883fc5fd77e34debf414ce8af37a3e9a7de05f33bc547899d25a13389d9048be8b2b7c9b836b401eac9c53ba3d44a14e683a6b378248fdedcbfbc408e5ec83a6c3f769533ae90e339222d80d433ae820fca43d80bca7fbd9d6e5eb760271c74a95a35a853285f7dcfe10182af922508501d269cc53ed38756e8d202b25782ec208a39981ef6c9e9342eb6cee2142471574ab39ebdbbf140f64e305d6f6aa73d49d5bb765347a27ebd0ba1bdcaa8a383526b41f3361faddafec726e185cc223f8465472d7a53ad2d74717960ba5684bc692f773e680d0247f9de1650c046ea56fa18c1c3fb6636327f863f37dca86867c57cc643ff6b7973f97c91716502c33f0dc66176357315f09064c70df7db026413567ac9ec7605acd80606d962f7463359998b92f8a33ccffc80f6e162c1b5a65404e0cc4e37688ded910e6ae5f774e3238a1248b05c53486d149734dd1e38073603df0aa4a4f01fd0c1a0e82b6a513f7d22d1b09ff107923462fbfffbf5fb1f2c043e305917eee1f7f18f62dc604a1d39e5bf675dca67da52f95d312b31be7720efd391bf4794f0555b299325b5dfea9eb00e695861634ad28717cffff266e30c864ba844b0c3fd88bd0cee4d929530c2bb3663c5170a15a66c412ff2fed2d962dd0ff145f19f8085931cbd6bde4c11c9c2debfbdb6748d1a6dabf1404762343d468944a0495026091bc44c69dad971890b7221e1eac2e985097d344a4e2375938aa93806ad1715be8d05f4068ec67ef411e704b01851b7427c4137e6dadedfbb25378ecf1c6b749214f2655cf43fdbb72cb842aaab3074cd792f30f96e874d92f2ef62667d81ccd663dfd969c18dff790b6b7b261f0a03baa85bfd7db72baffe2ddc4cc3985183301a77daa826b1f76f6b6f2ac075c6a2b86609e4d26eb08f3ed6f341d7946966e654ed2c30a629ff81a57014a84105a9ad36a525033f16e3e60d639bf8f89ae6cfdbcdb41e93859354957dcfe9a847757c3cd946d8994cda126f146b77119bbc87c49ce79ad844715cd9bb0c7a4f800fce14c81d175aed59fce0377e62c6e597ab5acf1b3b7100403f371c19dd0f131c4c572e57e3e13743f9bac24a6a177d71f03c5d185bb7e163ec5866dda629340a964bc442d423ad7bc5187f3da68d1498dfe9f1815b31bae11df3585ece230cc3521f7a0b4b3360ddf898984b528afff75f229915f1f2d5c4491c65e2f38d26c0de7dc483860da7bf52859fdbc21946badec0bbf00ba8143b8b7289cfb7096d3405e3183e56f1cbb8c48f25d058530894d0302cdce7c43ea31768b5b610820c97f6e9a8e31a8e9c4624117d213a03ef7d655513cffd8fb5606fc8790692c99828a47382e3e37b9c1317028f5c9d196ff3c2f09435df7614fb37ea20b2371c52b6c4667799922010edeb28001de2a137889db4d3dbf0a406ba90f3631be485ec5e4110b01502f557f15026716030fb6384499adee3cfc44015638e05b206ce3cbc3cbf21b7dd1dcb3b932629e7cd4f4f6b148f37f976803644e5ce792583daf39608a3cd02ff2f47e9bc6"

var hashFunc = sha3.NewKeccak256

func TestEncryptDataLongerThanPadding(t *testing.T) {
	enc := New(4095, uint32(0), hashFunc)

	data := make([]byte, 4096)
	key := make([]byte, 32)

	expectedError := "Data length longer than padding, data length 4096 padding 4095"

	_, err := enc.Encrypt(data, key)
	if err == nil || err.Error() != expectedError {
		t.Fatalf("Expected error \"%v\" got \"%v\"", expectedError, err)
	}
}

func TestEncryptDataZeroPadding(t *testing.T) {
	enc := New(0, uint32(0), hashFunc)

	data := make([]byte, 2048)
	key := make([]byte, 32)

	encrypted, err := enc.Encrypt(data, key)
	if err != nil {
		t.Fatalf("Expected no error got %v", err)
	}
	if len(encrypted) != 2048 {
		t.Fatalf("Encrypted data length expected \"%v\" got %v", 2048, len(encrypted))
	}
}

func TestEncryptDataLengthEqualsPadding(t *testing.T) {
	enc := New(4096, uint32(0), hashFunc)

	data := make([]byte, 4096)
	key := make([]byte, 32)

	encrypted, err := enc.Encrypt(data, key)
	if err != nil {
		t.Fatalf("Expected no error got %v", err)
	}
	encryptedHex := common.Bytes2Hex(encrypted)
	expectedTransformed := common.Hex2Bytes(expectedTransformedHex)

	if !bytes.Equal(encrypted, expectedTransformed) {
		t.Fatalf("Expected %v got %v", expectedTransformedHex, encryptedHex)
	}
}

func TestEncryptDataLengthSmallerThanPadding(t *testing.T) {
	enc := New(4096, uint32(0), hashFunc)

	data := make([]byte, 4080)
	key := make([]byte, 32)

	encrypted, err := enc.Encrypt(data, key)
	if err != nil {
		t.Fatalf("Expected no error got %v", err)
	}
	if len(encrypted) != 4096 {
		t.Fatalf("Encrypted data length expected %v got %v", 4096, len(encrypted))
	}
}

func TestEncryptDataCounterNonZero(t *testing.T) {
	// TODO
}

func TestDecryptDataLengthNotEqualsPadding(t *testing.T) {
	enc := New(4096, uint32(0), hashFunc)

	data := make([]byte, 4097)
	key := make([]byte, 32)

	expectedError := "Data length different than padding, data length 4097 padding 4096"

	_, err := enc.Decrypt(data, key)
	if err == nil || err.Error() != expectedError {
		t.Fatalf("Expected error \"%v\" got \"%v\"", expectedError, err)
	}
}

func TestEncryptDecryptIsIdentity(t *testing.T) {
	testEncryptDecryptIsIdentity(t, 2048, 0, 2048, 32)
	testEncryptDecryptIsIdentity(t, 4096, 0, 4096, 32)
	testEncryptDecryptIsIdentity(t, 4096, 0, 1000, 32)
	testEncryptDecryptIsIdentity(t, 32, 10, 32, 32)
}

func testEncryptDecryptIsIdentity(t *testing.T, padding int, initCtr uint32, dataLength int, keyLength int) {
	enc := New(padding, initCtr, hashFunc)

	data := make([]byte, dataLength)
	rand.Read(data)

	key := make([]byte, keyLength)
	rand.Read(key)

	encrypted, err := enc.Encrypt(data, key)
	if err != nil {
		t.Fatalf("Expected no error got %v", err)
	}

	decrypted, err := enc.Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Expected no error got %v", err)
	}
	if len(decrypted) != padding {
		t.Fatalf("Expected decrypted data length %v got %v", padding, len(decrypted))
	}

	// we have to remove the extra bytes which were randomly added to fill until padding
	if len(data) < padding {
		decrypted = decrypted[:len(data)]
	}

	if !bytes.Equal(data, decrypted) {
		t.Fatalf("Expected decrypted %v got %v", common.Bytes2Hex(data), common.Bytes2Hex(decrypted))
	}
}
