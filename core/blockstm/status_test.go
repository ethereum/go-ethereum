package blockstm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusBasics(t *testing.T) {
	t.Parallel()

	s := makeStatusManager(10)

	x := s.takeNextPending()
	require.Equal(t, 0, x)
	require.True(t, s.checkInProgress(x))

	x = s.takeNextPending()
	require.Equal(t, 1, x)
	require.True(t, s.checkInProgress(x))

	x = s.takeNextPending()
	require.Equal(t, 2, x)
	require.True(t, s.checkInProgress(x))

	s.markComplete(0)
	require.False(t, s.checkInProgress(0))
	s.markComplete(1)
	s.markComplete(2)
	require.False(t, s.checkInProgress(1))
	require.False(t, s.checkInProgress(2))
	require.Equal(t, 2, s.maxAllComplete())

	x = s.takeNextPending()
	require.Equal(t, 3, x)

	x = s.takeNextPending()
	require.Equal(t, 4, x)

	s.markComplete(x)
	require.False(t, s.checkInProgress(4))
	// PSP - is this correct? {s.maxAllComplete() -> 2}
	// s -> {[5 6 7 8 9] [3] [0 1 2 4] map[] map[]}
	require.Equal(t, 2, s.maxAllComplete(), "zero should still be min complete")

	exp := []int{1, 2}
	require.Equal(t, exp, s.getRevalidationRange(1))
}

func TestMaxComplete(t *testing.T) {
	t.Parallel()

	s := makeStatusManager(10)

	for {
		tx := s.takeNextPending()

		if tx == -1 {
			break
		}

		if tx != 7 {
			s.markComplete(tx)
		}
	}

	require.Equal(t, 6, s.maxAllComplete())

	s2 := makeStatusManager(10)

	for {
		tx := s2.takeNextPending()

		if tx == -1 {
			break
		}
	}
	s2.markComplete(2)
	s2.markComplete(4)
	require.Equal(t, -1, s2.maxAllComplete())

	s2.complete = insertInList(s2.complete, 4)
	require.Equal(t, 2, s2.countComplete())
}
