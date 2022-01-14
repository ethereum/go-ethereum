package utils

import (
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
)

func TestCompareSignersLists(t *testing.T) {
	list1 := []common.Address{
		common.StringToAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		common.StringToAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc"),
		common.StringToAddress("dddddddddddddddddddddddddddddddddddddddd"),
	}
	list2 := []common.Address{
		common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc"),
		common.StringToAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		common.StringToAddress("dddddddddddddddddddddddddddddddddddddddd"),
		common.StringToAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	}
	list3 := []common.Address{
		common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc"),
		common.StringToAddress("dddddddddddddddddddddddddddddddddddddddd"),
		common.StringToAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	}
	if !CompareSignersLists(list1, list2) {
		t.Error("list1 should be equal to list2", "list1", list1, "list2", list2)
	}
	if CompareSignersLists(list1, list3) {
		t.Error("list1 and list3 should not be same", "list1", list1, "list3", list3)
	}
	if !CompareSignersLists([]common.Address{}, []common.Address{}) {
		t.Error("Failed with empty list")
	}
	if !CompareSignersLists([]common.Address{common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc")}, []common.Address{common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc")}) {
		t.Error("Failed with list has only one signer")
	}
	if CompareSignersLists([]common.Address{common.StringToAddress("aaaaaaaaaaaaaaaa")}, []common.Address{common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc")}) {
		t.Error("Failed with list has only one signer")
	}
}
