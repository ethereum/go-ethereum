package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	correctTxDependency := [][]uint64{{}, {0}, {}, {1}, {3}, {}, {0, 2}, {5}, {}, {8}}
	wrongTxDependency := [][]uint64{{0}}
	wrongTxDependencyCircular := [][]uint64{{}, {2}, {1}}
	wrongTxDependencyOutOfRange := [][]uint64{{}, {}, {3}}

	var temp map[int][]int

	temp = GetDeps(correctTxDependency)
	assert.Equal(t, true, VerifyDeps(temp))

	temp = GetDeps(wrongTxDependency)
	assert.Equal(t, false, VerifyDeps(temp))

	temp = GetDeps(wrongTxDependencyCircular)
	assert.Equal(t, false, VerifyDeps(temp))

	temp = GetDeps(wrongTxDependencyOutOfRange)
	assert.Equal(t, false, VerifyDeps(temp))
}
