// Ported verbatim from github.com/QuarkChain/goquarkchain/account (byte-compatible).

package account

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

type AccountTestStruct struct {
	TKey       string `json:"tKey"`
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"`
	UUID       string `json:"uuid"`
}

// get abs file path
func GetFilesAndDirs(dirPth string) (files []string, err error) {
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil { //read dir err
		return nil, err
	}
	PthSep := string(os.PathSeparator)
	for _, fi := range dir {
		if fi.IsDir() {
			continue
		} else {
			ok := strings.HasSuffix(fi.Name(), ".json")
			if ok {
				files = append(files, dirPth+PthSep+fi.Name()) // is keystore file
			}
		}
	}
	return files, nil
}

func CheckAccountUnitTest(data AccountTestStruct, pathAll []string) bool {
	accountPath := ""
	for _, v := range pathAll { //find keystore file depend on uuid
		if strings.Contains(v, data.UUID) {
			accountPath = v
		}
	}
	if accountPath == "" { //can not find file
		fmt.Println("can not find path")
		return false
	}

	account, err := Load(accountPath, data.TKey)
	if err != nil { //load err
		fmt.Println("Load err", err)
		return false
	}
	address := account.Address()
	if "0x"+data.Address != address { //address is not match
		fmt.Printf("address is not match unexcepted %s,excepted %s\n", data.Address, address)
		return false
	}
	if data.UUID != account.UUID().String() { //uuid is not match
		fmt.Println("uuid is not match")
		return false
	}
	if data.PrivateKey != account.PrivateKey() { //privateKey is not match
		fmt.Println("privateKey is not match")
		return false
	}
	return true
}

// 1.python generate keystore and it's value
// 2.go.exe test it
func TestAccount(t *testing.T) {
	files, err := GetFilesAndDirs("./testdata/keystore/") //read test keystore file
	JSONParse := NewJSONStruct()
	data := []AccountTestStruct{}
	err = JSONParse.Load("./testdata/testAccount.json", &data) //analysis test data
	if err != nil {
		panic(err)
	}

	count := 0
	for _, v := range data {
		err := CheckAccountUnitTest(v, files) //unit test
		if err == false {
			panic(-1)
		}
		count++
	}
	fmt.Println("TestAccount:success test num:", count)
}

// 1.dump file
// 2.use python to load and check it's value
func TestDump(t *testing.T) {
	account, err := NewAccountWithoutKey()
	fmt.Println("err", err)
	fmt.Println("id", account.ID.String())
	fmt.Println("Private", account.PrivateKey())
	fmt.Println("Address", account.Address())
	_, err = account.Dump("test_password", true, true, "")
	fmt.Println("dump err", err)
}
