package l1

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryInBatches(t *testing.T) {
	tests := []struct {
		name          string
		fromBlock     uint64
		toBlock       uint64
		batchSize     uint64
		queryFunc     func(from, to uint64) (bool, error)
		expectErr     bool
		expectedErr   string
		expectedCalls []struct {
			from uint64
			to   uint64
		}
	}{
		{
			name:      "Successful query in single batch",
			fromBlock: 1,
			toBlock:   10,
			batchSize: 10,
			queryFunc: func(from, to uint64) (bool, error) {
				return true, nil
			},
			expectErr: false,
			expectedCalls: []struct {
				from uint64
				to   uint64
			}{
				{from: 1, to: 10},
			},
		},
		{
			name:      "Successful query in multiple batches",
			fromBlock: 1,
			toBlock:   80,
			batchSize: 10,
			queryFunc: func(from, to uint64) (bool, error) {
				return true, nil
			},
			expectErr: false,
			expectedCalls: []struct {
				from uint64
				to   uint64
			}{
				{from: 1, to: 10},
				{from: 11, to: 20},
				{from: 21, to: 30},
				{from: 31, to: 40},
				{from: 41, to: 50},
				{from: 51, to: 60},
				{from: 61, to: 70},
				{from: 71, to: 80},
			},
		},
		{
			name:      "Query function returns error",
			fromBlock: 1,
			toBlock:   10,
			batchSize: 10,
			queryFunc: func(from, to uint64) (bool, error) {
				return false, errors.New("query error")
			},
			expectErr:   true,
			expectedErr: "error querying blocks 1 to 10: query error",
			expectedCalls: []struct {
				from uint64
				to   uint64
			}{
				{from: 1, to: 10},
			},
		},
		{
			name:      "Query function returns false to stop",
			fromBlock: 1,
			toBlock:   20,
			batchSize: 10,
			queryFunc: func(from, to uint64) (bool, error) {
				if from == 1 {
					return false, nil
				}
				return true, nil
			},
			expectErr: false,
			expectedCalls: []struct {
				from uint64
				to   uint64
			}{
				{from: 1, to: 10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []struct {
				from uint64
				to   uint64
			}
			queryFunc := func(from, to uint64) (bool, error) {
				calls = append(calls, struct {
					from uint64
					to   uint64
				}{from, to})
				return tt.queryFunc(from, to)
			}
			err := queryInBatches(context.Background(), tt.fromBlock, tt.toBlock, tt.batchSize, queryFunc)
			if tt.expectErr {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedCalls, calls)
		})
	}
}
