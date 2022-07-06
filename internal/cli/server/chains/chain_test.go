package chains

import (
	"testing"
)

func TestChain_ImportFromFile(t *testing.T) {
	t.Parallel()

	type args struct {
		filename string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "ImportFromFile correct json file",
			args:    args{filename: "test_files/chain_test.json"},
			wantErr: false,
		},
		{
			name:    "ImportFromFile correct legacy json file",
			args:    args{filename: "test_files/chain_legacy_test.json"},
			wantErr: false,
		},
		{
			name:    "ImportFromFile wrong json file",
			args:    args{filename: "test_files/wrong_chain.json"},
			wantErr: true,
		},
		{
			name:    "ImportFromFile nonexistent json file",
			args:    args{filename: "test_files/chain_test_nonexistent.json"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ImportFromFile(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ImportFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
