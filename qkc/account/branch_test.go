// Ported verbatim from github.com/QuarkChain/goquarkchain/account (byte-compatible).

package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
)

type JSONStruct struct {
}

func NewJSONStruct() *JSONStruct {
	return &JSONStruct{}
}

func (jst *JSONStruct) Load(filename string, v interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file failed err %v", err)
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("Unmarshal data failed err %v", err)
	}
	return nil
}

type BranchTestStruct struct {
	Size           uint32 `json:"size"`
	Key            uint32 `json:"key"`
	TestIsInBranch uint32 `json:"testIsinBranch"`
	ChainID        uint32 `json:"chainID"`
	GetSize        uint32 `json:"getsize"`
	FullShardID    uint32 `json:"fullshardid"`
	ShardID        uint32 `json:"shardid"`
	IsInBranch     bool   `json:"isinbranch"`
}

func CheckBranchUnitTest(data BranchTestStruct) bool {
	tempBranch, err := CreatBranch(0, data.Size, data.Key) //create branch depend on special size ans key
	if err != nil {
		fmt.Printf("CreatBranch err %v\n", err)
		return false
	}
	if tempBranch.GetChainID() != data.ChainID { //checkGetChainID
		fmt.Printf("chainId is not match: unexcepted %d,except %d\n", tempBranch.GetChainID(), data.ChainID)
		return false
	}
	if tempBranch.GetShardSize() != data.GetSize { //checkGetShardSize
		fmt.Printf("ShardSze is not match: unexcepted %d,excepted %d\n", tempBranch.GetShardSize(), data.GetSize)
		return false
	}

	if tempBranch.GetShardID() != data.ShardID { //checkGetShardID
		fmt.Printf("shardid is not match: unexcepted %d,excepted %d\n", tempBranch.GetShardID(), data.ShardID)
		return false
	}

	if tempBranch.IsInBranch(data.TestIsInBranch) != data.IsInBranch { //checkIsInBranch
		fmt.Printf("isInBranch is not match: unexcepted %t,excepted %t\n", tempBranch.IsInBranch(data.TestIsInBranch), data.IsInBranch)
		return false
	}

	if tempBranch.GetFullShardID() != data.FullShardID { //checkGetFullShardID
		fmt.Printf("full shard id is not match: unexcepted %d,excepted %d\n", tempBranch.GetFullShardID(), data.FullShardID)
		return false
	}
	return true
}

// 1.python generate testdata
//
//	1.1 branch from size and key
//
// 2.go.exe to check
//
//	2.1 checkGetChainID
//	2.2 checkGetShardSize
//	2.3 checkGetShardID
//	2.4 checkIsInBranch
//	2.5 checkGetFullShardID
func TestBranch(t *testing.T) {
	JSONParse := NewJSONStruct()
	v := []BranchTestStruct{}
	err := JSONParse.Load("./testdata/testBranch.json", &v) //analysis test data
	if err != nil {
		panic(err)
	}
	count := 0
	for _, v := range v {
		status := CheckBranchUnitTest(v) //unit test
		if status == false {
			panic(errors.New("testFailed"))
		}
		count++
	}
	fmt.Println("TestBranch:success test num:", count)
}

func TestCreatBranch(t *testing.T) {
	b, err := CreatBranch(3, 8, 6)
	if err != nil {
		t.Fatal("CreatBranch error: ", err)
	}

	check := func(f string, got, want interface{}) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mismatch: got %v, want %v", f, got, want)
		}
	}
	check("GetChainID", int(b.GetChainID()), 3)
	check("GetShardSize", int(b.GetShardSize()), 8)
	check("GetShardID", int(b.GetShardID()), 6)

}
