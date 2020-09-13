package peggy

import (
	"github.com/althea-net/peggy/module/x/peggy/keeper"
	"github.com/althea-net/peggy/module/x/peggy/types"
)

const (
	ModuleName        = types.ModuleName
	RouterKey         = types.RouterKey
	StoreKey          = types.StoreKey
	DefaultParamspace = types.ModuleName
	QuerierRoute      = types.QuerierRoute
)

var (
	NewKeeper           = keeper.NewKeeper
	NewQuerier          = keeper.NewQuerier
	NewMsgSetEthAddress = types.NewMsgSetEthAddress
	ModuleCdc           = types.ModuleCdc
	RegisterCodec       = types.RegisterCodec
)

type (
	Keeper           = keeper.Keeper
	MsgSetEthAddress = types.MsgSetEthAddress
	MsgValsetConfirm = types.MsgValsetConfirm
	MsgValsetRequest = types.MsgValsetRequest
	MsgSendToEth     = types.MsgSendToEth
	MsgRequestBatch  = types.MsgRequestBatch
	MsgConfirmBatch  = types.MsgConfirmBatch
	MsgBatchInChain  = types.MsgBatchInChain
	MsgEthDeposit    = types.MsgEthDeposit
)
