package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// ModuleCdc is the codec for the module
var ModuleCdc = codec.New()

func init() {
	RegisterCodec(ModuleCdc)
}

// RegisterCodec registers concrete types on the Amino codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgSetEthAddress{}, "peggy/MsgSetEthAddress", nil)
	cdc.RegisterConcrete(MsgValsetRequest{}, "peggy/MsgValsetRequest", nil)
	cdc.RegisterConcrete(MsgValsetConfirm{}, "peggy/MsgValsetConfirm", nil)

	cdc.RegisterConcrete(Valset{}, "peggy/Valset", nil)

}
