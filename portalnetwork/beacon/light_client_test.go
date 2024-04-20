package beacon

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ ConsensusAPI = (*MockConsensusAPI)(nil)

type MockConsensusAPI struct {
	testdataDir string
}

func NewMockConsensusAPI(path string) (ConsensusAPI, error) {
	return &MockConsensusAPI{testdataDir: path}, nil
}

func (m MockConsensusAPI) GetUpdates(_, _ uint64) ([]common.SpecObj, error) {
	jsonStr, _ := os.ReadFile(m.testdataDir + "/updates.json")

	updates := make([]*capella.LightClientUpdate, 0)
	_ = json.Unmarshal(jsonStr, &updates)

	res := make([]common.SpecObj, 0)

	for _, item := range updates {
		res = append(res, item)
	}
	return res, nil
}

func (m MockConsensusAPI) GetCheckpointData(_ common.Root) (common.SpecObj, error) {
	jsonStr, _ := os.ReadFile(m.testdataDir + "/bootstrap.json")

	bootstrap := &capella.LightClientBootstrap{}
	_ = json.Unmarshal(jsonStr, &bootstrap)

	return bootstrap, nil
}

func (m MockConsensusAPI) GetFinalityData() (common.SpecObj, error) {
	jsonStr, _ := os.ReadFile(m.testdataDir + "/finality.json")

	finality := &capella.LightClientFinalityUpdate{}
	_ = json.Unmarshal(jsonStr, &finality)

	return finality, nil
}

func (m MockConsensusAPI) GetOptimisticData() (common.SpecObj, error) {
	jsonStr, _ := os.ReadFile(m.testdataDir + "/optimistic.json")

	optimistic := &capella.LightClientOptimisticUpdate{}
	_ = json.Unmarshal(jsonStr, &optimistic)

	return optimistic, nil
}

func (m MockConsensusAPI) ChainID() uint64 {
	panic("implement me")
}

func (m MockConsensusAPI) Name() string {
	return "mock"
}

func getClient(strictCheckpointAge bool, t *testing.T) (*ConsensusLightClient, error) {
	baseConfig := Mainnet()
	api, err := NewMockConsensusAPI("testdata/mockdata")
	assert.NoError(t, err)

	config := &Config{
		ConsensusAPI:        api.Name(),
		Chain:               baseConfig.Chain,
		Spec:                baseConfig.Spec,
		StrictCheckpointAge: strictCheckpointAge,
	}

	checkpoint := common.Root(hexutil.MustDecode("0xc62aa0de55e6f21230fa63713715e1a6c13e73005e89f6389da271955d819bde"))

	client, err := NewConsensusLightClient(api, config, checkpoint, testlog.Logger(t, log.LvlTrace))
	if err != nil {
		return nil, err
	}

	return client, nil
}

func TestVerifyCheckpointAgeInvalid(t *testing.T) {
	_, err := getClient(true, t)
	assert.ErrorContains(t, err, "checkpoint is too old")
}

func TestVerifyUpdate(t *testing.T) {
	client, err := getClient(false, t)
	require.NoError(t, err)
	client.Config.MaxCheckpointAge = 123123123

	period := CalcSyncPeriod(uint64(client.Store.FinalizedHeader.Slot))
	updates, err := client.API.GetUpdates(period, MaxRequestLightClientUpdates)
	require.NoError(t, err)
	// normal
	err = client.VerifyUpdate(updates[0])
	require.NoError(t, err)
	// ErrInvalidNextSyncCommitteeProof
	genericUpdate, err := FromLightClientUpdate(updates[0])
	require.NoError(t, err)
	genericUpdate.NextSyncCommittee.Pubkeys[0] = common.BLSPubkey{}
	err = client.VerifyGenericUpdate(genericUpdate)
	require.Equal(t, ErrInvalidNextSyncCommitteeProof, err)
	// ErrInvalidFinalityProof
	updates, err = client.API.GetUpdates(period, MaxRequestLightClientUpdates)
	require.NoError(t, err)
	genericUpdate, err = FromLightClientUpdate(updates[0])
	require.NoError(t, err)
	genericUpdate.FinalizedHeader = &common.BeaconBlockHeader{}
	err = client.VerifyGenericUpdate(genericUpdate)
	require.Equal(t, ErrInvalidFinalityProof, err)

	// ErrInvalidSignature
	updates, err = client.API.GetUpdates(period, MaxRequestLightClientUpdates)
	require.NoError(t, err)
	genericUpdate, err = FromLightClientUpdate(updates[0])
	require.NoError(t, err)
	genericUpdate.SyncAggregate.SyncCommitteeSignature[1] = 0xFE
	err = client.VerifyGenericUpdate(genericUpdate)
	require.Error(t, err)
}

func TestVerifyFinalityUpdate(t *testing.T) {
	client, err := getClient(false, t)
	require.NoError(t, err)

	update, err := client.API.GetFinalityData()
	require.NoError(t, err)

	// normal
	err = client.VerifyFinalityUpdate(update)
	require.NoError(t, err)

	genericUpdate, err := FromLightClientFinalityUpdate(update)
	require.NoError(t, err)

	genericUpdate.FinalizedHeader = &common.BeaconBlockHeader{}
	err = client.VerifyGenericUpdate(genericUpdate)
	require.Equal(t, ErrInvalidFinalityProof, err)
	// ErrInvalidSignature
	update, err = client.API.GetFinalityData()
	require.NoError(t, err)

	genericUpdate, err = FromLightClientFinalityUpdate(update)
	require.NoError(t, err)
	genericUpdate.SyncAggregate.SyncCommitteeSignature[1] = 0xFE
	err = client.VerifyGenericUpdate(genericUpdate)
	require.Error(t, err)
}

func TestVerifyOptimisticUpdate(t *testing.T) {
	client, err := getClient(false, t)
	require.NoError(t, err)

	update, err := client.API.GetOptimisticData()
	require.NoError(t, err)

	// normal
	err = client.VerifyOptimisticUpdate(update)
	require.NoError(t, err)

	genericUpdate, err := FromLightClientOptimisticUpdate(update)
	require.NoError(t, err)

	genericUpdate.SyncAggregate.SyncCommitteeSignature = common.BLSSignature{}
	err = client.VerifyGenericUpdate(genericUpdate)
	require.Error(t, err)
}

func TestSync(t *testing.T) {
	client, err := getClient(false, t)
	require.NoError(t, err)

	err = client.Sync()
	require.NoError(t, err)

	header := client.GetHeader()
	require.Equal(t, header.Slot, common.Slot(7358726))

	finalizedHead := client.GetFinalityHeader()
	require.Equal(t, finalizedHead.Slot, common.Slot(7358656))
}
