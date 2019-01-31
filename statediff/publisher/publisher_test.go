package publisher_test

import (
	"bytes"
	"encoding/csv"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/statediff"
	"github.com/ethereum/go-ethereum/statediff/builder"
	p "github.com/ethereum/go-ethereum/statediff/publisher"
	"github.com/ethereum/go-ethereum/statediff/testhelpers"
	"github.com/pkg/errors"
)

var (
	tempDir        = os.TempDir()
	testFilePrefix = "test-statediff"
	publisher      p.Publisher
	dir            string
	err            error
)

var expectedCreatedAccountRow = []string{
	strconv.FormatInt(testhelpers.BlockNumber, 10),
	testhelpers.BlockHash,
	"created",
	testhelpers.CodeHash,
	strconv.FormatUint(testhelpers.NewNonceValue, 10),
	strconv.FormatInt(testhelpers.NewBalanceValue, 10),
	testhelpers.ContractRoot,
	testhelpers.StoragePath,
	testhelpers.ContractAddress,
	"0000000000000000000000000000000000000000000000000000000000000001",
	testhelpers.StorageValue,
}

var expectedCreatedAccountWithoutStorageUpdateRow = []string{
	strconv.FormatInt(testhelpers.BlockNumber, 10),
	testhelpers.BlockHash,
	"created",
	testhelpers.CodeHash,
	strconv.FormatUint(testhelpers.NewNonceValue, 10),
	strconv.FormatInt(testhelpers.NewBalanceValue, 10),
	testhelpers.ContractRoot,
	"",
	testhelpers.AnotherContractAddress,
	"",
	"",
}

var expectedUpdatedAccountRow = []string{
	strconv.FormatInt(testhelpers.BlockNumber, 10),
	testhelpers.BlockHash,
	"updated",
	testhelpers.CodeHash,
	strconv.FormatUint(testhelpers.NewNonceValue, 10),
	strconv.FormatInt(testhelpers.NewBalanceValue, 10),
	testhelpers.ContractRoot,
	testhelpers.StoragePath,
	testhelpers.ContractAddress,
	"0000000000000000000000000000000000000000000000000000000000000001",
	testhelpers.StorageValue,
}

var expectedDeletedAccountRow = []string{
	strconv.FormatInt(testhelpers.BlockNumber, 10),
	testhelpers.BlockHash,
	"deleted",
	testhelpers.CodeHash,
	strconv.FormatUint(testhelpers.NewNonceValue, 10),
	strconv.FormatInt(testhelpers.NewBalanceValue, 10),
	testhelpers.ContractRoot,
	testhelpers.StoragePath,
	testhelpers.ContractAddress,
	"0000000000000000000000000000000000000000000000000000000000000001",
	testhelpers.StorageValue,
}

func TestPublisher(t *testing.T) {
	dir, err = ioutil.TempDir(tempDir, testFilePrefix)
	if err != nil {
		t.Error(err)
	}
	config := statediff.Config{
		Path: dir,
		Mode: statediff.CSV,
	}
	publisher, err = p.NewPublisher(config)
	if err != nil {
		t.Error(err)
	}

	type Test func(t *testing.T)

	var tests = []Test{
		testFileName,
		testColumnHeaders,
		testAccountDiffs,
		testWhenNoDiff,
		testDefaultPublisher,
		testDefaultDirectory,
	}

	for _, test := range tests {
		test(t)
		err := removeFilesFromDir(dir)
		if err != nil {
			t.Errorf("Error removing files from temp dir: %s", dir)
		}
	}
}

func removeFilesFromDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return err
	}

	for _, file := range files {
		err = os.RemoveAll(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func testFileName(t *testing.T) {
	fileName, err := publisher.PublishStateDiff(&testhelpers.TestStateDiff)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	if !strings.HasPrefix(fileName, dir) {
		t.Errorf(testhelpers.TestFailureFormatString, t.Name(), dir, fileName)
	}
	blockNumberWithFileExt := strconv.FormatInt(testhelpers.BlockNumber, 10) + ".csv"
	if !strings.HasSuffix(fileName, blockNumberWithFileExt) {
		t.Errorf(testhelpers.TestFailureFormatString, t.Name(), blockNumberWithFileExt, fileName)
	}
}

func testColumnHeaders(t *testing.T) {
	_, err = publisher.PublishStateDiff(&testhelpers.TestStateDiff)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	file, err := getTestDiffFile(dir)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	lines, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if len(lines) < 1 {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if !equals(lines[0], p.Headers) {
		t.Error()
	}
}

func testAccountDiffs(t *testing.T) {
	// it persists the created, updated and deleted account diffs to a CSV file
	_, err = publisher.PublishStateDiff(&testhelpers.TestStateDiff)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	file, err := getTestDiffFile(dir)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	lines, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if len(lines) <= 3 {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if !equals(lines[1], expectedCreatedAccountRow) {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if !equals(lines[2], expectedCreatedAccountWithoutStorageUpdateRow) {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if !equals(lines[3], expectedUpdatedAccountRow) {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if !equals(lines[4], expectedDeletedAccountRow) {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
}

func testWhenNoDiff(t *testing.T) {
	//it creates an empty CSV when there is no diff
	emptyDiff := builder.StateDiff{}
	_, err = publisher.PublishStateDiff(&emptyDiff)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	file, err := getTestDiffFile(dir)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	lines, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	if !equals(len(lines), 1) {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
}

func testDefaultPublisher(t *testing.T) {
	//it defaults to publishing state diffs to a CSV file when no mode is configured
	config := statediff.Config{Path: dir}
	publisher, err = p.NewPublisher(config)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	_, err = publisher.PublishStateDiff(&testhelpers.TestStateDiff)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	file, err := getTestDiffFile(dir)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	lines, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if !equals(len(lines), 5) {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if !equals(lines[0], p.Headers) {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
}

func testDefaultDirectory(t *testing.T) {
	//it defaults to publishing CSV files in the current directory when no path is configured
	config := statediff.Config{}
	publisher, err = p.NewPublisher(config)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	err := os.Chdir(dir)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	_, err = publisher.PublishStateDiff(&testhelpers.TestStateDiff)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	file, err := getTestDiffFile(dir)
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}

	lines, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if !equals(len(lines), 5) {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
	if !equals(lines[0], p.Headers) {
		t.Errorf(testhelpers.ErrorFormatString, t.Name(), err)
	}
}

func getTestDiffFile(dir string) (*os.File, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("There are 0 files.")
	}

	fileName := files[0].Name()
	filePath := filepath.Join(dir, fileName)

	return os.Open(filePath)
}

func equals(actual, expected interface{}) (success bool) {
	if actualByteSlice, ok := actual.([]byte); ok {
		if expectedByteSlice, ok := expected.([]byte); ok {
			return bytes.Equal(actualByteSlice, expectedByteSlice)
		}
	}

	return reflect.DeepEqual(actual, expected)
}
