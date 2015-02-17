package ini

import (
	"io/ioutil"
	"testing"
)

const (
	exampleStr = `key1 = true

[section1]
key1 = value2
key2 = 5
key3 = 1.3

[section2]
key1 = 5

`
)

var (
	dict Dict
	err  error
)

func init() {
	dict, err = Load("example.ini")
}

func TestLoad(t *testing.T) {
	if err != nil {
		t.Error("Example: load error:", err)
	}
}

func TestWrite(t *testing.T) {
	d, err := Load("empty.ini")
	if err != nil {
		t.Error("Example: load error:", err)
	}
	d.SetString("", "key", "value")
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Error("Write: Couldn't create temp file.", err)
	}
	err = Write(tempFile.Name(), &d)
	if err != nil {
		t.Error("Write: Couldn't write to temp config file.", err)
	}
	contents, err := ioutil.ReadFile(tempFile.Name())
	if err != nil {
		t.Error("Write: Couldn't read from the temp config file.", err)
	}
	if string(contents) != "key = value\n\n" {
		t.Error("Write: Contents of the config file doesn't match the expected.")
	}
}

func TestGetBool(t *testing.T) {
	b, found := dict.GetBool("pizza", "ham")
	if !found || !b {
		t.Error("Example: parse error for key ham of section pizza.")
	}
	b, found = dict.GetBool("pizza", "mushrooms")
	if !found || !b {
		t.Error("Example: parse error for key mushrooms of section pizza.")
	}
	b, found = dict.GetBool("pizza", "capres")
	if !found || b {
		t.Error("Example: parse error for key capres of section pizza.")
	}
	b, found = dict.GetBool("pizza", "cheese")
	if !found || b {
		t.Error("Example: parse error for key cheese of section pizza.")
	}
}

func TestGetStringIntAndDouble(t *testing.T) {
	str, found := dict.GetString("wine", "grape")
	if !found || str != "Cabernet Sauvignon" {
		t.Error("Example: parse error for key grape of section wine.")
	}
	i, found := dict.GetInt("wine", "year")
	if !found || i != 1989 {
		t.Error("Example: parse error for key year of section wine.")
	}
	str, found = dict.GetString("wine", "country")
	if !found || str != "Spain" {
		t.Error("Example: parse error for key grape of section wine.")
	}
	d, found := dict.GetDouble("wine", "alcohol")
	if !found || d != 12.5 {
		t.Error("Example: parse error for key grape of section wine.")
	}
}

func TestSetBoolAndStringAndIntAndDouble(t *testing.T) {
	dict.SetBool("pizza", "ham", false)
	b, found := dict.GetBool("pizza", "ham")
	if !found || b {
		t.Error("Example: bool set error for key ham of section pizza.")
	}
	dict.SetString("pizza", "ham", "no")
	n, found := dict.GetString("pizza", "ham")
	if !found || n != "no" {
		t.Error("Example: string set error for key ham of section pizza.")
	}
	dict.SetInt("wine", "year", 1978)
	i, found := dict.GetInt("wine", "year")
	if !found || i != 1978 {
		t.Error("Example: int set error for key year of section wine.")
	}
	dict.SetDouble("wine", "not-exists", 5.6)
	d, found := dict.GetDouble("wine", "not-exists")
	if !found || d != 5.6 {
		t.Error("Example: float set error for not existing key for wine.")
	}
}

func TestDelete(t *testing.T) {
	d, err := Load("empty.ini")
	if err != nil {
		t.Error("Example: load error:", err)
	}
	d.SetString("pizza", "ham", "yes")
	d.Delete("pizza", "ham")
	_, found := d.GetString("pizza", "ham")
	if found {
		t.Error("Example: delete error for key ham of section pizza.")
	}
	if len(d.GetSections()) > 1 {
		t.Error("Only a single section should exist after deletion.")
	}
}

func TestGetNotExist(t *testing.T) {
	_, found := dict.GetString("not", "exist")
	if found {
		t.Error("There is no key exist of section not.")
	}
}

func TestGetSections(t *testing.T) {
	sections := dict.GetSections()
	if len(sections) != 3 {
		t.Error("The number of sections is wrong:", len(sections))
	}
	for _, section := range sections {
		if section != "" && section != "pizza" && section != "wine" {
			t.Errorf("Section '%s' should not be exist.", section)
		}
	}
}

func TestString(t *testing.T) {
	d, err := Load("empty.ini")
	if err != nil {
		t.Error("Example: load error:", err)
	}
	d.SetBool("", "key1", true)
	d.SetString("section1", "key1", "value2")
	d.SetInt("section1", "key2", 5)
	d.SetDouble("section1", "key3", 1.3)
	d.SetDouble("section2", "key1", 5.0)
	if d.String() != exampleStr {
		t.Errorf("Dict cannot be stringified as expected.")
	}
}
