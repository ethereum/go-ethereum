package ethutil

import (
	"fmt"
	"testing"
)

func TestSize(t *testing.T) {
	fmt.Println(StorageSize(2381273))
	fmt.Println(StorageSize(2192))
	fmt.Println(StorageSize(12))
}
