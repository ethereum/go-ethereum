package trie

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	checker "gopkg.in/check.v1"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
)

const LONG_WORD = "1234567890abcdefghijklmnopqrstuvwxxzABCEFGHIJKLMNOPQRSTUVWXYZ"

type TrieSuite struct {
	db   *MemDatabase
	trie *Trie
}

type MemDatabase struct {
	db map[string][]byte
}

func NewMemDatabase() (*MemDatabase, error) {
	db := &MemDatabase{db: make(map[string][]byte)}
	return db, nil
}
func (db *MemDatabase) Put(key []byte, value []byte) {
	db.db[string(key)] = value
}
func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	return db.db[string(key)], nil
}
func (db *MemDatabase) Delete(key []byte) error {
	delete(db.db, string(key))
	return nil
}
func (db *MemDatabase) Print()              {}
func (db *MemDatabase) Close()              {}
func (db *MemDatabase) LastKnownTD() []byte { return nil }

func NewTrie() (*MemDatabase, *Trie) {
	db, _ := NewMemDatabase()
	return db, New(db, "")
}

func (s *TrieSuite) SetUpTest(c *checker.C) {
	s.db, s.trie = NewTrie()
}

func (s *TrieSuite) TestTrieSync(c *checker.C) {
	s.trie.Update("dog", LONG_WORD)
	c.Assert(s.db.db, checker.HasLen, 0, checker.Commentf("Expected no data in database"))
	s.trie.Sync()
	c.Assert(s.db.db, checker.HasLen, 3)
}

func (s *TrieSuite) TestTrieDirtyTracking(c *checker.C) {
	s.trie.Update("dog", LONG_WORD)
	c.Assert(s.trie.cache.IsDirty, checker.Equals, true, checker.Commentf("Expected no data in database"))

	s.trie.Sync()
	c.Assert(s.trie.cache.IsDirty, checker.Equals, false, checker.Commentf("Expected trie to be dirty"))

	s.trie.Update("test", LONG_WORD)
	s.trie.cache.Undo()
	c.Assert(s.trie.cache.IsDirty, checker.Equals, false)
}

func (s *TrieSuite) TestTrieReset(c *checker.C) {
	s.trie.Update("cat", LONG_WORD)
	c.Assert(s.trie.cache.nodes, checker.HasLen, 1, checker.Commentf("Expected cached nodes"))

	s.trie.cache.Undo()
	c.Assert(s.trie.cache.nodes, checker.HasLen, 0, checker.Commentf("Expected no nodes after undo"))
}

func (s *TrieSuite) TestTrieGet(c *checker.C) {
	s.trie.Update("cat", LONG_WORD)
	x := s.trie.Get("cat")
	c.Assert(x, checker.DeepEquals, LONG_WORD)
}

func (s *TrieSuite) TestTrieUpdating(c *checker.C) {
	s.trie.Update("cat", LONG_WORD)
	s.trie.Update("cat", LONG_WORD+"1")
	x := s.trie.Get("cat")
	c.Assert(x, checker.DeepEquals, LONG_WORD+"1")
}

func (s *TrieSuite) TestTrieCmp(c *checker.C) {
	_, trie1 := NewTrie()
	_, trie2 := NewTrie()

	trie1.Update("doge", LONG_WORD)
	trie2.Update("doge", LONG_WORD)
	c.Assert(trie1, checker.DeepEquals, trie2)

	trie1.Update("dog", LONG_WORD)
	trie2.Update("cat", LONG_WORD)
	c.Assert(trie1, checker.Not(checker.DeepEquals), trie2)
}

func (s *TrieSuite) TestTrieDelete(c *checker.C) {
	s.trie.Update("cat", LONG_WORD)
	exp := s.trie.Root
	s.trie.Update("dog", LONG_WORD)
	s.trie.Delete("dog")
	c.Assert(s.trie.Root, checker.DeepEquals, exp)

	s.trie.Update("dog", LONG_WORD)
	exp = s.trie.Root
	s.trie.Update("dude", LONG_WORD)
	s.trie.Delete("dude")
	c.Assert(s.trie.Root, checker.DeepEquals, exp)
}

func (s *TrieSuite) TestTrieDeleteWithValue(c *checker.C) {
	s.trie.Update("c", LONG_WORD)
	exp := s.trie.Root
	s.trie.Update("ca", LONG_WORD)
	s.trie.Update("cat", LONG_WORD)
	s.trie.Delete("ca")
	s.trie.Delete("cat")
	c.Assert(s.trie.Root, checker.DeepEquals, exp)
}

func (s *TrieSuite) TestTriePurge(c *checker.C) {
	s.trie.Update("c", LONG_WORD)
	s.trie.Update("ca", LONG_WORD)
	s.trie.Update("cat", LONG_WORD)

	lenBefore := len(s.trie.cache.nodes)
	it := s.trie.NewIterator()
	num := it.Purge()
	c.Assert(num, checker.Equals, 3)
	c.Assert(len(s.trie.cache.nodes), checker.Equals, lenBefore)
}

func h(str string) string {
	d, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}

	return string(d)
}

func get(in string) (out string) {
	if len(in) > 2 && in[:2] == "0x" {
		out = h(in[2:])
	} else {
		out = in
	}

	return
}

type TrieTest struct {
	Name string
	In   map[string]string
	Root string
}

func CreateTest(name string, data []byte) (TrieTest, error) {
	t := TrieTest{Name: name}
	err := json.Unmarshal(data, &t)
	if err != nil {
		return TrieTest{}, fmt.Errorf("%v", err)
	}

	return t, nil
}

func CreateTests(uri string, cb func(TrieTest)) map[string]TrieTest {
	resp, err := http.Get(uri)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)

	var objmap map[string]*json.RawMessage
	err = json.Unmarshal(data, &objmap)
	if err != nil {
		panic(err)
	}

	tests := make(map[string]TrieTest)
	for name, testData := range objmap {
		test, err := CreateTest(name, *testData)
		if err != nil {
			panic(err)
		}

		if cb != nil {
			cb(test)
		}
		tests[name] = test
	}

	return tests
}

func RandomData() [][]string {
	data := [][]string{
		{"0x000000000000000000000000ec4f34c97e43fbb2816cfd95e388353c7181dab1", "0x4e616d6552656700000000000000000000000000000000000000000000000000"},
		{"0x0000000000000000000000000000000000000000000000000000000000000045", "0x22b224a1420a802ab51d326e29fa98e34c4f24ea"},
		{"0x0000000000000000000000000000000000000000000000000000000000000046", "0x67706c2076330000000000000000000000000000000000000000000000000000"},
		{"0x000000000000000000000000697c7b8c961b56f675d570498424ac8de1a918f6", "0x6f6f6f6820736f2067726561742c207265616c6c6c793f000000000000000000"},
		{"0x0000000000000000000000007ef9e639e2733cb34e4dfc576d4b23f72db776b2", "0x4655474156000000000000000000000000000000000000000000000000000000"},
		{"0x6f6f6f6820736f2067726561742c207265616c6c6c793f000000000000000000", "0x697c7b8c961b56f675d570498424ac8de1a918f6"},
		{"0x4655474156000000000000000000000000000000000000000000000000000000", "0x7ef9e639e2733cb34e4dfc576d4b23f72db776b2"},
		{"0x4e616d6552656700000000000000000000000000000000000000000000000000", "0xec4f34c97e43fbb2816cfd95e388353c7181dab1"},
	}

	var c [][]string
	for len(data) != 0 {
		e := rand.Intn(len(data))
		c = append(c, data[e])

		copy(data[e:], data[e+1:])
		data[len(data)-1] = nil
		data = data[:len(data)-1]
	}

	return c
}

const MaxTest = 1000

// This test insert data in random order and seeks to find indifferences between the different tries
func (s *TrieSuite) TestRegression(c *checker.C) {
	rand.Seed(time.Now().Unix())

	roots := make(map[string]int)
	for i := 0; i < MaxTest; i++ {
		_, trie := NewTrie()
		data := RandomData()

		for _, test := range data {
			trie.Update(test[0], test[1])
		}
		trie.Delete("0x4e616d6552656700000000000000000000000000000000000000000000000000")

		roots[string(trie.Root.([]byte))] += 1
	}

	c.Assert(len(roots) <= 1, checker.Equals, true)
	// if len(roots) > 1 {
	// 	for root, num := range roots {
	// 		t.Errorf("%x => %d\n", root, num)
	// 	}
	// }
}

func (s *TrieSuite) TestDelete(c *checker.C) {
	s.trie.Update("a", "jeffreytestlongstring")
	s.trie.Update("aa", "otherstring")
	s.trie.Update("aaa", "othermorestring")
	s.trie.Update("aabbbbccc", "hithere")
	s.trie.Update("abbcccdd", "hstanoehutnaheoustnh")
	s.trie.Update("rnthaoeuabbcccdd", "hstanoehutnaheoustnh")
	s.trie.Update("rneuabbcccdd", "hstanoehutnaheoustnh")
	s.trie.Update("rneuabboeusntahoeucccdd", "hstanoehutnaheoustnh")
	s.trie.Update("rnxabboeusntahoeucccdd", "hstanoehutnaheoustnh")
	s.trie.Delete("aaboaestnuhbccc")
	s.trie.Delete("a")
	s.trie.Update("a", "nthaonethaosentuh")
	s.trie.Update("c", "shtaosntehua")
	s.trie.Delete("a")
	s.trie.Update("aaaa", "testmegood")

	_, t2 := NewTrie()
	s.trie.NewIterator().Each(func(key string, v *ethutil.Value) {
		if key == "aaaa" {
			t2.Update(key, v.Str())
		} else {
			t2.Update(key, v.Str())
		}
	})

	a := ethutil.NewValue(s.trie.Root).Bytes()
	b := ethutil.NewValue(t2.Root).Bytes()

	c.Assert(a, checker.DeepEquals, b)
}

func (s *TrieSuite) TestTerminator(c *checker.C) {
	key := CompactDecode("hello")
	c.Assert(HasTerm(key), checker.Equals, true, checker.Commentf("Expected %v to have a terminator", key))
}

func (s *TrieSuite) TestIt(c *checker.C) {
	s.trie.Update("cat", "cat")
	s.trie.Update("doge", "doge")
	s.trie.Update("wallace", "wallace")
	it := s.trie.Iterator()

	inputs := []struct {
		In, Out string
	}{
		{"", "cat"},
		{"bobo", "cat"},
		{"c", "cat"},
		{"car", "cat"},
		{"catering", "doge"},
		{"w", "wallace"},
		{"wallace123", ""},
	}

	for _, test := range inputs {
		res := string(it.Next(test.In))
		c.Assert(res, checker.Equals, test.Out)
	}
}

func (s *TrieSuite) TestBeginsWith(c *checker.C) {
	a := CompactDecode("hello")
	b := CompactDecode("hel")

	c.Assert(BeginsWith(a, b), checker.Equals, false)
	c.Assert(BeginsWith(b, a), checker.Equals, true)
}

func TestItems(t *testing.T) {
	_, trie := NewTrie()
	trie.Update("A", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	exp := "d23786fb4a010da3ce639d66d5e904a11dbc02746d1ce25029e53290cabf28ab"
	if bytes.Compare(trie.GetRoot(), ethutil.Hex2Bytes(exp)) != 0 {
		t.Errorf("Expected root to be %s but got", exp, trie.GetRoot())
	}
}

/*
func TestRndCase(t *testing.T) {
	_, trie := NewTrie()

	data := []struct{ k, v string }{
		{"0000000000000000000000000000000000000000000000000000000000000001", "a07573657264617461000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000003", "8453bb5b31"},
		{"0000000000000000000000000000000000000000000000000000000000000004", "850218711a00"},
		{"0000000000000000000000000000000000000000000000000000000000000005", "9462d7705bd0b3ecbc51a8026a25597cb28a650c79"},
		{"0000000000000000000000000000000000000000000000000000000000000010", "947e70f9460402290a3e487dae01f610a1a8218fda"},
		{"0000000000000000000000000000000000000000000000000000000000000111", "01"},
		{"0000000000000000000000000000000000000000000000000000000000000112", "a053656e6174650000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000113", "a053656e6174650000000000000000000000000000000000000000000000000000"},
		{"53656e6174650000000000000000000000000000000000000000000000000000", "94977e3f62f5e1ed7953697430303a3cfa2b5b736e"},
	}
	for _, e := range data {
		trie.Update(string(ethutil.Hex2Bytes(e.k)), string(ethutil.Hex2Bytes(e.v)))
	}

	fmt.Printf("root after update %x\n", trie.Root)
	trie.NewIterator().Each(func(k string, v *ethutil.Value) {
		fmt.Printf("%x %x\n", k, v.Bytes())
	})

	data = []struct{ k, v string }{
		{"0000000000000000000000000000000000000000000000000000000000000112", ""},
		{"436974697a656e73000000000000000000000000000000000000000000000001", ""},
		{"436f757274000000000000000000000000000000000000000000000000000002", ""},
		{"53656e6174650000000000000000000000000000000000000000000000000000", ""},
		{"436f757274000000000000000000000000000000000000000000000000000000", ""},
		{"53656e6174650000000000000000000000000000000000000000000000000001", ""},
		{"0000000000000000000000000000000000000000000000000000000000000113", ""},
		{"436974697a656e73000000000000000000000000000000000000000000000000", ""},
		{"436974697a656e73000000000000000000000000000000000000000000000002", ""},
		{"436f757274000000000000000000000000000000000000000000000000000001", ""},
		{"0000000000000000000000000000000000000000000000000000000000000111", ""},
		{"53656e6174650000000000000000000000000000000000000000000000000002", ""},
	}

	for _, e := range data {
		trie.Delete(string(ethutil.Hex2Bytes(e.k)))
	}

	fmt.Printf("root after delete %x\n", trie.Root)

	trie.NewIterator().Each(func(k string, v *ethutil.Value) {
		fmt.Printf("%x %x\n", k, v.Bytes())
	})

	fmt.Printf("%x\n", trie.Get(string(ethutil.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))))
}
*/
