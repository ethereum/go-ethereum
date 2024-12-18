package backoff

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExponentialBackoff(t *testing.T) {
	t.Run("Multiple attempts", func(t *testing.T) {
		e := NewExponential(100*time.Millisecond, 10*time.Second, 0)
		expectedDurations := []time.Duration{
			100 * time.Millisecond,
			200 * time.Millisecond,
			400 * time.Millisecond,
			800 * time.Millisecond,
			1600 * time.Millisecond,
			3200 * time.Millisecond,
			6400 * time.Millisecond,
			10 * time.Second, // capped at max
		}
		for i, expected := range expectedDurations {
			require.Equal(t, expected, e.NextDuration(), "attempt %d", i)
		}
	})

	t.Run("Jitter added", func(t *testing.T) {
		e := NewExponential(1*time.Second, 10*time.Second, 1*time.Second)
		duration := e.NextDuration()
		require.GreaterOrEqual(t, duration, 1*time.Second)
		require.Less(t, duration, 2*time.Second)
	})

	t.Run("Edge case: min > max", func(t *testing.T) {
		e := NewExponential(10*time.Second, 5*time.Second, 0)
		require.Equal(t, 5*time.Second, e.NextDuration())
	})
}
