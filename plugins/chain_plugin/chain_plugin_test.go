package chain_plugin

import (
	"encoding/json"
	"github.com/eosspark/eos-go/chain/types"
	"github.com/eosspark/eos-go/common"
	"github.com/eosspark/eos-go/crypto"
	"github.com/eosspark/eos-go/crypto/ecc"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApiParams(t *testing.T) {
	//get_currency_balance params
	getCurrencyBalanceParams := GetCurrencyBalanceParams{
		Code:    common.N("eosio.token"),
		Account: common.N("eosio"),
		Symbol:  "SYS",
	}

	var variant common.Variant
	err := common.ToVariant(getCurrencyBalanceParams, &variant)
	assert.NoError(t, err)
	body, err := json.Marshal(variant)
	assert.NoError(t, err)

	var params GetCurrencyBalanceParams
	err = json.Unmarshal(body, &params)
	assert.NoError(t, err)
	assert.Equal(t, common.N("eosio.token"), params.Code)
	assert.Equal(t, common.N("eosio"), params.Account)
	assert.Equal(t, "SYS", params.Symbol)

	//push_transaction params
	action := []byte("\"code\":\"eosio\", \"action\":\"newaccount\", \"data\":\"eosio.token\"")
	prikey, _ := ecc.NewRandomPrivateKey()
	sign, err := prikey.Sign(crypto.Hash256(action).Bytes())
	assert.NoError(t, err)

	packed := types.PackedTransaction{
		Signatures:            []ecc.Signature{sign},
		Compression:           types.CompressionNone,
		PackedContextFreeData: action,
		PackedTrx:             action,
		UnpackedTrx:           nil,
	}

	err = common.ToVariant(packed, &variant)
	assert.NoError(t, err)

	body, err = json.Marshal(variant)
	assert.NoError(t, err)

	var pushTrxParams PushTransactionParams
	err = json.Unmarshal(body, &pushTrxParams)
	assert.NoError(t, err)

	var prettyInput types.PackedTransaction
	err = common.FromVariant(pushTrxParams, &prettyInput)
	assert.NoError(t, err)
	assert.Equal(t, sign, prettyInput.Signatures[0])
	assert.Equal(t, types.CompressionNone, prettyInput.Compression)
	assert.Equal(t, common.HexBytes(action), prettyInput.PackedTrx)
	assert.Equal(t, common.HexBytes(action), prettyInput.PackedContextFreeData)
	assert.Equal(t, (*types.Transaction)(nil), prettyInput.UnpackedTrx)
}
