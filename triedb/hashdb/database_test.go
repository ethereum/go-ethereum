package hashdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/triedb/rawdb"
)

func TestCleanerDelete(t *testing.T) {
	// Create a new database with clean cache enabled
	db := New(rawdb.NewMemoryDatabase(), &Config{CleanCacheSize: 256 * 1024})
	defer db.Close()
	
	// Create a test node and add it to clean cache
	hash := common.HexToHash("0x1234")
	testData := []byte("test data")
	db.cleans.Set(hash[:], testData)
	
	// Create cleaner and delete the node
	c := &cleaner{db: db}
	if err := c.Delete(hash[:]); err != nil {
		t.Fatalf("failed to delete node: %v", err)
	}
	
	// Verify node was deleted from clean cache
	if data := db.cleans.Get(nil, hash[:]); data != nil {
		t.Error("node was not deleted from clean cache")
	}
	
	// Test delete with nil clean cache
	db.cleans = nil
	if err := c.Delete(hash[:]); err != nil {
		t.Fatalf("failed to handle nil clean cache: %v", err)
	}
} 
