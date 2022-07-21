package builder

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	boostTypes "github.com/flashbots/go-boost-utils/types"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestRemoteRelay(t *testing.T) {
	r := mux.NewRouter()
	var validatorsHandler func(w http.ResponseWriter, r *http.Request)
	r.HandleFunc("/relay/v1/builder/validators", func(w http.ResponseWriter, r *http.Request) { validatorsHandler(w, r) })

	validatorsHandler = func(w http.ResponseWriter, r *http.Request) {
		resp := `[{
  "slot": "123",
  "entry": {
    "message": {
      "fee_recipient": "0xabcf8e0d4e9587369b2301d0790347320302cc09",
      "gas_limit": "1",
      "timestamp": "1",
      "pubkey": "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }}, {
  "slot": "155",
  "entry": {
    "message": {
      "fee_recipient": "0xabcf8e0d4e9587369b2301d0790347320302cc10",
      "gas_limit": "1",
      "timestamp": "1",
      "pubkey": "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
}]`

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}

	srv := httptest.NewServer(r)
	relay, err := NewRemoteRelay(srv.URL, nil)
	require.NoError(t, err)
	vd, found := relay.validatorSlotMap[123]
	require.True(t, found)
	expectedValidator_123 := ValidatorData{
		Pubkey:       "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a",
		FeeRecipient: boostTypes.Address{0xab, 0xcf, 0x8e, 0xd, 0x4e, 0x95, 0x87, 0x36, 0x9b, 0x23, 0x1, 0xd0, 0x79, 0x3, 0x47, 0x32, 0x3, 0x2, 0xcc, 0x9},
		GasLimit:     uint64(1),
		Timestamp:    uint64(1),
	}
	require.Equal(t, expectedValidator_123, vd)

	vd, err = relay.GetValidatorForSlot(123)
	require.NoError(t, err)
	require.Equal(t, expectedValidator_123, vd)

	vd, err = relay.GetValidatorForSlot(124)
	require.Error(t, err)
	require.Equal(t, vd, ValidatorData{})

	validatorsRequested := make(chan struct{})
	validatorsHandler = func(w http.ResponseWriter, r *http.Request) {
		resp := `[{
  "slot": "155",
  "entry": {
    "message": {
      "fee_recipient": "0xabcf8e0d4e9587369b2301d0790347320302cc10",
      "gas_limit": "1",
      "timestamp": "1",
      "pubkey": "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }}, {
  "slot": "156",
  "entry": {
    "message": {
      "fee_recipient": "0xabcf8e0d4e9587369b2301d0790347320302cc11",
      "gas_limit": "1",
      "timestamp": "1",
      "pubkey": "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
}]`

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
		validatorsRequested <- struct{}{}
	}

	expectedValidator_155 := ValidatorData{
		Pubkey:       "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a",
		FeeRecipient: boostTypes.Address{0xab, 0xcf, 0x8e, 0xd, 0x4e, 0x95, 0x87, 0x36, 0x9b, 0x23, 0x1, 0xd0, 0x79, 0x3, 0x47, 0x32, 0x3, 0x2, 0xcc, 0x10},
		GasLimit:     uint64(1),
		Timestamp:    uint64(1),
	}

	expectedValidator_156 := ValidatorData{
		Pubkey:       "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a",
		FeeRecipient: boostTypes.Address{0xab, 0xcf, 0x8e, 0xd, 0x4e, 0x95, 0x87, 0x36, 0x9b, 0x23, 0x1, 0xd0, 0x79, 0x3, 0x47, 0x32, 0x3, 0x2, 0xcc, 0x11},
		GasLimit:     uint64(1),
		Timestamp:    uint64(1),
	}

	vd, err = relay.GetValidatorForSlot(155)
	require.NoError(t, err)
	require.Equal(t, expectedValidator_155, vd)

	select {
	case <-validatorsRequested:
		for i := 0; i < 10 && relay.lastRequestedSlot != 155; i++ {
			time.Sleep(time.Millisecond)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for validator registration request")
	}

	vd, err = relay.GetValidatorForSlot(156)
	require.NoError(t, err)
	require.Equal(t, expectedValidator_156, vd)
}
