package set

import (
	"reflect"
	"strings"
	"testing"
)

func TestSetNonTS_NewNonTS_parameters(t *testing.T) {
	s := NewNonTS("string", "another_string", 1, 3.14)

	if s.Size() != 4 {
		t.Error("NewNonTS: calling with parameters should create a set with size of four")
	}
}

func TestSetNonTS_Add(t *testing.T) {
	s := NewNonTS()
	s.Add(1)
	s.Add(2)
	s.Add(2) // duplicate
	s.Add("fatih")
	s.Add("zeynep")
	s.Add("zeynep") // another duplicate

	if s.Size() != 4 {
		t.Error("Add: items are not unique. The set size should be four")
	}

	if !s.Has(1, 2, "fatih", "zeynep") {
		t.Error("Add: added items are not availabile in the set.")
	}
}

func TestSetNonTS_Add_multiple(t *testing.T) {
	s := NewNonTS()
	s.Add("ankara", "san francisco", 3.14)

	if s.Size() != 3 {
		t.Error("Add: items are not unique. The set size should be three")
	}

	if !s.Has("ankara", "san francisco", 3.14) {
		t.Error("Add: added items are not availabile in the set.")
	}
}

func TestSetNonTS_Remove(t *testing.T) {
	s := NewNonTS()
	s.Add(1)
	s.Add(2)
	s.Add("fatih")

	s.Remove(1)
	if s.Size() != 2 {
		t.Error("Remove: set size should be two after removing")
	}

	s.Remove(1)
	if s.Size() != 2 {
		t.Error("Remove: set size should be not change after trying to remove a non-existing item")
	}

	s.Remove(2)
	s.Remove("fatih")
	if s.Size() != 0 {
		t.Error("Remove: set size should be zero")
	}

	s.Remove("fatih") // try to remove something from a zero length set
}

func TestSetNonTS_Remove_multiple(t *testing.T) {
	s := NewNonTS()
	s.Add("ankara", "san francisco", 3.14, "istanbul")
	s.Remove("ankara", "san francisco", 3.14)

	if s.Size() != 1 {
		t.Error("Remove: items are not unique. The set size should be four")
	}

	if !s.Has("istanbul") {
		t.Error("Add: added items are not availabile in the set.")
	}
}

func TestSetNonTS_Pop(t *testing.T) {
	s := NewNonTS()
	s.Add(1)
	s.Add(2)
	s.Add("fatih")

	a := s.Pop()
	if s.Size() != 2 {
		t.Error("Pop: set size should be two after popping out")
	}

	if s.Has(a) {
		t.Error("Pop: returned item should not exist")
	}

	s.Pop()
	s.Pop()
	b := s.Pop()
	if b != nil {
		t.Error("Pop: should return nil because set is empty")
	}

	s.Pop() // try to remove something from a zero length set
}

func TestSetNonTS_Has(t *testing.T) {
	s := NewNonTS("1", "2", "3", "4")

	if !s.Has("1") {
		t.Error("Has: the item 1 exist, but 'Has' is returning false")
	}

	if !s.Has("1", "2", "3", "4") {
		t.Error("Has: the items all exist, but 'Has' is returning false")
	}
}

func TestSetNonTS_Clear(t *testing.T) {
	s := NewNonTS()
	s.Add(1)
	s.Add("istanbul")
	s.Add("san francisco")

	s.Clear()
	if s.Size() != 0 {
		t.Error("Clear: set size should be zero")
	}
}

func TestSetNonTS_IsEmpty(t *testing.T) {
	s := NewNonTS()

	empty := s.IsEmpty()
	if !empty {
		t.Error("IsEmpty: set is empty, it should be true")
	}

	s.Add(2)
	s.Add(3)
	notEmpty := s.IsEmpty()

	if notEmpty {
		t.Error("IsEmpty: set is filled, it should be false")
	}
}

func TestSetNonTS_IsEqual(t *testing.T) {
	s := NewNonTS("1", "2", "3")
	u := NewNonTS("1", "2", "3")

	ok := s.IsEqual(u)
	if !ok {
		t.Error("IsEqual: set s and t are equal. However it returns false")
	}

	// same size, different content
	a := NewNonTS("1", "2", "3")
	b := NewNonTS("4", "5", "6")

	ok = a.IsEqual(b)
	if ok {
		t.Error("IsEqual: set a and b are now equal (1). However it returns true")
	}

	// different size, similar content
	a = NewNonTS("1", "2", "3")
	b = NewNonTS("1", "2", "3", "4")

	ok = a.IsEqual(b)
	if ok {
		t.Error("IsEqual: set s and t are now equal (2). However it returns true")
	}

}

func TestSetNonTS_IsSubset(t *testing.T) {
	s := NewNonTS("1", "2", "3", "4")
	u := NewNonTS("1", "2", "3")

	ok := s.IsSubset(u)
	if !ok {
		t.Error("IsSubset: u is a subset of s. However it returns false")
	}

	ok = u.IsSubset(s)
	if ok {
		t.Error("IsSubset: s is not a subset of u. However it returns true")
	}

}

func TestSetNonTS_IsSuperset(t *testing.T) {
	s := NewNonTS("1", "2", "3", "4")
	u := NewNonTS("1", "2", "3")

	ok := u.IsSuperset(s)
	if !ok {
		t.Error("IsSuperset: s is a superset of u. However it returns false")
	}

	ok = s.IsSuperset(u)
	if ok {
		t.Error("IsSuperset: u is not a superset of u. However it returns true")
	}

}

func TestSetNonTS_String(t *testing.T) {
	s := NewNonTS()
	if s.String() != "[]" {
		t.Errorf("String: output is not what is excepted '%s'", s.String())
	}

	s.Add("1", "2", "3", "4")
	if !strings.HasPrefix(s.String(), "[") {
		t.Error("String: output should begin with a square bracket")
	}

	if !strings.HasSuffix(s.String(), "]") {
		t.Error("String: output should end with a square bracket")
	}

}

func TestSetNonTS_List(t *testing.T) {
	s := NewNonTS("1", "2", "3", "4")

	// this returns a slice of interface{}
	if len(s.List()) != 4 {
		t.Error("List: slice size should be four.")
	}

	for _, item := range s.List() {
		r := reflect.TypeOf(item)
		if r.Kind().String() != "string" {
			t.Error("List: slice item should be a string")
		}
	}
}

func TestSetNonTS_Copy(t *testing.T) {
	s := NewNonTS("1", "2", "3", "4")
	r := s.Copy()

	if !s.IsEqual(r) {
		t.Error("Copy: set s and r are not equal")
	}
}

func TestSetNonTS_Merge(t *testing.T) {
	s := NewNonTS("1", "2", "3")
	r := NewNonTS("3", "4", "5")
	s.Merge(r)

	if s.Size() != 5 {
		t.Error("Merge: the set doesn't have all items in it.")
	}

	if !s.Has("1", "2", "3", "4", "5") {
		t.Error("Merge: merged items are not availabile in the set.")
	}
}

func TestSetNonTS_Separate(t *testing.T) {
	s := NewNonTS("1", "2", "3")
	r := NewNonTS("3", "5")
	s.Separate(r)

	if s.Size() != 2 {
		t.Error("Separate: the set doesn't have all items in it.")
	}

	if !s.Has("1", "2") {
		t.Error("Separate: items after separation are not availabile in the set.")
	}
}
