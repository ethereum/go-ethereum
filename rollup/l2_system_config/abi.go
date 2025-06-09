package l2_system_config

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

var (
	l2SystemConfigABI *abi.ABI

	baseFeeOverheadUpdatedEventName = "BaseFeeOverheadUpdated"
	baseFeeScalarUpdatedEventName   = "BaseFeeScalarUpdated"

	BaseFeeOverheadUpdatedTopic common.Hash
	BaseFeeScalarUpdatedTopic   common.Hash
)

func init() {
	l2SystemConfigABI, _ = l2SystemConfigMetaData.GetAbi()

	BaseFeeOverheadUpdatedTopic = l2SystemConfigABI.Events[baseFeeOverheadUpdatedEventName].ID
	BaseFeeScalarUpdatedTopic = l2SystemConfigABI.Events[baseFeeScalarUpdatedEventName].ID
}

var l2SystemConfigMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"event\",\"name\":\"BaseFeeOverheadUpdated\",\"inputs\":[{\"name\":\"oldBaseFeeOverhead\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"newBaseFeeOverhead\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BaseFeeScalarUpdated\",\"inputs\":[{\"name\":\"oldBaseFeeScalar\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"newBaseFeeScalar\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
}

type BaseFeeOverheadUpdatedEventUnpacked struct {
	OldBaseFeeOverhead *big.Int
	NewBaseFeeOverhead *big.Int
}

type BaseFeeScalarUpdatedEventUnpacked struct {
	OldBaseFeeScalar *big.Int
	NewBaseFeeScalar *big.Int
}

func UnpackBaseFeeOverheadUpdatedEvent(log types.Log) (*BaseFeeOverheadUpdatedEventUnpacked, error) {
	event := &BaseFeeOverheadUpdatedEventUnpacked{}
	err := l1.UnpackLog(l2SystemConfigABI, event, baseFeeOverheadUpdatedEventName, log)
	return event, err
}

func UnpackBaseFeeScalarUpdatedEvent(log types.Log) (*BaseFeeScalarUpdatedEventUnpacked, error) {
	event := &BaseFeeScalarUpdatedEventUnpacked{}
	err := l1.UnpackLog(l2SystemConfigABI, event, baseFeeScalarUpdatedEventName, log)
	return event, err
}
