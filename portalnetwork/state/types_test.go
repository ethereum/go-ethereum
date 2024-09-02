package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNibblesEncodeDecode(t *testing.T) {
	type fields struct {
		Nibbles []byte
	}
	type args struct {
		buf bytes.Buffer
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		encodeds string
	}{
		{
			name: "emptyNibbles",
			fields: fields{
				Nibbles: []byte{},
			},
			args: args{
				bytes.Buffer{},
			},
			encodeds: "0x00",
		},
		{
			name: "singleNibble",
			fields: fields{
				Nibbles: []byte{10},
			},
			args: args{
				bytes.Buffer{},
			},
			encodeds: "0x1a",
		},
		{
			name: "evenNumberNibbles",
			fields: fields{
				Nibbles: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			},
			args: args{
				bytes.Buffer{},
			},
			encodeds: "0x00123456789abc",
		},
		{
			name: "oddNumberNibbles",
			fields: fields{
				Nibbles: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
			},
			args: args{
				bytes.Buffer{},
			},
			encodeds: "0x1123456789abcd",
		},
		{
			name: "maxNumberNibbles",
			fields: fields{
				Nibbles: initSlice(64, 10),
			},
			args: args{
				bytes.Buffer{},
			},
			encodeds: "0x00aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := FromUnpackedNibbles(tt.fields.Nibbles)
			assert.NoError(t, err)
			err = n.Serialize(codec.NewEncodingWriter(&tt.args.buf))
			assert.NoError(t, err)
			assert.Equal(t, tt.encodeds, hexutil.Encode(tt.args.buf.Bytes()))

			newNibble := &Nibbles{}
			err = newNibble.Deserialize(codec.NewDecodingReader(&tt.args.buf, uint64(len(tt.args.buf.Bytes()))))
			require.NoError(t, err)
			require.Equal(t, newNibble.Nibbles, n.Nibbles)
		})
	}
}

func TestFromUnpackedShouldFailForInvalidNibbles(t *testing.T) {
	type fields struct {
		Nibbles []byte
	}

	tests := []struct {
		name     string
		fields   fields
		encodeds string
	}{
		{
			name: "singleNibble",
			fields: fields{
				Nibbles: []byte{0x10},
			},
		},
		{
			name: "firstOutOfTwo",
			fields: fields{
				Nibbles: []byte{0x11, 0x01},
			},
		},
		{
			name: "secondOutOfTwo",
			fields: fields{
				Nibbles: []byte{0x01, 0x12},
			},
		},
		{
			name: "firstOutOfThree",
			fields: fields{
				Nibbles: []byte{0x11, 0x02, 0x03},
			},
		},
		{
			name: "secondOutOfThree",
			fields: fields{
				Nibbles: []byte{0x01, 0x12, 0x03},
			},
		},
		{
			name: "thirdOutOfThree",
			fields: fields{
				Nibbles: []byte{0x01, 0x02, 0x13},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromUnpackedNibbles(tt.fields.Nibbles)
			assert.Error(t, err)
		})
	}
}

func TestDecodeShouldFailForInvalidBytes(t *testing.T) {
	type fields struct {
		Nibbles string
	}

	tests := []struct {
		name     string
		fields   fields
		encodeds string
	}{
		{
			name: "empty",
			fields: fields{
				Nibbles: "0x",
			},
		},
		{
			name: "invalid flag",
			fields: fields{
				Nibbles: "0x20",
			},
		},
		{
			name: "low bits not empty for even length",
			fields: fields{
				Nibbles: "0x01",
			},
		},
		{
			name: "too long",
			fields: fields{
				Nibbles: "0x1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nibbles := hexutil.MustDecode(tt.fields.Nibbles)
			var n Nibbles
			err := n.Deserialize(codec.NewDecodingReader(bytes.NewReader(nibbles), uint64(len(nibbles))))
			assert.Error(t, err)
		})
	}
}

func TestFromUnpackedShouldFailForTooManyNibbles(t *testing.T) {
	_, err := FromUnpackedNibbles(initSlice(65, 10))
	assert.Error(t, err)
}

func initSlice(n int, v byte) []byte {
	s := make([]byte, n)
	for i := range s {
		s[i] = v
	}
	return s
}

func TestSSZ(t *testing.T) {
	originalData, err := GetTestData()
	require.NoError(t, err)

	AccountTrieNodeTest("AccountTrieNode", originalData, t)
	ContractStorageTrieNodeTest("ContractStorageTrieNode", originalData, t)
	ContractBytecodeTest("ContractByteCode", originalData, t)
}

func AccountTrieNodeTest(name string, original *TestData, t *testing.T) {
	testcase := expects[name]
	n, err := FromUnpackedNibbles(testcase.Path)
	require.NoError(t, err)

	accountTrieNode := &AccountTrieNodeKey{
		Path:     *n,
		NodeHash: common.Bytes32(hexutil.MustDecode(testcase.NodeHash)),
	}
	// content key encode and decode test
	var buf bytes.Buffer
	err = accountTrieNode.Serialize(codec.NewEncodingWriter(&buf))
	require.NoError(t, err)
	selector := hexutil.MustDecode(testcase.Selector)
	selector = append(selector, buf.Bytes()...)
	contentId := sha256.Sum256(selector)
	require.Equal(t, hexutil.Encode(contentId[:]), testcase.ContentId)
	hexStr := hexutil.Encode(selector)
	require.Equal(t, hexStr, testcase.ContentKey)

	newAccount := &AccountTrieNodeKey{}
	err = newAccount.Deserialize(codec.NewDecodingReader(&buf, uint64(len(buf.Bytes()))))
	require.NoError(t, err)
	require.Equal(t, newAccount.NodeHash, accountTrieNode.NodeHash)
	require.Equal(t, newAccount.Path.Nibbles, accountTrieNode.Path.Nibbles)

	// content_value_retrieval encode and decode test
	accountProofs := original.GetAccountTrieProof()
	// get the last item, according to trin test
	trieNode := &TrieNode{
		Node: accountProofs[len(accountProofs)-1],
	}
	var trieNodeBuf bytes.Buffer
	err = trieNode.Serialize(codec.NewEncodingWriter(&trieNodeBuf))
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(trieNodeBuf.Bytes()), testcase.ContentValueRetrieval)

	newTrieNode := &TrieNode{}
	err = newTrieNode.Deserialize(codec.NewDecodingReader(&trieNodeBuf, uint64(len(trieNodeBuf.Bytes()))))
	require.NoError(t, err)
	require.Equal(t, newTrieNode.Node, trieNode.Node)

	// content_value_offer encode and decode test
	accountTrieNodeWithProof := &AccountTrieNodeWithProof{
		Proof:     TrieProof(original.GetAccountTrieProof()),
		BlockHash: tree.Root(original.GetBlockHash()),
	}

	var accountTrieNodeWithProofBuf bytes.Buffer
	err = accountTrieNodeWithProof.Serialize(codec.NewEncodingWriter(&accountTrieNodeWithProofBuf))
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(accountTrieNodeWithProofBuf.Bytes()), testcase.ContentValueOffer)

	newAccountTrieNodeWithProof := &AccountTrieNodeWithProof{}
	err = newAccountTrieNodeWithProof.Deserialize(codec.NewDecodingReader(&accountTrieNodeWithProofBuf, uint64(len(accountTrieNodeWithProofBuf.Bytes()))))
	require.NoError(t, err)
	require.Equal(t, newAccountTrieNodeWithProof.BlockHash, accountTrieNodeWithProof.BlockHash)
	require.Equal(t, newAccountTrieNodeWithProof.Proof, accountTrieNodeWithProof.Proof)
}

func ContractStorageTrieNodeTest(name string, original *TestData, t *testing.T) {
	testcase := expects[name]
	path, err := FromUnpackedNibbles(testcase.Path)
	require.NoError(t, err)

	contractStorage := &ContractStorageTrieNodeKey{
		AddressHash: common.Bytes32(hexutil.MustDecode(testcase.AddressHash)),
		Path:        *path,
		NodeHash:    common.Bytes32(hexutil.MustDecode(testcase.NodeHash)),
	}

	// content key encode and decode test
	var buf bytes.Buffer
	err = contractStorage.Serialize(codec.NewEncodingWriter(&buf))
	require.NoError(t, err)

	contentKey := make([]byte, 0)
	contentKey = append(contentKey, hexutil.MustDecode(testcase.Selector)...)
	contentKey = append(contentKey, buf.Bytes()...)
	require.Equal(t, testcase.ContentKey, hexutil.Encode(contentKey))

	newContractStorage := &ContractStorageTrieNodeKey{}
	err = newContractStorage.Deserialize(codec.NewDecodingReader(&buf, uint64(len(buf.Bytes()))))
	require.NoError(t, err)
	require.Equal(t, newContractStorage.NodeHash, contractStorage.NodeHash)
	require.Equal(t, newContractStorage.Path.Nibbles, contractStorage.Path.Nibbles)
	require.Equal(t, newContractStorage.AddressHash, contractStorage.AddressHash)

	// content_value encode and decode test
	contentValue := ContractStorageTrieNodeWithProof{
		StoregeProof: original.GetStorageTrieProof(),
		AccountProof: original.GetAccountTrieProof(),
		BlockHash:    tree.Root(original.GetBlockHash()),
	}
	var contentValueBuf bytes.Buffer
	err = contentValue.Serialize(codec.NewEncodingWriter(&contentValueBuf))
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(contentValueBuf.Bytes()), testcase.ContentValueOffer)

	newContentValue := &ContractStorageTrieNodeWithProof{}
	err = newContentValue.Deserialize(codec.NewDecodingReader(&contentValueBuf, uint64(len(contentValueBuf.Bytes()))))
	require.NoError(t, err)
	require.Equal(t, newContentValue.StoregeProof, contentValue.StoregeProof)
	require.Equal(t, newContentValue.AccountProof, contentValue.AccountProof)
	require.Equal(t, newContentValue.BlockHash, contentValue.BlockHash)
}

func ContractBytecodeTest(name string, original *TestData, t *testing.T) {
	testcase := expects[name]

	bytecode := &ContractBytecodeKey{
		AddressHash: common.Bytes32(hexutil.MustDecode(testcase.AddressHash)),
		CodeHash:    common.Bytes32(hexutil.MustDecode(testcase.CodeHash)),
	}

	var buf bytes.Buffer
	err := bytecode.Serialize(codec.NewEncodingWriter(&buf))
	require.NoError(t, err)

	contentKey := make([]byte, 0)
	contentKey = append(contentKey, hexutil.MustDecode(testcase.Selector)...)
	contentKey = append(contentKey, buf.Bytes()...)
	require.Equal(t, testcase.ContentKey, hexutil.Encode(contentKey))

	// content_value_retrieval encode and decode test
	var contentForRetrievalBuf bytes.Buffer
	contentForRetrieval := &ContractBytecodeContainer{
		Code: hexutil.MustDecode(original.Bytecode),
	}
	err = contentForRetrieval.Serialize(codec.NewEncodingWriter(&contentForRetrievalBuf))
	require.NoError(t, err)
	require.Equal(t, testcase.ContentValueRetrieval, hexutil.Encode(contentForRetrievalBuf.Bytes()))

	newContentForRetrieval := &ContractBytecodeContainer{}
	err = newContentForRetrieval.Deserialize(codec.NewDecodingReader(&contentForRetrievalBuf, uint64(len(contentForRetrievalBuf.Bytes()))))
	require.NoError(t, err)
	require.Equal(t, contentForRetrieval.Code, newContentForRetrieval.Code)

	// content_value encode and decode test
	contentValue := ContractBytecodeWithProof{
		Code:         hexutil.MustDecode(original.Bytecode),
		AccountProof: original.GetAccountTrieProof(),
		BlockHash:    tree.Root(original.GetBlockHash()),
	}
	var contentValueBuf bytes.Buffer
	err = contentValue.Serialize(codec.NewEncodingWriter(&contentValueBuf))
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(contentValueBuf.Bytes()), testcase.ContentValueOffer)

	newContentValue := &ContractBytecodeWithProof{}
	err = newContentValue.Deserialize(codec.NewDecodingReader(&contentValueBuf, uint64(len(contentValueBuf.Bytes()))))
	require.NoError(t, err)
	require.Equal(t, newContentValue.Code, contentValue.Code)
	require.Equal(t, newContentValue.AccountProof, contentValue.AccountProof)
	require.Equal(t, newContentValue.BlockHash, contentValue.BlockHash)
}

func GetTestData() (*TestData, error) {
	res := new(TestData)
	data, err := os.ReadFile("./testdata/data.json")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, res)
	return res, err
}

type block struct {
	Number    uint64 `json:"number"`
	BlockHash string `json:"block_hash"`
	StateRoot string `jsin:"state_root"`
}

type account struct {
	Nonce       string `json:"nonce"`
	Balance     string `json:"balance"`
	StorageHash string `json:"storage_hash"`
	CodeHash    string `json:"code_hash"`
}

type TestData struct {
	Block        block    `json:"block"`
	Address      string   `json:"address"`
	Account      account  `json:"account"`
	StorageSlot  string   `json:"storage_slot"`
	StorageValue string   `json:"storage_value"`
	AccountProof []string `json:"account_proof"`
	StorageProof []string `json:"storage_proof"`
	Bytecode     string   `json:"bytecode"`
}

func (t *TestData) GetAccountTrieProof() []EncodedTrieNode {
	res := make([]EncodedTrieNode, 0, len(t.AccountProof))
	for _, proof := range t.AccountProof {
		res = append(res, hexutil.MustDecode(proof))
	}
	return res
}

func (t *TestData) GetStorageTrieProof() []EncodedTrieNode {
	res := make([]EncodedTrieNode, 0, len(t.AccountProof))
	for _, proof := range t.StorageProof {
		res = append(res, hexutil.MustDecode(proof))
	}
	return res
}

func (t *TestData) GetBlockHash() []byte {
	return hexutil.MustDecode(t.Block.BlockHash)
}

func (t *TestData) GetStorageProof() [][]byte {
	res := make([][]byte, 0, len(t.StorageProof))
	for _, proof := range t.StorageProof {
		res = append(res, hexutil.MustDecode(proof))
	}
	return res
}

type testcase struct {
	Selector              string
	Path                  []byte
	NodeHash              string
	AddressHash           string
	CodeHash              string
	ContentKey            string
	ContentId             string
	ContentValueOffer     string
	ContentValueRetrieval string
}

var expects = map[string]testcase{
	"AccountTrieNode": {
		Selector:              "0x20",
		Path:                  []byte{8, 6, 7, 9, 14, 8, 14, 13},
		NodeHash:              "0x6225fcc63b22b80301d9f2582014e450e91f9b329b7cc87ad16894722fff5296",
		ContentKey:            "0x20240000006225fcc63b22b80301d9f2582014e450e91f9b329b7cc87ad16894722fff5296008679e8ed",
		ContentId:             "0xe9d3cd4020b96d4c9222854f541eac0db76335c22bc3d1ea002f0a9ddcad7bf8",
		ContentValueOffer:     "0x24000000cf384012b91b081230cdf17a3f7dd370d8e67056058af6b272b3d54aa2714fac24000000380200004c0400006006000074080000880a00009c0c00008f0d0000e20d0000f90211a0491f396d5d4768a01ee4282a3ab1127f2a4dc7d42e6c1dbb3e71ad4e9299f5e7a05b4645219e614b388ba9672452b40f291987b15e35bdbd3dfebfac9a085aeab2a0979ebca2a6a0df389fdfef5bfa4a31f2efa8d385bf2f43cd69c27e61165c3667a01bbe04543bb6bf8026ee3ec2a3ec3d6173a60c090008b59779f767e5769d7a0aa051d347ec61c7dde5c4149a943d0aa489544cbea511ed3d7f4fc6aaa52a420d3ea05358bc8e1e1f20e510887226d67efee73769d0b13ebb24a3e750c2214ef090d5a04df1c24ebf40befce60c8eb30a31894102881454fcdeb6f7b83e1fb916c953a8a00a85e04f30a4978712c58a825e7bd2f8a83731ab1aa3e234a0207e918790505fa0778a45437218486b7849aef890d483dcc2deb1423cb81e488b7be2921d505bf4a051dbaef3c3d3fe82bd1484954f38508f651d867f387f71dedbfb0f7355987ed5a061679f43f3db26673bb687c584f30ae3cff7261e71acf07ed1feaecae098fc79a099761ea7d94e01b14285b7ced7dcd1677fbd3ce093a627a8e1472074baad8bd9a0bb6d269fc61443aa28b14af0448aaf6f1f570477d8b4ef2eb7d31543202ce602a06c41cd2d1701b058853591a2f306ddcfd1e55d3e12011cadb100e7c525aaf460a0bcf37abaac566bb98e091575af403804dbb278fbfa5547de6805cbaa529ae137a0de9918a2a976a2b0b3f4aeff801fc79557426b19c15d414ddf8fae843785b0b780f90211a0be51f8518274d3e1beeaaa8cebe00428d5fd388e5a2a50404a4278e4dbb822b8a0c89fe8e16d89a82d5dcde02ef4a861c49549f23caa89c6772cd3e046924170a2a09d3e56c85bde8ff3e48f1029a3ed19f6cc1657ea7148396264117128916bf192a0dfe966436ccd76314d1ff27c3f314ba4bd67cdbb84bf6e7aa838407acd31f8c2a02a4659357dbb06d71dc46b900459e5bff6f3e49e4ef001cf4c8a2c62a7a227f9a02db2fff7623b55bef3e3e97d4596d35bf648209ed27a1dfa4574a950e76e9866a0e9b7cec39e827b538d84d697c774c2415d4c779b4633e3512b1b3b037e5be466a024a73f497d6080270bd263da5eb8c65a8495202d23880bb332a5fc7c4b18c4a8a08461ab3f12ab5d327b9496351467b25b27fab2e83c0c05618eb1fc20611f1958a022c7ff9562e90fee635a098995fb60330f313c683bf387b16e36f3ed01df69cca068c8d94e7b4b8a512cc108fd5fc723c53524f18d15c5aa354a5ed4076d5818a6a0d8162bc7fbbc17e616c126ba764cc4117ff3ec948a3bd588ea3287f8b901c4a7a0292d04df2f33b68e0033209c36035212c23581c8be47c5e0ea67e18fe8521f58a0e7ae2ece7c4c44a19a4d576904e39574030d6f739b450e75f36c59c83ae91829a0d577206430b32c22323816ba5309a85e5f71954e2bf5c755f5e33ee00f0de899a0f1f58b86f4addc6c095586a5333d6816b9b8a6d97b486798442863ea9615ae1380f90211a0256e27eecd0670e56ade58d99348a11f6fa4985d1f7b23143c61ee0a04a5a053a050b3d0ae7fdcad74ee9e322bb5b5fedb329db8b5bb95b281ee6d5b090314520ea07800c5365602d0c6eed658ee1056a0a2576809372757b1a42bafcc98438aebdba09bb7c947da462574a0fc0ae5608c25ac3272bb8fe3f2e619b20a10ced54738cfa0f73e7c44ce8ab0bc3495eba1aa7e2a0db44505d690e9d2daf4fab856cff999e2a022820f57846aa9ca4d0ebab020327b14178da53c2e006aaf5a22cc6648bc055ca0989235e562ad6e89656e7b8ccb301c300138d1eceddc2cfb724799a13f089783a0aa9b63d694d348219c0fb7f77b45e0e6835c99cda60a8f2030f0f712716b3751a0a34922189d09d23f5e0c8ff2f0b2407ff98a308cb3e8a9239cd4c776cee1e23ea06af9c10b566519fbbeb36dd1b3458b23fd25aeef27275a88df2d7662c2084ca9a05ce8b9e271738c6c52b1078ec04566681c8f11f679eeb2e83197431798cae230a05bfd293e2680e5c45fba826bb768e4d2cad9ab1451a712e98b0cdfba4d31b3f7a0f3ca96a33140e3e0e6ed23e857b6e5c5c7bdb77d7ef520aa4e10213d525bae84a003de0c0d867465dbb29e938fe08f3a508b26abacf7fdf8e02ffb485229195f14a0141a995d94148f08196179583fc853e65f18c338b1afe93b948701b795720960a01a35da701f908a90c16c4cc97de451ac2c35d52b019a58a078e61afe0b9a1cd480f90211a019c38310558d06bcbb4da68f64e61f2e6ee0f9bf3f6d643099e2e5c537294f6ba03a2ea461fdc547b6f72bc5d0b6f0a350aea0cb4b8ce1b036d123a5a00ec58b4aa07804bb6a71cf8c1153df5aa4ef84a409f26dd96c376e793da8cf9bcc802b4c26a076b61be012d6c7494b54c77a2dc6fcfe5568eef756d7a9b827c71ff25849dd87a0c8e8448bf1fbb0511823f5a92ff0febd7cad713967b5bbd0a4603d4ef2b9ef01a04c6d270a17b8fc136e0741940349c43d1e4ca894f191eefa13851da489de99d5a01076e9645c48dfed84933a8fae6016be4d47946f8105ccbc892ab0ae6ccf837aa0c6fea6b5dd5a7bae7e3c2ffae6841b292b613bf4c7421bdb90226f53e9eebc51a073b6c1de7658ec6f66fd52f1d66ac278ed0f3db37201981a9ca4b64a951fc52ba00bc5ecf44d423b4b6e0030d057939b739b85981c2bf2b4ad4f968bebd83cba7da0ed7e7a51c44fc09f230853a018c49eb711fbfa439904fd5ceb99c00b166f3ac7a005bf4f540bbc5bd6731dd89f4ff0a28c8ad8ee39d9611ef4bda0e2016bbd644ea0a1e0d3c4b952dce2233de45cca6e6c771e3f552ff6ff0194ff3d1b44d620683ea081e8d789f13dac46de04320c3a9c7d3e488ec667a619c21a020f95a41093e338a0dc9b8851a29da9247a4423fdcc689ae44c7843d31a891a067509780a2664c0a6a064f75b0a17b502154c6d316e7244bfde51a39b211484d311be73b80fbb93fd9c80f90211a09c0680faf2d7a9a79b5eedf6a8f846b95431cf91391973f6d1a6ec66dc7c3c24a02fd29886972cc78eab95ef19fd5cdd434e8d60c79b7d7aaff5b183469b848892a0cd246c92230812e63f13e8fddf2d0a0a976fde22e78256f9f3c7fd8219c71494a0f1d0b771957fe8d4c908d4ca730deca96c4ead47a9cde9648dd371da93bedbfca0de3910f22bd6ac087a1f778dfb20c7df43f7b3177e1abfe89d76d094dbd5753ba034b9341a9dad17687f5c22155e87ddbfea86dc0374e2071c6fac2a34fed5d7eea015eb41b2ece7cffbeaa040429aea2c73cf1c97a79282522f9eb25fcad1379b03a032214af47db0deb697b79ab12adc1b92cbace8395c8772544e803f33496c8073a01ac13a790f63114ce9bde82925bb5be8f4a09ee8ecf086d7675517248f8b8d4da04fe95c1a0f329cbf6b40efb608497e1d35f86919270a1c3b0a5b8f5c4675fbd6a0649e3ea0c951c4a16658f463755df800dd83f331f25a37275476877ce2eed44ea0157f094338c21f03a13a16793af96cc11a8f8a4a23b1f6105816833f603eabd0a086913ed8ad6c2c5d3d1e37b18bf7522aa080c04f3acd32a521efc7297653b658a0ed086a76c87f4654a164d1c4ec2626743cfa89bc1c4af41eba7369b42708c690a0c6730ce5e1df17601ac55b243bcba90a9f49dc835044bda9e2d7a76885b36598a0368943b40d006cdd58e0f576314c626fae7bc832d9451c5758a15c03e0bd08f680f90211a0515cb567f896e05d10003f6fff6b848448ecdfa82d6f854a8bf0058a57f53563a03661a6bd0c24511a05b40f8ae74c737f47d0df70bbf3c7a0816810ed90a84426a0c4d94bcf7df0b9c6707deb12770d992c1e60724ec2f40633a02ef1c4375326bba06af61ac19cd4496fb6d376dc38a8165a2561022866ce6fef41f8ade0fd2a407ba003a8b55e6cf12c06d0962f587bc293bca33303b7cc747f4ffbd17332cdbb33dda04f96708c335dd6364f8c5b4fc62edf32f5fd9a7107a99be1e9f6914b4093b88ca0e3981f17dbc1a00258432c0e34232f267d05e271590ee7a9783a51870f9873c8a03f2b45b47cdc12f020c55f6da3f7a348ec409de53a618b9cda4523c97d754b82a0935bc9189b81bb782d6b4d1451ab3d165b94a6fc4de4ab275528c06d9a128729a0c07c4048481c07b118748b2a30e851cf5c6c87af6212257f8845eb9ce07a02e4a0012fab999f402d6b302a07d3f7973ee8c357d5db8c7991903bd54d54d507cc23a05923155adc8cd3aceaf27c07858d290a3b85fc378fdd98e9f4f7f70d157f28baa0c55b2981a6fe08260cb6a076c76858d56aafdf255d0a12a2c50abe35d468c7e7a0e85b45f9a72f3abf2758687f2bdc33deab733aa4ef769bd5648f6a55ae1fb123a06ef38fec665b8eb25934622af1112b9a9d52408c94d2c0124d6e24b7ff4296c0a0867f6119f66c88787520dc8899d07d0e49598fa8dde1f33e611871eff6cd049680f8f1a0ca06c2b4c97d9941e56c3c752abe4c2b0b2cd162e22a5d25f61774dc453deedfa0344f34e01710ba897da06172844f373b281598b859086cf00c546594b955b87080a09bc4a42b6376f15f2639c98ad195b6fb948459cca93c568eacb33574b826a7af80a0525e7dd1bf391cf7df9ffaaa07093363a2c7a1c7d467d01403e368bd8c1f4e56808080808080a0758bf45f49922e3f1273d3e589753038a18ce3bbd961e3493f276eb7c5d04a3fa0235db60b9fecfc721d53cb6624da22433e765569a8312e86a6f0b47faf4a2a23a02f35f91fe878f56f1dd0b738bd12d9c8ed0f9b0f6be4146b66ae2c5625cc156b8080f85180a07f152c1e0fbe4b406b9a774b132347f174f02f3c2d6d1d4ad005c979996754b28080808080808080808080a06225fcc63b22b80301d9f2582014e450e91f9b329b7cc87ad16894722fff5296808080f8719d20a65bd257638cf8cf09b8238888947cc3c0bea2aa2cc3f1c4ac7a3002b851f84f018b02b4f32ee2f03d31ee3fbba046d5eb15d44b160805e80d05e2a47d434053e6c4b3ef9d1111773039e9586661a0d0a06b12ac47863b5c7be4185c2deaad1c61557033f56c7d4ea74429cbb25e23",
		ContentValueRetrieval: "0x04000000f8719d20a65bd257638cf8cf09b8238888947cc3c0bea2aa2cc3f1c4ac7a3002b851f84f018b02b4f32ee2f03d31ee3fbba046d5eb15d44b160805e80d05e2a47d434053e6c4b3ef9d1111773039e9586661a0d0a06b12ac47863b5c7be4185c2deaad1c61557033f56c7d4ea74429cbb25e23",
	},
	"ContractStorageTrieNode": {
		Selector:              "0x21",
		Path:                  []byte{4, 0, 5, 7, 8, 7},
		AddressHash:           "0x8679e8eda65bd257638cf8cf09b8238888947cc3c0bea2aa2cc3f1c4ac7a3002",
		NodeHash:              "0xeb43d68008d216e753fef198cf51077f5a89f406d9c244119d1643f0f2b19011",
		ContentKey:            "0x218679e8eda65bd257638cf8cf09b8238888947cc3c0bea2aa2cc3f1c4ac7a300244000000eb43d68008d216e753fef198cf51077f5a89f406d9c244119d1643f0f2b1901100405787",
		ContentId:             "0x696d71ff38bb79786bf25d30963e6ae07740788d46dbd8304355abb50fea3242",
		ContentValueOffer:     "0x280000005c0b0000cf384012b91b081230cdf17a3f7dd370d8e67056058af6b272b3d54aa2714fac1c0000003002000044040000580600006c080000800a0000130b0000f90211a0a0a734698552c6653d671994515b5f957ee1181abbfcda9d6ab8d245f89d0ebaa0674f8923e71d1155c248954b0f84eec649b0ae699442e707a998ec03773216f9a0ec35530a26811392e912f5c35df2b882cefba34f3f48190b3863cb1c78112520a09e19f39e13a8ada173c1723d1412592ec04e759da3e1bbfa5b8414d403c6badba05f12db0b204db3bc2d6c7b742d42ac8aebf20625465bb70268318068c6f95dd1a05aae7a4e7f57117bee5bef3c427f08e41ee81df13feb9663877d8763d0e67b16a0d911c69d2884ea02a8234b65576b60c89df36a3e41185d52f882f4800dac0171a07aab7513e12239d8a60120d9eb4d90d8857dcc2674180e80606f023c51a4ffdda0eacf1a83f5b253084e53a96532a75ac8346f12279595cbe485e3c880ed8a3bdba0c1d8d9c079eadb177ab06642433db67924c9f3291b85bfa33313e2f434bfa87da0c3a6abc8f5f37d46fd39330822449d0f2027571d5adbf186f66a3d153baecf58a0f07c540cacacb94fad3a8f913fd5bb60b12ea0b22f98deee14fbf64a402365bba0b2da1bbd6cda80337b186e485cf03318fa9eb895e2f10510856557b5556f1d19a0f08121eddb12c6b4c1471dcc76d4476af1b2387909e8c18fc02ff205b8734d83a0f1676391668dcee8af59dca3e49bf6adf69500187898992929d971ec019ba474a0eecf014df9ccb652baff40b102dfcc57cb8ec54805f907ce8a6a7d212bb47eec80f90211a0d44f76309b04285d7d4aff58b56aa3a2ff8b95da59b9535b7c9b31c9f582bad9a0b3dcf5ba7bb211621b0819f8adf1f300937e3c43699a6323fcd73b251b929a4aa05547a48db300342cad811ff8b0fc2af4f20f014ae6b5e5b86db51b07e9a833f6a0b90cbb4c7bd0b4f6f68e097c6309df5ded9424156638499a619643dd0205aeada0acb7bbf4a7013d34a5cac2c4c0e22d64c8db501589d1d1004c03f5bada1d747da0da0071908331247c33ccd3709a7e7d552591541874af73e1258a5df028ae5e63a064e408adb681365971c9fb12f53e4a8397419012881d2fad55a3760d97c3878fa0c33d50245a2c5afaa69511527bd7c47b319680edb536c940f43713cf556adf0da06e6c3c870c3c0b1adee85c44ede8cbe765a1473a8d0066c40f5402c38b6cf50ea0e88a4156c982d96e45c5d83b9f13cb7e8cb06bb0a1394b31a9d2709340c42588a0216b6c808f330cef5c684349690e514a4acf04c38fb3cdb18fdc6deefffb0beda09f725359320ca158811eb6af4654bd2dd68129019dd41028540cd4628f4504bda00fa15a861831834dd7a667e896aa0c87c8b48935435039157d8933f4d5aac90ea08e5624252293da2a75d756f10d23f7e7384b0b3b329512ecc88303408f3462cda0956adecc13bd4e8a14c43e5afa7b676578fb2ab7966037af1a94d095c7320801a0917af97d64aea571c5b2ea43300d863aa383aeb4d5f9807fe8e71bc01af2795a80f90211a077fb8073776e6bc376dd9c71048aa8e9f2fd36255cb92dd1bc77c1c68d0d7cb9a06134d20ea572e393ae7ca9e1d7c246731de3b573d32f469de00438880a9931a7a0503a7c42261fc325d6f5f7f45226affca2afc0be1105e957da5aefde4b6d05a3a0d9a27feeda690d27bdc3db7a0c5d6ffcf7ca7a44fcc84ceddaa0a53bfb8ad673a0cad9b09825416720f05a428b207cefdc768ca77a9c4f208f44adff0366bb6145a0a08eabe201b7a04939124052e364fe89b98d3dbeda2d92ff700d2358a319f238a0f2e6bb361675c8c1bfda5397e0530df89ff33d14bed771d054c26d1193468444a04707fa3c3c461698296bcde4e498cccb93a3f525811f8656d241f4573b10e3eca04d059adbc5185f218220e58e5bf9dfbd99952064f65ef8aa67ef541e1fa5831fa0871fb653564b4165fb10ce9145429dd15aac1707f8e580c783336f77bd5c9a96a043158ed6016199c7b9957c084c0349323b90cb8123a224c68bc222545b7c8321a00567ece2acc1800ebc43372660cbc3568b263d16833d49737d340a1b36fdd4bea0536c2166f8e88ec79e4dc0d5a3f46f44c8b5c4bc652ce91208d6388e4201ffd2a09ca540765900e026e46050fa9197218bd1075d24483c816ac8f42a0f819d9bdfa0a4d3505e91bbe8a40cd09280fcc4cf64d96a6e763831dec380b3f2ba3e37a4e0a0f245af40b496ce96c29ab97b6062a126be2f91b7761139a4258746ee891eec6380f90211a07662fedac79b0c428231ba283133012b0e3b75215b24a360a6a6eba6109653f4a0bcdfb8605e99657104135a390403b8f53804b5bf2203c9c1f5b3883180ec0bc6a0bcdf0d4fbd3f6a2656336c6df2b4fbdc79ce572671fcd26c4d66e58befc2d78ba0f7a4b4fb00382806a8e6b8b2559569eb7ec70b5e1e2e1c917f1f5c4b268c1600a0e9f007c95e2858bb59db8939bdcb8fb96d5100844551777c507794ed7d522d4ca0374b2c6be44f285ae1f87904618e3777ac999619153ee5d99517fdaad2368206a0f292ee114e85842bed63ba3aea7a8f22738b05f1a418cb30146f5edf46fc7781a0acfba64d149f25bc2a8ecd263bfe93b1363bbd3c364f047e0ff8f9b8faac1e75a0a28907885b59d01e2b5158f53880df233c0684ca242d7da1127238df84dc7c68a080b19db0f0c9965719499d3b90568e879ed429b6cffbf8103c1e21e8be07e580a0625dc89d33af85c1cf901bcce2686fa07672c51627866994c89510a6a7062912a0ebc653913a4833d2a971b12ce03178f63dead0c9e640fbd122563607f00f2d91a00cded7cb7c86826a9e755e04432340a9d4d8fa3bf5597e84ea91a7e6258e8f22a08e3b2e73b1cb8cfaa702e7b0cd5ff7e80ddb33f47dd16b5c0214103cb86aa003a0ead26815cc1a11843006f8892683f8732d3a519238b3cb63d2a57893aaaa0beba05c24ed83d1b166ecd5c9c1784bf113b0a9550c8909b0619133f559b37515dfc680f90211a0c675510b69c14ea4601897a064338a6b83692a6c29e6c6ac0f226025cda017c0a048baf29b758cb2b03835b778b1a3c80ce11e171614d9af7b1165f5c47fa290c5a03506d01d719efac9cf74f211b7ac92e8aecc05a3779018bc318741b60fee075ca0218f79c956e996bdc6a69476f0d691be85d625685f4e28a10e1c1a1289211356a0889eaa3d6289449e7ee16bc9bd14798223967a3b5a77e72f01ea39ba353ef42ba00eef8e973e31f307f1e74d54e4b13d169504eda8cf88d112b9f22d45c6ef08eda0a90cbcc7c75529502492cec75875c7344a3a5cc6ab91e3678ec8fc6eae250aaaa0b0a8313cdc75f6997ffbab1a2107b5ea0666b18999e9ae6c4217751d38ab85e3a07562413c88751df2b0c6c9c0490ebe5ed8d29c77681baf0dcb01183f2622a65aa0799d16f091a5bc05cb67a3fb489348845d5f548f5666b98941ebf392b3c02acba0d4c9d6926affbfd016d9474d49712bf10b75933010da8fb502c6345de068bdb8a00e6768ad039910330294c2a39a5c66065edf58913b7c9a974a57099f91a16a3ca03753183cb71d4e2f579f627e5e627e4280762f8a8ce1c20eabd9ac6a3fa46716a0575f79b487c821de2c9b7c4e29622b43b01e50d3196f5120d9a6e8cce4af4852a0eb48cda9ec3a4bfd2b07eda786f9b252079f081c54b84dccb33c9815d7f0d777a0b1d2ca6c8051e662589abe1d8d369538347f0f279897a128029be481c216615e80f891808080808080a0e701b13b6586c51266db06e2ecf5d2feb7f18f7cc2130dc8ddf942a27e9d5aa3a0eb43d68008d216e753fef198cf51077f5a89f406d9c244119d1643f0f2b19011a07edce1a61a9ba0549ef730adf66f30a4e8eafcda332a8cc04547d554a6bc1acd808080808080a026e499fde69be18e4abaf6c0def40aad830245fd0988d03f234ae3f5ef21ce7080e09e20fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace1224000000380200004c0400006006000074080000880a00009c0c00008f0d0000e20d0000f90211a0491f396d5d4768a01ee4282a3ab1127f2a4dc7d42e6c1dbb3e71ad4e9299f5e7a05b4645219e614b388ba9672452b40f291987b15e35bdbd3dfebfac9a085aeab2a0979ebca2a6a0df389fdfef5bfa4a31f2efa8d385bf2f43cd69c27e61165c3667a01bbe04543bb6bf8026ee3ec2a3ec3d6173a60c090008b59779f767e5769d7a0aa051d347ec61c7dde5c4149a943d0aa489544cbea511ed3d7f4fc6aaa52a420d3ea05358bc8e1e1f20e510887226d67efee73769d0b13ebb24a3e750c2214ef090d5a04df1c24ebf40befce60c8eb30a31894102881454fcdeb6f7b83e1fb916c953a8a00a85e04f30a4978712c58a825e7bd2f8a83731ab1aa3e234a0207e918790505fa0778a45437218486b7849aef890d483dcc2deb1423cb81e488b7be2921d505bf4a051dbaef3c3d3fe82bd1484954f38508f651d867f387f71dedbfb0f7355987ed5a061679f43f3db26673bb687c584f30ae3cff7261e71acf07ed1feaecae098fc79a099761ea7d94e01b14285b7ced7dcd1677fbd3ce093a627a8e1472074baad8bd9a0bb6d269fc61443aa28b14af0448aaf6f1f570477d8b4ef2eb7d31543202ce602a06c41cd2d1701b058853591a2f306ddcfd1e55d3e12011cadb100e7c525aaf460a0bcf37abaac566bb98e091575af403804dbb278fbfa5547de6805cbaa529ae137a0de9918a2a976a2b0b3f4aeff801fc79557426b19c15d414ddf8fae843785b0b780f90211a0be51f8518274d3e1beeaaa8cebe00428d5fd388e5a2a50404a4278e4dbb822b8a0c89fe8e16d89a82d5dcde02ef4a861c49549f23caa89c6772cd3e046924170a2a09d3e56c85bde8ff3e48f1029a3ed19f6cc1657ea7148396264117128916bf192a0dfe966436ccd76314d1ff27c3f314ba4bd67cdbb84bf6e7aa838407acd31f8c2a02a4659357dbb06d71dc46b900459e5bff6f3e49e4ef001cf4c8a2c62a7a227f9a02db2fff7623b55bef3e3e97d4596d35bf648209ed27a1dfa4574a950e76e9866a0e9b7cec39e827b538d84d697c774c2415d4c779b4633e3512b1b3b037e5be466a024a73f497d6080270bd263da5eb8c65a8495202d23880bb332a5fc7c4b18c4a8a08461ab3f12ab5d327b9496351467b25b27fab2e83c0c05618eb1fc20611f1958a022c7ff9562e90fee635a098995fb60330f313c683bf387b16e36f3ed01df69cca068c8d94e7b4b8a512cc108fd5fc723c53524f18d15c5aa354a5ed4076d5818a6a0d8162bc7fbbc17e616c126ba764cc4117ff3ec948a3bd588ea3287f8b901c4a7a0292d04df2f33b68e0033209c36035212c23581c8be47c5e0ea67e18fe8521f58a0e7ae2ece7c4c44a19a4d576904e39574030d6f739b450e75f36c59c83ae91829a0d577206430b32c22323816ba5309a85e5f71954e2bf5c755f5e33ee00f0de899a0f1f58b86f4addc6c095586a5333d6816b9b8a6d97b486798442863ea9615ae1380f90211a0256e27eecd0670e56ade58d99348a11f6fa4985d1f7b23143c61ee0a04a5a053a050b3d0ae7fdcad74ee9e322bb5b5fedb329db8b5bb95b281ee6d5b090314520ea07800c5365602d0c6eed658ee1056a0a2576809372757b1a42bafcc98438aebdba09bb7c947da462574a0fc0ae5608c25ac3272bb8fe3f2e619b20a10ced54738cfa0f73e7c44ce8ab0bc3495eba1aa7e2a0db44505d690e9d2daf4fab856cff999e2a022820f57846aa9ca4d0ebab020327b14178da53c2e006aaf5a22cc6648bc055ca0989235e562ad6e89656e7b8ccb301c300138d1eceddc2cfb724799a13f089783a0aa9b63d694d348219c0fb7f77b45e0e6835c99cda60a8f2030f0f712716b3751a0a34922189d09d23f5e0c8ff2f0b2407ff98a308cb3e8a9239cd4c776cee1e23ea06af9c10b566519fbbeb36dd1b3458b23fd25aeef27275a88df2d7662c2084ca9a05ce8b9e271738c6c52b1078ec04566681c8f11f679eeb2e83197431798cae230a05bfd293e2680e5c45fba826bb768e4d2cad9ab1451a712e98b0cdfba4d31b3f7a0f3ca96a33140e3e0e6ed23e857b6e5c5c7bdb77d7ef520aa4e10213d525bae84a003de0c0d867465dbb29e938fe08f3a508b26abacf7fdf8e02ffb485229195f14a0141a995d94148f08196179583fc853e65f18c338b1afe93b948701b795720960a01a35da701f908a90c16c4cc97de451ac2c35d52b019a58a078e61afe0b9a1cd480f90211a019c38310558d06bcbb4da68f64e61f2e6ee0f9bf3f6d643099e2e5c537294f6ba03a2ea461fdc547b6f72bc5d0b6f0a350aea0cb4b8ce1b036d123a5a00ec58b4aa07804bb6a71cf8c1153df5aa4ef84a409f26dd96c376e793da8cf9bcc802b4c26a076b61be012d6c7494b54c77a2dc6fcfe5568eef756d7a9b827c71ff25849dd87a0c8e8448bf1fbb0511823f5a92ff0febd7cad713967b5bbd0a4603d4ef2b9ef01a04c6d270a17b8fc136e0741940349c43d1e4ca894f191eefa13851da489de99d5a01076e9645c48dfed84933a8fae6016be4d47946f8105ccbc892ab0ae6ccf837aa0c6fea6b5dd5a7bae7e3c2ffae6841b292b613bf4c7421bdb90226f53e9eebc51a073b6c1de7658ec6f66fd52f1d66ac278ed0f3db37201981a9ca4b64a951fc52ba00bc5ecf44d423b4b6e0030d057939b739b85981c2bf2b4ad4f968bebd83cba7da0ed7e7a51c44fc09f230853a018c49eb711fbfa439904fd5ceb99c00b166f3ac7a005bf4f540bbc5bd6731dd89f4ff0a28c8ad8ee39d9611ef4bda0e2016bbd644ea0a1e0d3c4b952dce2233de45cca6e6c771e3f552ff6ff0194ff3d1b44d620683ea081e8d789f13dac46de04320c3a9c7d3e488ec667a619c21a020f95a41093e338a0dc9b8851a29da9247a4423fdcc689ae44c7843d31a891a067509780a2664c0a6a064f75b0a17b502154c6d316e7244bfde51a39b211484d311be73b80fbb93fd9c80f90211a09c0680faf2d7a9a79b5eedf6a8f846b95431cf91391973f6d1a6ec66dc7c3c24a02fd29886972cc78eab95ef19fd5cdd434e8d60c79b7d7aaff5b183469b848892a0cd246c92230812e63f13e8fddf2d0a0a976fde22e78256f9f3c7fd8219c71494a0f1d0b771957fe8d4c908d4ca730deca96c4ead47a9cde9648dd371da93bedbfca0de3910f22bd6ac087a1f778dfb20c7df43f7b3177e1abfe89d76d094dbd5753ba034b9341a9dad17687f5c22155e87ddbfea86dc0374e2071c6fac2a34fed5d7eea015eb41b2ece7cffbeaa040429aea2c73cf1c97a79282522f9eb25fcad1379b03a032214af47db0deb697b79ab12adc1b92cbace8395c8772544e803f33496c8073a01ac13a790f63114ce9bde82925bb5be8f4a09ee8ecf086d7675517248f8b8d4da04fe95c1a0f329cbf6b40efb608497e1d35f86919270a1c3b0a5b8f5c4675fbd6a0649e3ea0c951c4a16658f463755df800dd83f331f25a37275476877ce2eed44ea0157f094338c21f03a13a16793af96cc11a8f8a4a23b1f6105816833f603eabd0a086913ed8ad6c2c5d3d1e37b18bf7522aa080c04f3acd32a521efc7297653b658a0ed086a76c87f4654a164d1c4ec2626743cfa89bc1c4af41eba7369b42708c690a0c6730ce5e1df17601ac55b243bcba90a9f49dc835044bda9e2d7a76885b36598a0368943b40d006cdd58e0f576314c626fae7bc832d9451c5758a15c03e0bd08f680f90211a0515cb567f896e05d10003f6fff6b848448ecdfa82d6f854a8bf0058a57f53563a03661a6bd0c24511a05b40f8ae74c737f47d0df70bbf3c7a0816810ed90a84426a0c4d94bcf7df0b9c6707deb12770d992c1e60724ec2f40633a02ef1c4375326bba06af61ac19cd4496fb6d376dc38a8165a2561022866ce6fef41f8ade0fd2a407ba003a8b55e6cf12c06d0962f587bc293bca33303b7cc747f4ffbd17332cdbb33dda04f96708c335dd6364f8c5b4fc62edf32f5fd9a7107a99be1e9f6914b4093b88ca0e3981f17dbc1a00258432c0e34232f267d05e271590ee7a9783a51870f9873c8a03f2b45b47cdc12f020c55f6da3f7a348ec409de53a618b9cda4523c97d754b82a0935bc9189b81bb782d6b4d1451ab3d165b94a6fc4de4ab275528c06d9a128729a0c07c4048481c07b118748b2a30e851cf5c6c87af6212257f8845eb9ce07a02e4a0012fab999f402d6b302a07d3f7973ee8c357d5db8c7991903bd54d54d507cc23a05923155adc8cd3aceaf27c07858d290a3b85fc378fdd98e9f4f7f70d157f28baa0c55b2981a6fe08260cb6a076c76858d56aafdf255d0a12a2c50abe35d468c7e7a0e85b45f9a72f3abf2758687f2bdc33deab733aa4ef769bd5648f6a55ae1fb123a06ef38fec665b8eb25934622af1112b9a9d52408c94d2c0124d6e24b7ff4296c0a0867f6119f66c88787520dc8899d07d0e49598fa8dde1f33e611871eff6cd049680f8f1a0ca06c2b4c97d9941e56c3c752abe4c2b0b2cd162e22a5d25f61774dc453deedfa0344f34e01710ba897da06172844f373b281598b859086cf00c546594b955b87080a09bc4a42b6376f15f2639c98ad195b6fb948459cca93c568eacb33574b826a7af80a0525e7dd1bf391cf7df9ffaaa07093363a2c7a1c7d467d01403e368bd8c1f4e56808080808080a0758bf45f49922e3f1273d3e589753038a18ce3bbd961e3493f276eb7c5d04a3fa0235db60b9fecfc721d53cb6624da22433e765569a8312e86a6f0b47faf4a2a23a02f35f91fe878f56f1dd0b738bd12d9c8ed0f9b0f6be4146b66ae2c5625cc156b8080f85180a07f152c1e0fbe4b406b9a774b132347f174f02f3c2d6d1d4ad005c979996754b28080808080808080808080a06225fcc63b22b80301d9f2582014e450e91f9b329b7cc87ad16894722fff5296808080f8719d20a65bd257638cf8cf09b8238888947cc3c0bea2aa2cc3f1c4ac7a3002b851f84f018b02b4f32ee2f03d31ee3fbba046d5eb15d44b160805e80d05e2a47d434053e6c4b3ef9d1111773039e9586661a0d0a06b12ac47863b5c7be4185c2deaad1c61557033f56c7d4ea74429cbb25e23",
		ContentValueRetrieval: "0x04000000e09e20fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace12",
	},
	"ContractByteCode": {
		Selector:              "0x22",
		AddressHash:           "0x8679e8eda65bd257638cf8cf09b8238888947cc3c0bea2aa2cc3f1c4ac7a3002",
		CodeHash:              "0xd0a06b12ac47863b5c7be4185c2deaad1c61557033f56c7d4ea74429cbb25e23",
		ContentKey:            "0x228679e8eda65bd257638cf8cf09b8238888947cc3c0bea2aa2cc3f1c4ac7a3002d0a06b12ac47863b5c7be4185c2deaad1c61557033f56c7d4ea74429cbb25e23",
		ContentId:             "0x555a5d13dde0274db1fd43c32a81e10bc5ad35d62012beb55bca8afeefb31d32",
		ContentValueOffer:     "0x280000005c0c0000cf384012b91b081230cdf17a3f7dd370d8e67056058af6b272b3d54aa2714fac6060604052600436106100af576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806306fdde03146100b9578063095ea7b31461014757806318160ddd146101a157806323b872dd146101ca5780632e1a7d4d14610243578063313ce5671461026657806370a082311461029557806395d89b41146102e2578063a9059cbb14610370578063d0e30db0146103ca578063dd62ed3e146103d4575b6100b7610440565b005b34156100c457600080fd5b6100cc6104dd565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561010c5780820151818401526020810190506100f1565b50505050905090810190601f1680156101395780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561015257600080fd5b610187600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001909190505061057b565b604051808215151515815260200191505060405180910390f35b34156101ac57600080fd5b6101b461066d565b6040518082815260200191505060405180910390f35b34156101d557600080fd5b610229600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001909190505061068c565b604051808215151515815260200191505060405180910390f35b341561024e57600080fd5b61026460048080359060200190919050506109d9565b005b341561027157600080fd5b610279610b05565b604051808260ff1660ff16815260200191505060405180910390f35b34156102a057600080fd5b6102cc600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610b18565b6040518082815260200191505060405180910390f35b34156102ed57600080fd5b6102f5610b30565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561033557808201518184015260208101905061031a565b50505050905090810190601f1680156103625780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561037b57600080fd5b6103b0600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091908035906020019091905050610bce565b604051808215151515815260200191505060405180910390f35b6103d2610440565b005b34156103df57600080fd5b61042a600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610be3565b6040518082815260200191505060405180910390f35b34600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055503373ffffffffffffffffffffffffffffffffffffffff167fe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c346040518082815260200191505060405180910390a2565b60008054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156105735780601f1061054857610100808354040283529160200191610573565b820191906000526020600020905b81548152906001019060200180831161055657829003601f168201915b505050505081565b600081600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040518082815260200191505060405180910390a36001905092915050565b60003073ffffffffffffffffffffffffffffffffffffffff1631905090565b600081600360008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054101515156106dc57600080fd5b3373ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff16141580156107b457507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205414155b156108cf5781600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020541015151561084457600080fd5b81600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825403925050819055505b81600360008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254039250508190555081600360008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a3600190509392505050565b80600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410151515610a2757600080fd5b80600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825403925050819055503373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f193505050501515610ab457600080fd5b3373ffffffffffffffffffffffffffffffffffffffff167f7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65826040518082815260200191505060405180910390a250565b600260009054906101000a900460ff1681565b60036020528060005260406000206000915090505481565b60018054600181600116156101000203166002900480601f016020809104026020016040519081016040528092919081815260200182805460018160011615610100020316600290048015610bc65780601f10610b9b57610100808354040283529160200191610bc6565b820191906000526020600020905b815481529060010190602001808311610ba957829003601f168201915b505050505081565b6000610bdb33848461068c565b905092915050565b60046020528160005260406000206020528060005260406000206000915091505054815600a165627a7a72305820deb4c2ccab3c2fdca32ab3f46728389c2fe2c165d5fafa07661e4e004f6c344a002924000000380200004c0400006006000074080000880a00009c0c00008f0d0000e20d0000f90211a0491f396d5d4768a01ee4282a3ab1127f2a4dc7d42e6c1dbb3e71ad4e9299f5e7a05b4645219e614b388ba9672452b40f291987b15e35bdbd3dfebfac9a085aeab2a0979ebca2a6a0df389fdfef5bfa4a31f2efa8d385bf2f43cd69c27e61165c3667a01bbe04543bb6bf8026ee3ec2a3ec3d6173a60c090008b59779f767e5769d7a0aa051d347ec61c7dde5c4149a943d0aa489544cbea511ed3d7f4fc6aaa52a420d3ea05358bc8e1e1f20e510887226d67efee73769d0b13ebb24a3e750c2214ef090d5a04df1c24ebf40befce60c8eb30a31894102881454fcdeb6f7b83e1fb916c953a8a00a85e04f30a4978712c58a825e7bd2f8a83731ab1aa3e234a0207e918790505fa0778a45437218486b7849aef890d483dcc2deb1423cb81e488b7be2921d505bf4a051dbaef3c3d3fe82bd1484954f38508f651d867f387f71dedbfb0f7355987ed5a061679f43f3db26673bb687c584f30ae3cff7261e71acf07ed1feaecae098fc79a099761ea7d94e01b14285b7ced7dcd1677fbd3ce093a627a8e1472074baad8bd9a0bb6d269fc61443aa28b14af0448aaf6f1f570477d8b4ef2eb7d31543202ce602a06c41cd2d1701b058853591a2f306ddcfd1e55d3e12011cadb100e7c525aaf460a0bcf37abaac566bb98e091575af403804dbb278fbfa5547de6805cbaa529ae137a0de9918a2a976a2b0b3f4aeff801fc79557426b19c15d414ddf8fae843785b0b780f90211a0be51f8518274d3e1beeaaa8cebe00428d5fd388e5a2a50404a4278e4dbb822b8a0c89fe8e16d89a82d5dcde02ef4a861c49549f23caa89c6772cd3e046924170a2a09d3e56c85bde8ff3e48f1029a3ed19f6cc1657ea7148396264117128916bf192a0dfe966436ccd76314d1ff27c3f314ba4bd67cdbb84bf6e7aa838407acd31f8c2a02a4659357dbb06d71dc46b900459e5bff6f3e49e4ef001cf4c8a2c62a7a227f9a02db2fff7623b55bef3e3e97d4596d35bf648209ed27a1dfa4574a950e76e9866a0e9b7cec39e827b538d84d697c774c2415d4c779b4633e3512b1b3b037e5be466a024a73f497d6080270bd263da5eb8c65a8495202d23880bb332a5fc7c4b18c4a8a08461ab3f12ab5d327b9496351467b25b27fab2e83c0c05618eb1fc20611f1958a022c7ff9562e90fee635a098995fb60330f313c683bf387b16e36f3ed01df69cca068c8d94e7b4b8a512cc108fd5fc723c53524f18d15c5aa354a5ed4076d5818a6a0d8162bc7fbbc17e616c126ba764cc4117ff3ec948a3bd588ea3287f8b901c4a7a0292d04df2f33b68e0033209c36035212c23581c8be47c5e0ea67e18fe8521f58a0e7ae2ece7c4c44a19a4d576904e39574030d6f739b450e75f36c59c83ae91829a0d577206430b32c22323816ba5309a85e5f71954e2bf5c755f5e33ee00f0de899a0f1f58b86f4addc6c095586a5333d6816b9b8a6d97b486798442863ea9615ae1380f90211a0256e27eecd0670e56ade58d99348a11f6fa4985d1f7b23143c61ee0a04a5a053a050b3d0ae7fdcad74ee9e322bb5b5fedb329db8b5bb95b281ee6d5b090314520ea07800c5365602d0c6eed658ee1056a0a2576809372757b1a42bafcc98438aebdba09bb7c947da462574a0fc0ae5608c25ac3272bb8fe3f2e619b20a10ced54738cfa0f73e7c44ce8ab0bc3495eba1aa7e2a0db44505d690e9d2daf4fab856cff999e2a022820f57846aa9ca4d0ebab020327b14178da53c2e006aaf5a22cc6648bc055ca0989235e562ad6e89656e7b8ccb301c300138d1eceddc2cfb724799a13f089783a0aa9b63d694d348219c0fb7f77b45e0e6835c99cda60a8f2030f0f712716b3751a0a34922189d09d23f5e0c8ff2f0b2407ff98a308cb3e8a9239cd4c776cee1e23ea06af9c10b566519fbbeb36dd1b3458b23fd25aeef27275a88df2d7662c2084ca9a05ce8b9e271738c6c52b1078ec04566681c8f11f679eeb2e83197431798cae230a05bfd293e2680e5c45fba826bb768e4d2cad9ab1451a712e98b0cdfba4d31b3f7a0f3ca96a33140e3e0e6ed23e857b6e5c5c7bdb77d7ef520aa4e10213d525bae84a003de0c0d867465dbb29e938fe08f3a508b26abacf7fdf8e02ffb485229195f14a0141a995d94148f08196179583fc853e65f18c338b1afe93b948701b795720960a01a35da701f908a90c16c4cc97de451ac2c35d52b019a58a078e61afe0b9a1cd480f90211a019c38310558d06bcbb4da68f64e61f2e6ee0f9bf3f6d643099e2e5c537294f6ba03a2ea461fdc547b6f72bc5d0b6f0a350aea0cb4b8ce1b036d123a5a00ec58b4aa07804bb6a71cf8c1153df5aa4ef84a409f26dd96c376e793da8cf9bcc802b4c26a076b61be012d6c7494b54c77a2dc6fcfe5568eef756d7a9b827c71ff25849dd87a0c8e8448bf1fbb0511823f5a92ff0febd7cad713967b5bbd0a4603d4ef2b9ef01a04c6d270a17b8fc136e0741940349c43d1e4ca894f191eefa13851da489de99d5a01076e9645c48dfed84933a8fae6016be4d47946f8105ccbc892ab0ae6ccf837aa0c6fea6b5dd5a7bae7e3c2ffae6841b292b613bf4c7421bdb90226f53e9eebc51a073b6c1de7658ec6f66fd52f1d66ac278ed0f3db37201981a9ca4b64a951fc52ba00bc5ecf44d423b4b6e0030d057939b739b85981c2bf2b4ad4f968bebd83cba7da0ed7e7a51c44fc09f230853a018c49eb711fbfa439904fd5ceb99c00b166f3ac7a005bf4f540bbc5bd6731dd89f4ff0a28c8ad8ee39d9611ef4bda0e2016bbd644ea0a1e0d3c4b952dce2233de45cca6e6c771e3f552ff6ff0194ff3d1b44d620683ea081e8d789f13dac46de04320c3a9c7d3e488ec667a619c21a020f95a41093e338a0dc9b8851a29da9247a4423fdcc689ae44c7843d31a891a067509780a2664c0a6a064f75b0a17b502154c6d316e7244bfde51a39b211484d311be73b80fbb93fd9c80f90211a09c0680faf2d7a9a79b5eedf6a8f846b95431cf91391973f6d1a6ec66dc7c3c24a02fd29886972cc78eab95ef19fd5cdd434e8d60c79b7d7aaff5b183469b848892a0cd246c92230812e63f13e8fddf2d0a0a976fde22e78256f9f3c7fd8219c71494a0f1d0b771957fe8d4c908d4ca730deca96c4ead47a9cde9648dd371da93bedbfca0de3910f22bd6ac087a1f778dfb20c7df43f7b3177e1abfe89d76d094dbd5753ba034b9341a9dad17687f5c22155e87ddbfea86dc0374e2071c6fac2a34fed5d7eea015eb41b2ece7cffbeaa040429aea2c73cf1c97a79282522f9eb25fcad1379b03a032214af47db0deb697b79ab12adc1b92cbace8395c8772544e803f33496c8073a01ac13a790f63114ce9bde82925bb5be8f4a09ee8ecf086d7675517248f8b8d4da04fe95c1a0f329cbf6b40efb608497e1d35f86919270a1c3b0a5b8f5c4675fbd6a0649e3ea0c951c4a16658f463755df800dd83f331f25a37275476877ce2eed44ea0157f094338c21f03a13a16793af96cc11a8f8a4a23b1f6105816833f603eabd0a086913ed8ad6c2c5d3d1e37b18bf7522aa080c04f3acd32a521efc7297653b658a0ed086a76c87f4654a164d1c4ec2626743cfa89bc1c4af41eba7369b42708c690a0c6730ce5e1df17601ac55b243bcba90a9f49dc835044bda9e2d7a76885b36598a0368943b40d006cdd58e0f576314c626fae7bc832d9451c5758a15c03e0bd08f680f90211a0515cb567f896e05d10003f6fff6b848448ecdfa82d6f854a8bf0058a57f53563a03661a6bd0c24511a05b40f8ae74c737f47d0df70bbf3c7a0816810ed90a84426a0c4d94bcf7df0b9c6707deb12770d992c1e60724ec2f40633a02ef1c4375326bba06af61ac19cd4496fb6d376dc38a8165a2561022866ce6fef41f8ade0fd2a407ba003a8b55e6cf12c06d0962f587bc293bca33303b7cc747f4ffbd17332cdbb33dda04f96708c335dd6364f8c5b4fc62edf32f5fd9a7107a99be1e9f6914b4093b88ca0e3981f17dbc1a00258432c0e34232f267d05e271590ee7a9783a51870f9873c8a03f2b45b47cdc12f020c55f6da3f7a348ec409de53a618b9cda4523c97d754b82a0935bc9189b81bb782d6b4d1451ab3d165b94a6fc4de4ab275528c06d9a128729a0c07c4048481c07b118748b2a30e851cf5c6c87af6212257f8845eb9ce07a02e4a0012fab999f402d6b302a07d3f7973ee8c357d5db8c7991903bd54d54d507cc23a05923155adc8cd3aceaf27c07858d290a3b85fc378fdd98e9f4f7f70d157f28baa0c55b2981a6fe08260cb6a076c76858d56aafdf255d0a12a2c50abe35d468c7e7a0e85b45f9a72f3abf2758687f2bdc33deab733aa4ef769bd5648f6a55ae1fb123a06ef38fec665b8eb25934622af1112b9a9d52408c94d2c0124d6e24b7ff4296c0a0867f6119f66c88787520dc8899d07d0e49598fa8dde1f33e611871eff6cd049680f8f1a0ca06c2b4c97d9941e56c3c752abe4c2b0b2cd162e22a5d25f61774dc453deedfa0344f34e01710ba897da06172844f373b281598b859086cf00c546594b955b87080a09bc4a42b6376f15f2639c98ad195b6fb948459cca93c568eacb33574b826a7af80a0525e7dd1bf391cf7df9ffaaa07093363a2c7a1c7d467d01403e368bd8c1f4e56808080808080a0758bf45f49922e3f1273d3e589753038a18ce3bbd961e3493f276eb7c5d04a3fa0235db60b9fecfc721d53cb6624da22433e765569a8312e86a6f0b47faf4a2a23a02f35f91fe878f56f1dd0b738bd12d9c8ed0f9b0f6be4146b66ae2c5625cc156b8080f85180a07f152c1e0fbe4b406b9a774b132347f174f02f3c2d6d1d4ad005c979996754b28080808080808080808080a06225fcc63b22b80301d9f2582014e450e91f9b329b7cc87ad16894722fff5296808080f8719d20a65bd257638cf8cf09b8238888947cc3c0bea2aa2cc3f1c4ac7a3002b851f84f018b02b4f32ee2f03d31ee3fbba046d5eb15d44b160805e80d05e2a47d434053e6c4b3ef9d1111773039e9586661a0d0a06b12ac47863b5c7be4185c2deaad1c61557033f56c7d4ea74429cbb25e23",
		ContentValueRetrieval: "0x040000006060604052600436106100af576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806306fdde03146100b9578063095ea7b31461014757806318160ddd146101a157806323b872dd146101ca5780632e1a7d4d14610243578063313ce5671461026657806370a082311461029557806395d89b41146102e2578063a9059cbb14610370578063d0e30db0146103ca578063dd62ed3e146103d4575b6100b7610440565b005b34156100c457600080fd5b6100cc6104dd565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561010c5780820151818401526020810190506100f1565b50505050905090810190601f1680156101395780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561015257600080fd5b610187600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001909190505061057b565b604051808215151515815260200191505060405180910390f35b34156101ac57600080fd5b6101b461066d565b6040518082815260200191505060405180910390f35b34156101d557600080fd5b610229600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001909190505061068c565b604051808215151515815260200191505060405180910390f35b341561024e57600080fd5b61026460048080359060200190919050506109d9565b005b341561027157600080fd5b610279610b05565b604051808260ff1660ff16815260200191505060405180910390f35b34156102a057600080fd5b6102cc600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610b18565b6040518082815260200191505060405180910390f35b34156102ed57600080fd5b6102f5610b30565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561033557808201518184015260208101905061031a565b50505050905090810190601f1680156103625780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561037b57600080fd5b6103b0600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091908035906020019091905050610bce565b604051808215151515815260200191505060405180910390f35b6103d2610440565b005b34156103df57600080fd5b61042a600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610be3565b6040518082815260200191505060405180910390f35b34600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055503373ffffffffffffffffffffffffffffffffffffffff167fe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c346040518082815260200191505060405180910390a2565b60008054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156105735780601f1061054857610100808354040283529160200191610573565b820191906000526020600020905b81548152906001019060200180831161055657829003601f168201915b505050505081565b600081600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040518082815260200191505060405180910390a36001905092915050565b60003073ffffffffffffffffffffffffffffffffffffffff1631905090565b600081600360008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054101515156106dc57600080fd5b3373ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff16141580156107b457507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205414155b156108cf5781600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020541015151561084457600080fd5b81600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825403925050819055505b81600360008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254039250508190555081600360008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a3600190509392505050565b80600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410151515610a2757600080fd5b80600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825403925050819055503373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f193505050501515610ab457600080fd5b3373ffffffffffffffffffffffffffffffffffffffff167f7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65826040518082815260200191505060405180910390a250565b600260009054906101000a900460ff1681565b60036020528060005260406000206000915090505481565b60018054600181600116156101000203166002900480601f016020809104026020016040519081016040528092919081815260200182805460018160011615610100020316600290048015610bc65780601f10610b9b57610100808354040283529160200191610bc6565b820191906000526020600020905b815481529060010190602001808311610ba957829003601f168201915b505050505081565b6000610bdb33848461068c565b905092915050565b60046020528160005260406000206020528060005260406000206000915091505054815600a165627a7a72305820deb4c2ccab3c2fdca32ab3f46728389c2fe2c165d5fafa07661e4e004f6c344a0029",
	},
}
