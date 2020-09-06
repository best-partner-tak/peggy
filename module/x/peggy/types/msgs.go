package types

import (
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/althea-net/peggy/module/x/peggy/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/crypto"
)

// ValsetConfirm
// this is the message sent by the validators when they wish to submit their signatures over
// the validator set at a given block height. A validator must first call MsgSetEthAddress to
// set their Ethereum address to be used for signing. Then someone (anyone) must make a ValsetRequest
// the request is essentially a messaging mechanism to determine which block all validators should submit
// signatures over. Finally validators sign the validator set, powers, and Ethereum addresses of the
// entire validator set at the height of a ValsetRequest and submit that signature with this message
// a ValsetConfirm.
//
// If a sufficient number of validators (66% of voting power) (A) have set Ethereum addresses and (B)
// submit ValsetConfirm messages with their signatures it is then possible for anyone to view these
// signatures in the chain store and submit them to Ethereum to update the validator set
// -------------
type MsgValsetConfirm struct {
	Nonce     int64          `json:"nonce"`
	Validator sdk.AccAddress `json:"validator"`
	Signature []byte         `json:"signature"`
}

func NewMsgValsetConfirm(nonce int64, validator sdk.AccAddress, signature []byte) MsgValsetConfirm {
	return MsgValsetConfirm{
		Nonce:     nonce,
		Validator: validator,
		Signature: signature,
	}
}

// Route should return the name of the module
func (msg MsgValsetConfirm) Route() string { return RouterKey }

// Type should return the action
func (msg MsgValsetConfirm) Type() string { return "valset_confirm" }

// Stateless checks
func (msg MsgValsetConfirm) ValidateBasic() error {
	if msg.Validator.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Validator.String())
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgValsetConfirm) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgValsetConfirm) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Validator}
}

// ValsetRequest
// This message starts off the validator set update process by coordinating a block height
// around which signatures over the validators, powers, and ethereum addresses will be made
// and submitted using a ValsetConfirm. Anyone can send this message as it is not authenticated
// in any way. In theory people could spam it and the validators will have to determine which
// block to actually coordinate around by looking over the valset requests and seeing which one
// some other validator has already submitted a ValsetResponse for.
// -------------
type MsgValsetRequest struct {
	Requester sdk.AccAddress `json:"requester"`
}

func NewMsgValsetRequest(requester sdk.AccAddress) MsgValsetRequest {
	return MsgValsetRequest{
		Requester: requester,
	}
}

// Route should return the name of the module
func (msg MsgValsetRequest) Route() string { return RouterKey }

// Type should return the action
func (msg MsgValsetRequest) Type() string { return "valset_request" }

func (msg MsgValsetRequest) ValidateBasic() error { return nil }

// GetSignBytes encodes the message for signing
func (msg MsgValsetRequest) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgValsetRequest) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Requester}
}

// SetEthAddress
// This is used by the validators to set the Ethereum address that represents them on the
// Ethereum side of the bridge. They must sign their Cosmos address using the Ethereum address
// they have submitted.
// Like ValsetResponse this message can in theory be submitted by anyone, but only the current
// validator sets submissions carry any weight.
// -------------
type MsgSetEthAddress struct {
	// the ethereum address
	Address   string         `json:"address"`
	Validator sdk.AccAddress `json:"validator"`
	Signature string         `json:"signature"`
}

func NewMsgSetEthAddress(address string, validator sdk.AccAddress, signature string) MsgSetEthAddress {
	return MsgSetEthAddress{
		Address:   address,
		Validator: validator,
		Signature: signature,
	}
}

// Route should return the name of the module
func (msg MsgSetEthAddress) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSetEthAddress) Type() string { return "set_eth_address" }

// ValidateBasic runs stateless checks on the message
// Checks if the Eth address is valid, and whether the Eth address has signed the validator address
// (proving control of the Eth address)
func (msg MsgSetEthAddress) ValidateBasic() error {
	if msg.Validator.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Validator.String())
	}
	valdiateEthAddr := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")

	if !valdiateEthAddr.MatchString(msg.Address) {
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "This is not a valid Ethereum address")
	}
	sigBytes, hexErr := hex.DecodeString(msg.Signature)
	if hexErr != nil {
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, fmt.Sprintf("Could not decode hex string %s", msg.Signature))
	}

	err := utils.ValidateEthSig(crypto.Keccak256(msg.Validator.Bytes()), sigBytes, msg.Address)

	if err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, fmt.Sprintf("digest: %x sig: %x address %s error: %s", crypto.Keccak256(msg.Validator.Bytes()), msg.Signature, msg.Address, err.Error()))
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSetEthAddress) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgSetEthAddress) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Validator}
}

// MsgSendToEth
// This is the message that a user calls when they want to bridge an asset
// TODO right now this needs to be locked to a single ERC20
// TODO fixed fee amounts for now, variable fee amounts in the fee field later
// TODO actually remove amounts form the users bank balances
// TODO this message modifies the on chain store by adding itself to a txpool
// it will later be removed when it is included in a batch and successfully submitted
// tokens are removed from the users balance immediately
// -------------
type MsgSendToEth struct {
	// the source address on Cosmos
	Sender sdk.AccAddress `json:"sender"`
	// the destination address on Ethereum
	DestAddress string `json:"dest_address"`
	// the coin to send across the bridge, note the restriction that this is a
	// single coin not a set of coins that is normal in other Cosmos messages
	Send sdk.Coin `json:"send"`
	// the fee paid for the bridge, distinct from the fee paid to the chain to
	// actually send this message in the first place. So a successful send has
	// two layers of fees for the user
	BridgeFee sdk.Coin `json:"bridge_fee"`
}

func NewMsgSendToEth(sender sdk.AccAddress, destAddress string, send sdk.Coin, bridgeFee sdk.Coin) MsgSendToEth {
	return MsgSendToEth{
		Sender:      sender,
		DestAddress: destAddress,
		Send:        send,
		BridgeFee:   bridgeFee,
	}
}

// Route should return the name of the module
func (msg MsgSendToEth) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSendToEth) Type() string { return "send_to_eth" }

// ValidateBasic runs stateless checks on the message
// Checks if the Eth address is valid
func (msg MsgSendToEth) ValidateBasic() error {
	// fee and send must be of the same denom
	if msg.Send.Denom != msg.BridgeFee.Denom {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, fmt.Sprintf("Fee and Send must be the same type!"))
	}
	// TODO validate eth address
	// TODO for demo get single allowed demon from the store
	// TODO validate fee is sufficient, fixed fee to start

	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSendToEth) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgSendToEth) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}

// MsgRequestBatch
// this is a message anyone can send that requests a batch of transactions to send across
// the bridge be created for whatever block height this message is included in. This acts as
// a coordination point, the handler for this message looks at the SendToEth tx's in the store
// and generates a batch, also available in the store tied to this message. The validators then
// grab this batch, sign it, submit the signatures with a MsgConfirmBatch before a relayer can
// finally submit the batch
// -------------
type MsgRequestBatch struct {
	Requester sdk.AccAddress `json:"requester"`
	Denom     string         `json:"denom"`
}

func NewMsgRequestBatch(requester sdk.AccAddress) MsgRequestBatch {
	return MsgRequestBatch{
		Requester: requester,
	}
}

// Route should return the name of the module
func (msg MsgRequestBatch) Route() string { return RouterKey }

// Type should return the action
func (msg MsgRequestBatch) Type() string { return "request_batch" }

func (msg MsgRequestBatch) ValidateBasic() error {
	// TODO ensure that Demon matches hardcoded allowed value
	// TODO later make sure that Demon matches a list of tokens already
	// in the bridge to send
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgRequestBatch) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgRequestBatch) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Requester}
}

// MsgConfirmBatch
// When validators observe a MsgRequestBatch they form a batch by ordering transactions currently
// in the txqueue in order of highest to lowest fee, cutting off when the batch either reaches a
// hardcoded maximum size (to be decided, probably around 100) or when transactions stop being
// profitable (TODO determine this without nondeterminism)
// This message includes the batch as well as an Ethereum signature over this batch by the validator
// -------------
type MsgConfirmBatch struct {
	Nonce     int64          `json:"nonce"`
	Validator sdk.AccAddress `json:"validator"`
	Signature string         `json:"signature"`
}

func NewMsgConfirmBatch(nonce int64, validator sdk.AccAddress, signature string) MsgConfirmBatch {
	return MsgConfirmBatch{
		Nonce:     nonce,
		Validator: validator,
		Signature: signature,
	}
}

// Route should return the name of the module
func (msg MsgConfirmBatch) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConfirmBatch) Type() string { return "confirm_batch" }

func (msg MsgConfirmBatch) ValidateBasic() error {
	// TODO validate signature
	// TODO get batch from storage
	// TODO generate batch in storage on MsgRequestBatch in the first place
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgConfirmBatch) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConfirmBatch) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Validator}
}

// MsgBatchInChain
// this message essentially acts as the oracle between Ethereum and Cosmos, when a validator sees
// that a batch has been submitted on to the Ethereum blockchain they will submit this message
// which acts as their oracle attestation. When more than 66% of the active validator set has
// claimed to have seen the transaction batch enter the ethereum blockchain the transactions are
// removed from the tx queue in the store and finally considered transferred. Transactions in the
// txqueue have a batch number they are included in transactions in lower batches that have never
// been submitted are once again valid for inclusion in blocks.
// -------------
type MsgBatchInChain struct {
	Nonce     int64          `json:"nonce"`
	Validator sdk.AccAddress `json:"validator"`
}

func NewMsgBatchInChain(nonce int64, validator sdk.AccAddress) MsgBatchInChain {
	return MsgBatchInChain{
		Nonce:     nonce,
		Validator: validator,
	}
}

// Route should return the name of the module
func (msg MsgBatchInChain) Route() string { return RouterKey }

// Type should return the action
func (msg MsgBatchInChain) Type() string { return "batch_in_chain" }

func (msg MsgBatchInChain) ValidateBasic() error {
	// TODO ensure that nonce is > the previous BatchInChain
	// TODO think about dealing with changing validator sets during the confirmation process
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgBatchInChain) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgBatchInChain) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Validator}
}

// // MsgSetName defines a SetName message
// type MsgSetName struct {
// 	Name  string         `json:"name"`
// 	Value string         `json:"value"`
// 	Owner sdk.AccAddress `json:"owner"`
// }

// // NewMsgSetName is a constructor function for MsgSetName
// func NewMsgSetName(name string, value string, owner sdk.AccAddress) MsgSetName {
// 	return MsgSetName{
// 		Name:  name,
// 		Value: value,
// 		Owner: owner,
// 	}
// }

// // Route should return the name of the module
// func (msg MsgSetName) Route() string { return RouterKey }

// // Type should return the action
// func (msg MsgSetName) Type() string { return "set_name" }

// // ValidateBasic runs stateless checks on the message
// func (msg MsgSetName) ValidateBasic() error {
// 	if msg.Owner.Empty() {
// 		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Owner.String())
// 	}
// 	if len(msg.Name) == 0 || len(msg.Value) == 0 {
// 		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "Name and/or Value cannot be empty")
// 	}
// 	return nil
// }

// // GetSignBytes encodes the message for signing
// func (msg MsgSetName) GetSignBytes() []byte {
// 	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
// }

// // GetSigners defines whose signature is required
// func (msg MsgSetName) GetSigners() []sdk.AccAddress {
// 	return []sdk.AccAddress{msg.Owner}
// }

// // MsgBuyName defines the BuyName message
// type MsgBuyName struct {
// 	Name  string         `json:"name"`
// 	Bid   sdk.Coins      `json:"bid"`
// 	Buyer sdk.AccAddress `json:"buyer"`
// }

// // NewMsgBuyName is the constructor function for MsgBuyName
// func NewMsgBuyName(name string, bid sdk.Coins, buyer sdk.AccAddress) MsgBuyName {
// 	return MsgBuyName{
// 		Name:  name,
// 		Bid:   bid,
// 		Buyer: buyer,
// 	}
// }

// // Route should return the name of the module
// func (msg MsgBuyName) Route() string { return RouterKey }

// // Type should return the action
// func (msg MsgBuyName) Type() string { return "buy_name" }

// // ValidateBasic runs stateless checks on the message
// func (msg MsgBuyName) ValidateBasic() error {
// 	if msg.Buyer.Empty() {
// 		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Buyer.String())
// 	}
// 	if len(msg.Name) == 0 {
// 		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "Name cannot be empty")
// 	}
// 	if !msg.Bid.IsAllPositive() {
// 		return sdkerrors.ErrInsufficientFunds
// 	}
// 	return nil
// }

// // GetSignBytes encodes the message for signing
// func (msg MsgBuyName) GetSignBytes() []byte {
// 	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
// }

// // GetSigners defines whose signature is required
// func (msg MsgBuyName) GetSigners() []sdk.AccAddress {
// 	return []sdk.AccAddress{msg.Buyer}
// }

// // MsgDeleteName defines a DeleteName message
// type MsgDeleteName struct {
// 	Name  string         `json:"name"`
// 	Owner sdk.AccAddress `json:"owner"`
// }

// // NewMsgDeleteName is a constructor function for MsgDeleteName
// func NewMsgDeleteName(name string, owner sdk.AccAddress) MsgDeleteName {
// 	return MsgDeleteName{
// 		Name:  name,
// 		Owner: owner,
// 	}
// }

// // Route should return the name of the module
// func (msg MsgDeleteName) Route() string { return RouterKey }

// // Type should return the action
// func (msg MsgDeleteName) Type() string { return "delete_name" }

// // ValidateBasic runs stateless checks on the message
// func (msg MsgDeleteName) ValidateBasic() error {
// 	if msg.Owner.Empty() {
// 		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Owner.String())
// 	}
// 	if len(msg.Name) == 0 {
// 		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "Name cannot be empty")
// 	}
// 	return nil
// }

// // GetSignBytes encodes the message for signing
// func (msg MsgDeleteName) GetSignBytes() []byte {
// 	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
// }

// // GetSigners defines whose signature is required
// func (msg MsgDeleteName) GetSigners() []sdk.AccAddress {
// 	return []sdk.AccAddress{msg.Owner}
// }
