// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wasmgo_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/eosspark/eos-go/chain"
	"github.com/eosspark/eos-go/chain/types"
	"github.com/eosspark/eos-go/common"
	"github.com/eosspark/eos-go/crypto"
	"github.com/eosspark/eos-go/crypto/ecc"
	"github.com/eosspark/eos-go/exception"
	"github.com/eosspark/eos-go/exception/try"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/eosspark/eos-go/crypto/rlp"
	"github.com/eosspark/eos-go/wasmgo"
	"github.com/stretchr/testify/assert"
)

const crypto_api_exception int = 0

func TestContextApis(t *testing.T) {
	fnames, err := filepath.Glob(filepath.Join("testdata_context", "*.wasm"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fname := range fnames {
		name := fname
		t.Run(filepath.Base(name), func(t *testing.T) {
			code, err := ioutil.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}

			_, fileName := filepath.Split(name)
			if strings.Compare(fileName, "hello.wasm") == 0 {
				fmt.Println(fileName)
				wasm := wasmgo.NewWasmGo()
				applyContext := &chain.ApplyContext{
					Receiver: common.AccountName(common.N("hello")),
					Act: &types.Action{
						Account: common.AccountName(common.N("hello")),
						Name:    common.ActionName(common.N("hi")),
						Data:    []byte{0x00, 0x00, 0x00, 0x00, 0x5c, 0x05, 0xa3, 0xe1}, //'{"walker"}'
					},
				}

				codeVersion := crypto.NewSha256Byte([]byte(code))
				wasm.Apply(codeVersion, code, applyContext)

				//print "hello,walker"
				//fmt.Println(applyContext.PendingConsoleOutput)
				if strings.Compare(applyContext.PendingConsoleOutput, "Hello, walker") != 0 {
					t.Fatalf("error excute hello.wasm")
				}
			}

		})
	}
}

const DUMMY_ACTION_DEFAULT_A = 0x45
const DUMMY_ACTION_DEFAULT_B = 0xab11cd1244556677
const DUMMY_ACTION_DEFAULT_C = 0x7451ae12

type dummy_action struct {
	A byte
	B uint64
	C int32
}

func (d *dummy_action) get_name() uint64 {
	return common.N("dummy_action")
}

func (d *dummy_action) get_account() uint64 {
	return common.N("testapi")
}

func TestContextAction(t *testing.T) {

	name := "testdata_context/test_api.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}

		dummy13 := dummy_action{DUMMY_ACTION_DEFAULT_A, DUMMY_ACTION_DEFAULT_B, DUMMY_ACTION_DEFAULT_C}

		callTestFunction(code, "test_action", "assert_true", []byte{})
		callTestFunction(code, "test_action", "assert_false", []byte{})

		b, _ := rlp.EncodeToBytes(&dummy13)
		callTestFunction(code, "test_action", "read_action_normal", b)

		//rawBytes := []byte{(1 << 16)}
		b = bytes.Repeat([]byte{byte(0x01)}, 1<<16)
		callTestFunction(code, "test_action", "read_action_to_0", b)
		b = bytes.Repeat([]byte{byte(0x01)}, 1<<16+1)
		callTestFunction(code, "test_action", "read_action_to_0", b)

		b = bytes.Repeat([]byte{byte(0x01)}, 1)
		callTestFunction(code, "test_action", "read_action_to_64k", b)
		b = bytes.Repeat([]byte{byte(0x01)}, 3)
		callTestFunction(code, "test_action", "read_action_to_64k", b)

		callTestFunction(code, "test_action", "require_auth", []byte{})

		a3only := []types.PermissionLevel{{common.AccountName(common.N("acc3")), common.PermissionName(common.N("active"))}}
		b, _ = rlp.EncodeToBytes(a3only)
		callTestFunction(code, "test_action", "require_auth", b)

		a4only := []types.PermissionLevel{{common.AccountName(common.N("acc4")), common.PermissionName(common.N("active"))}}
		b, _ = rlp.EncodeToBytes(a4only)
		callTestFunction(code, "test_action", "require_auth", b)

	})

}

func TestContextPrint(t *testing.T) {

	name := "testdata_context/test_api.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}

		trace := callTestFunction(code, "test_print", "test_prints", []byte{})
		result := trace.PendingConsoleOutput
		assert.Equal(t, result, "abcefg")

		trace = callTestFunction(code, "test_print", "test_prints_l", []byte{})
		result = trace.PendingConsoleOutput
		assert.Equal(t, result, "abatest")

		trace = callTestFunction(code, "test_print", "test_printi", []byte{})
		result = trace.PendingConsoleOutput
		assert.Equal(t, result[0:1], string(strconv.FormatInt(0, 10)))
		assert.Equal(t, result[1:7], string(strconv.FormatInt(556644, 10)))
		assert.Equal(t, result[7:9], string(strconv.FormatInt(-1, 10)))

		trace = callTestFunction(code, "test_print", "test_printui", []byte{})
		result = trace.PendingConsoleOutput
		assert.Equal(t, result[0:1], string(strconv.FormatInt(0, 10)))
		assert.Equal(t, result[1:7], string(strconv.FormatInt(556644, 10)))

		v := -1
		assert.Equal(t, result[7:len(result)], string(strconv.FormatUint(uint64(v), 10))) //-1 / 1844674407370955161

		trace = callTestFunction(code, "test_print", "test_printn", []byte{})
		result = trace.PendingConsoleOutput
		assert.Equal(t, result[0:5], "abcde")
		assert.Equal(t, result[5:10], "ab.de")
		assert.Equal(t, result[10:16], "1q1q1q")
		assert.Equal(t, result[16:27], "abcdefghijk")
		assert.Equal(t, result[27:39], "abcdefghijkl")
		assert.Equal(t, result[39:52], "abcdefghijkl1")
		assert.Equal(t, result[52:65], "abcdefghijkl1")
		assert.Equal(t, result[65:78], "abcdefghijkl1")

		trace = callTestFunction(code, "test_print", "test_printi128", []byte{})
		result = trace.PendingConsoleOutput

		s := strings.Split(result, "\n")
		assert.Equal(t, s[0], "1")
		assert.Equal(t, s[1], "0")
		assert.Equal(t, s[2], "-170141183460469231731687303715884105728")
		assert.Equal(t, s[3], "-87654323456")

		trace = callTestFunction(code, "test_print", "test_printui128", []byte{})
		result = trace.PendingConsoleOutput
		s = strings.Split(result, "\n")
		assert.Equal(t, s[0], "340282366920938463463374607431768211455")
		assert.Equal(t, s[1], "0")
		assert.Equal(t, s[2], "87654323456")

		trace = callTestFunction(code, "test_print", "test_printsf", []byte{})
		result = trace.PendingConsoleOutput
		r := strings.Split(result, "\n")
		assert.Equal(t, r[0], "5.000000e-01")
		assert.Equal(t, r[1], "-3.750000e+00")
		assert.Equal(t, r[2], "6.666667e-07")

		trace = callTestFunction(code, "test_print", "test_printdf", []byte{})
		result = trace.PendingConsoleOutput
		r = strings.Split(result, "\n")
		assert.Equal(t, r[0], "5.000000000000000e-01")
		assert.Equal(t, r[1], "-3.750000000000000e+00")
		assert.Equal(t, r[2], "6.666666666666666e-07")

	})

}

func TestContextTypes(t *testing.T) {

	name := "testdata_context/test_api.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}

		callTestFunction(code, "test_types", "types_size", []byte{})
		callTestFunction(code, "test_types", "char_to_symbol", []byte{})
		callTestFunction(code, "test_types", "string_to_name", []byte{})
		callTestFunction(code, "test_types", "name_class", []byte{})

	})

}

func TestContextMemory(t *testing.T) {

	name := "testdata_context/test_api_mem.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		callTestFunction(code, "test_memory", "test_memory_allocs", []byte{})
		callTestFunction(code, "test_memory", "test_memory_hunk", []byte{})
		callTestFunction(code, "test_memory", "test_memory_hunks", []byte{})
		//callTestFunction(code, "test_memory", "test_memory_hunks_disjoint", []byte{})
		callTestFunction(code, "test_memory", "test_memset_memcpy", []byte{})

		callTestFunctionCheckException(code, "test_memory", "test_memcpy_overlap_start", []byte{}, exception.OverlappingMemoryError{}.Code(), exception.OverlappingMemoryError{}.What())
		callTestFunctionCheckException(code, "test_memory", "test_memcpy_overlap_end", []byte{}, exception.OverlappingMemoryError{}.Code(), exception.OverlappingMemoryError{}.What())

		callTestFunction(code, "test_memory", "test_memcmp", []byte{})

		//callTestFunction(code, "test_memory", "test_outofbound_0", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_1", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_2", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_3", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_4", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_5", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_6", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_7", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_8", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_9", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_10", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_11", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_12", []byte{})
		// callTestFunction(code, "test_memory", "test_outofbound_13", []byte{})

		callTestFunction(code, "test_extended_memory", "test_initial_buffer", []byte{})
		callTestFunction(code, "test_extended_memory", "test_page_memory", []byte{})
		callTestFunction(code, "test_extended_memory", "test_page_memory_exceeded", []byte{})
		callTestFunction(code, "test_extended_memory", "test_page_memory_negative_bytes", []byte{})
	})

}

func TestContextAuth(t *testing.T) {

	name := "testdata_context/auth.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(name)
		wasm := wasmgo.NewWasmGo()
		param, _ := rlp.EncodeToBytes(common.N("walker"))
		applyContext := &chain.ApplyContext{
			Receiver: common.AccountName(common.N("ctx.auth")),
			Act: &types.Action{
				Account: common.AccountName(common.N("ctx.auth")),
				Name:    common.ActionName(common.N("test")),
				Data:    param,
				Authorization: []types.PermissionLevel{{
					Actor:      common.AccountName(common.N("walker")),
					Permission: common.PermissionName(common.N("active")),
				}},
			},
			UsedAuthorizations: make([]bool, 1),
		}

		codeVersion := crypto.NewSha256Byte([]byte(code))
		wasm.Apply(codeVersion, code, applyContext)

		result := fmt.Sprintf("%v", applyContext.PendingConsoleOutput)
		assert.Equal(t, result, "walker has authorization,walker is account")

	})

}

func TestContextCrypto(t *testing.T) {

	name := "testdata_context/test_api.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(name)

		wif := "5KQwrPbwdL6PhXujxW37FSSQZ1JiwsST4cqQzDeyXtP79zkvFD3"
		privKey, err := ecc.NewPrivateKey(wif)

		chainID, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
		payload, err := hex.DecodeString("88e4b25a00006c08ac5b595b000000000000")
		digest := sigDigest(chainID, payload)
		sig, err := privKey.Sign(digest)
		pubKey, err := sig.PublicKey(digest)

		load := digest

		p, _ := rlp.EncodeToBytes(pubKey)
		load = append(load, p...)

		s, _ := rlp.EncodeToBytes(sig)
		load = append(load, s...)

		fmt.Println("load:", hex.EncodeToString(load))

		callTestFunction(code, "test_crypto", "test_recover_key", load)
		callTestFunction(code, "test_crypto", "test_recover_key_assert_true", load)
		callTestFunction(code, "test_crypto", "test_sha1", []byte{})
		callTestFunction(code, "test_crypto", "test_sha256", []byte{})
		callTestFunction(code, "test_crypto", "test_sha512", []byte{})
		callTestFunction(code, "test_crypto", "test_ripemd160", []byte{})
		callTestFunction(code, "test_crypto", "sha1_no_data", []byte{})
		callTestFunction(code, "test_crypto", "sha256_no_data", []byte{})
		callTestFunction(code, "test_crypto", "sha512_no_data", []byte{})
		callTestFunction(code, "test_crypto", "ripemd160_no_data", []byte{})
		callTestFunction(code, "test_crypto", "assert_sha256_true", []byte{})
		callTestFunction(code, "test_crypto", "assert_sha1_true", []byte{})
		callTestFunction(code, "test_crypto", "assert_sha512_true", []byte{})
		callTestFunction(code, "test_crypto", "assert_ripemd160_true", []byte{})

		callTestFunctionCheckException(code, "test_crypto", "assert_sha256_false", []byte{}, exception.CryptoApiException{}.Code(), exception.CryptoApiException{}.What())
		callTestFunctionCheckException(code, "test_crypto", "assert_sha1_false", []byte{}, exception.CryptoApiException{}.Code(), exception.CryptoApiException{}.What())
		callTestFunctionCheckException(code, "test_crypto", "assert_sha512_false", []byte{}, exception.CryptoApiException{}.Code(), exception.CryptoApiException{}.What())
		callTestFunctionCheckException(code, "test_crypto", "assert_ripemd160_false", []byte{}, exception.CryptoApiException{}.Code(), exception.CryptoApiException{}.What())

	})
}

func TestContextFixedPoint(t *testing.T) {

	name := "testdata_context/test_api.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		callTestFunction(code, "test_fixedpoint", "create_instances", []byte{})
		callTestFunction(code, "test_fixedpoint", "test_addition", []byte{})
		callTestFunction(code, "test_fixedpoint", "test_subtraction", []byte{})
		callTestFunction(code, "test_fixedpoint", "test_multiplication", []byte{})
		callTestFunction(code, "test_fixedpoint", "test_division", []byte{})
		callTestFunction(code, "test_fixedpoint", "test_division_by_0", []byte{})

	})
}

func TestContextChecktime(t *testing.T) {

	name := "testdata_context/test_api.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		callTestFunction(code, "test_checktime", "checktime_pass", []byte{})
		//callTestFunction(code, "test_checktime", "checktime_failure", []byte{})
		callTestFunction(code, "test_checktime", "checktime_sha1_failure", []byte{})
		callTestFunction(code, "test_checktime", "checktime_assert_sha1_failure", []byte{})
		callTestFunction(code, "test_checktime", "checktime_sha256_failure", []byte{})
		callTestFunction(code, "test_checktime", "checktime_assert_sha256_failure", []byte{})
		callTestFunction(code, "test_checktime", "checktime_sha512_failure", []byte{})
		callTestFunction(code, "test_checktime", "checktime_assert_sha512_failure", []byte{})
		callTestFunction(code, "test_checktime", "checktime_ripemd160_failure", []byte{})
		callTestFunction(code, "test_checktime", "checktime_assert_ripemd160_failure", []byte{})

	})
}

func TestContextDatastream(t *testing.T) {

	name := "testdata_context/test_api.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		callTestFunction(code, "test_datastream", "test_basic", []byte{})

	})
}

func TestContextCompilerBuiltin(t *testing.T) {

	name := "testdata_context/compiler_builtin.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}

		callTestFunction(code, "test_compiler_builtins", "test_ashrti3", []byte{})
		callTestFunction(code, "test_compiler_builtins", "test_ashlti3", []byte{})
		callTestFunction(code, "test_compiler_builtins", "test_lshrti3", []byte{})
		callTestFunction(code, "test_compiler_builtins", "test_lshlti3", []byte{})

		callTestFunction(code, "test_compiler_builtins", "test_umodti3", []byte{})
		callTestFunctionCheckException(code, "test_compiler_builtins", "test_umodti3_by_0", []byte{},
			exception.ArithmeticException{}.Code(), exception.ArithmeticException{}.What())

		callTestFunction(code, "test_compiler_builtins", "test_modti3", []byte{})
		callTestFunctionCheckException(code, "test_compiler_builtins", "test_modti3_by_0", []byte{},
			exception.ArithmeticException{}.Code(), exception.ArithmeticException{}.What())

		callTestFunction(code, "test_compiler_builtins", "test_udivti3", []byte{})
		callTestFunctionCheckException(code, "test_compiler_builtins", "test_udivti3_by_0", []byte{},
			exception.ArithmeticException{}.Code(), exception.ArithmeticException{}.What())

		callTestFunction(code, "test_compiler_builtins", "test_divti3", []byte{})
		callTestFunctionCheckException(code, "test_compiler_builtins", "test_divti3_by_0", []byte{},
			exception.ArithmeticException{}.Code(), exception.ArithmeticException{}.What())

		callTestFunction(code, "test_compiler_builtins", "test_multi3", []byte{})
	})
}

type invalidAccessAction struct {
	Code  uint64
	Val   uint64
	Index uint32
	Store bool
}

func TestContextDB(t *testing.T) {

	name := "testdata_context/test_api_db.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}

		control := startBlock()
		createNewAccount(control, "testapi")
		createNewAccount(control, "testapi2")

		//callTestFunction2(control, code, "test_db", "primary_i64_general", []byte{}, "testapi")
		//callTestFunction2(control, code, "test_db", "primary_i64_lowerbound", []byte{}, "testapi")
		//callTestFunction2(control, code, "test_db", "primary_i64_upperbound", []byte{}, "testapi")
		//callTestFunction2(control, code, "test_db", "idx64_general", []byte{}, "testapi")
		//callTestFunction2(control, code, "test_db", "idx64_lowerbound", []byte{}, "testapi")
		//callTestFunction2(control, code, "test_db", "idx64_upperbound", []byte{}, "testapi")
		//
		//action1 := invalidAccessAction{common.N("testapi"), 10, 0, true}
		//actionData1, _ := rlp.EncodeToBytes(&action1)
		//ret := pushAction(control, code, "test_db", "test_invalid_access", actionData1, "testapi")
		//assert.Equal(t, ret, "")
		//
		//action2 := invalidAccessAction{action1.Code, 20, 0, true}
		//actionData2, _ := rlp.EncodeToBytes(&action2)
		//ret = pushAction(control, code, "test_db", "test_invalid_access", actionData2, "testapi2")
		//assert.Equal(t, ret, "db access violation")
		//
		//action1.Store = false
		//actionData3, _ := rlp.EncodeToBytes(&action1)
		//ret = pushAction(control, code, "test_db", "test_invalid_access", actionData3, "testapi")
		//assert.Equal(t, ret, "")
		//
		//action1.Store = true
		//action1.Index = 1
		//actionData1, _ = rlp.EncodeToBytes(&action1)
		//ret = pushAction(control, code, "test_db", "test_invalid_access", actionData1, "testapi")
		//assert.Equal(t, ret, "")
		//
		//action2.Index = 1
		//actionData2, _ = rlp.EncodeToBytes(&action2)
		//ret = pushAction(control, code, "test_db", "test_invalid_access", actionData2, "testapi2")
		//assert.Equal(t, ret, "db access violation")
		//
		//action1.Store = false
		//actionData3, _ = rlp.EncodeToBytes(&action1)
		//ret = pushAction(control, code, "test_db", "test_invalid_access", actionData3, "testapi")
		//assert.Equal(t, ret, "")

		callTestFunctionCheckException2(control, code, "test_db", "idx_double_nan_create_fail", []byte{}, "testapi",
			exception.TransactionException{}.Code(), exception.TransactionException{}.What())

		//callTestFunctionCheckException2(control, code, "test_db", "idx_double_nan_modify_fail", []byte{}, "testapi",
		//	exception.TransactionException{}.Code(), exception.TransactionException{}.What())
		//
		//var loopupType uint32 = 0
		//l,_ := rlp.EncodeToBytes(&loopupType)
		//callTestFunctionCheckException2(control, code, "test_db", "idx_double_nan_lookup_fail", l, "testapi",
		//	exception.TransactionException{}.Code(), exception.TransactionException{}.What())
		//
		//loopupType = 1
		//l,_ = rlp.EncodeToBytes(&loopupType)
		//callTestFunctionCheckException2(control, code, "test_db", "idx_double_nan_lookup_fail", l,"testapi",
		//	exception.TransactionException{}.Code(), exception.TransactionException{}.What())
		//
		//loopupType = 2
		//l,_ = rlp.EncodeToBytes(&loopupType)
		//callTestFunctionCheckException2(control, code, "test_db", "idx_double_nan_lookup_fail", l,"testapi",
		//	exception.TransactionException{}.Code(), exception.TransactionException{}.What())

		//fmt.Println(ret)

		stopBlock(control)

	})
}

func TestContextMultiIndex(t *testing.T) {

	name := "testdata_context/test_api_multi_index.wasm"
	t.Run(filepath.Base(name), func(t *testing.T) {
		code, err := ioutil.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		callTestFunction(code, "test_multi_index", "idx64_general", []byte{})
		//callTestFunction(code, "test_multi_index", "primary_i64_lowerbound", []byte{})

	})
}

func DJBH(str string) uint32 {
	var hash uint32 = 5381
	bytes := []byte(str)

	for i := 0; i < len(bytes); i++ {
		hash = 33*hash ^ uint32(bytes[i])
	}
	return hash
}

func wasmTestAction(cls string, method string) uint64 {
	return uint64(DJBH(cls))<<32 | uint64(DJBH(method))
}

func newApplyContext(control *chain.Controller, action *types.Action) *chain.ApplyContext {

	//pack a singedTrx
	trxHeader := types.TransactionHeader{
		Expiration:       common.MaxTimePointSec(),
		RefBlockNum:      4,
		RefBlockPrefix:   3832731038,
		MaxNetUsageWords: 0,
		MaxCpuUsageMS:    0,
		DelaySec:         0,
	}

	trx := types.Transaction{
		TransactionHeader:     trxHeader,
		ContextFreeActions:    []*types.Action{},
		Actions:               []*types.Action{action},
		TransactionExtensions: []*types.Extension{},
	}
	signedTrx := types.NewSignedTransaction(&trx, []ecc.Signature{}, []common.HexBytes{})
	privateKey, _ := ecc.NewRandomPrivateKey()
	chainIdType := common.ChainIdType(*crypto.NewSha256String("cf057bbfb72640471fd910bcb67639c22df9f92470936cddc1ade0e2f2e7dc4f"))
	signedTrx.Sign(privateKey, &chainIdType)
	trxContext := chain.NewTransactionContext(control, signedTrx, trx.ID(), common.Now())

	//pack a applycontext from control, trxContext and act
	a := chain.NewApplyContext(control, trxContext, action, 0)
	return a
}

func createNewAccount(control *chain.Controller, name string) {

	//action for create a new account
	wif := "5KQwrPbwdL6PhXujxW37FSSQZ1JiwsST4cqQzDeyXtP79zkvFD3"
	privKey, _ := ecc.NewPrivateKey(wif)
	pubKey := privKey.PublicKey()

	creator := chain.NewAccount{
		Creator: common.AccountName(common.N("eosio")),
		Name:    common.AccountName(common.N(name)),
		Owner: types.Authority{
			Threshold: 1,
			Keys:      []types.KeyWeight{{Key: pubKey, Weight: 1}},
		},
		Active: types.Authority{
			Threshold: 1,
			Keys:      []types.KeyWeight{{Key: pubKey, Weight: 1}},
		},
	}

	buffer, _ := rlp.EncodeToBytes(&creator)

	act := types.Action{
		Account: common.AccountName(common.N("eosio")),
		Name:    common.ActionName(common.N("newaccount")),
		Data:    buffer,
		Authorization: []types.PermissionLevel{
			//types.PermissionLevel{Actor: common.AccountName(common.N("eosio.token")), Permission: common.PermissionName(common.N("active"))},
			{Actor: common.AccountName(common.N("eosio")), Permission: common.PermissionName(common.N("active"))},
		},
	}

	a := newApplyContext(control, &act)

	//create new account
	chain.ApplyEosioNewaccount(a)
}

func pushAction(control *chain.Controller, code []byte, cls string, method string, payload []byte, authorizer string) (ret string) {

	wasm := wasmgo.NewWasmGo()
	action := wasmTestAction(cls, method)

	//fmt.Println(cls, method, action)
	//createNewAccount(control, authorizer)
	act := types.Action{
		Account:       common.AccountName(common.N(authorizer)),
		Name:          common.ActionName(action),
		Data:          payload,
		Authorization: []types.PermissionLevel{types.PermissionLevel{Actor: common.AccountName(common.N(authorizer)), Permission: common.PermissionName(common.N("active"))}},
	}

	applyContext := newApplyContext(control, &act)
	codeVersion := crypto.NewSha256Byte([]byte(code))

	defer try.HandleReturn()
	try.Try(func() {
		wasm.Apply(codeVersion, code, applyContext)
	}).Catch(func(e exception.Exception) {
		ret = e.Message()
		try.Return()
	}).End()

	return ""
}

func startBlock() *chain.Controller {
	control := chain.GetControllerInstance()
	blockTimeStamp := common.NewBlockTimeStamp(common.Now())
	control.StartBlock(blockTimeStamp, 0)
	return control
}

func stopBlock(c *chain.Controller) {
	c.Close()
}

func callTestFunction2(control *chain.Controller, code []byte, cls string, method string, payload []byte, authorizer string) (ret string) {

	wasm := wasmgo.NewWasmGo()
	action := wasmTestAction(cls, method)

	act := types.Action{
		Account:       common.AccountName(common.N(authorizer)),
		Name:          common.ActionName(action),
		Data:          payload,
		Authorization: []types.PermissionLevel{types.PermissionLevel{Actor: common.AccountName(common.N(authorizer)), Permission: common.PermissionName(common.N("active"))}},
	}

	applyContext := newApplyContext(control, &act)

	//fmt.Println(cls, method, action)
	codeVersion := crypto.NewSha256Byte([]byte(code))

	defer try.HandleReturn()
	try.Try(func() {
		wasm.Apply(codeVersion, code, applyContext)
	}).Catch(func(e exception.Exception) {
		ret = e.Message()
		try.Return()
	}).End()

	return applyContext.PendingConsoleOutput

}

func callTestFunctionCheckException2(control *chain.Controller, code []byte, cls string, method string, payload []byte, authorizer string, errCode exception.ExcTypes, errMsg string) (ret bool) {

	wasm := wasmgo.NewWasmGo()
	action := wasmTestAction(cls, method)

	// control := chain.GetControllerInstance()
	// blockTimeStamp := common.NewBlockTimeStamp(common.Now())
	// control.StartBlock(blockTimeStamp, 0)

	act := types.Action{
		Account:       common.AccountName(common.N(authorizer)),
		Name:          common.ActionName(action),
		Data:          payload,
		Authorization: []types.PermissionLevel{types.PermissionLevel{Actor: common.AccountName(common.N(authorizer)), Permission: common.PermissionName(common.N("active"))}},
	}

	applyContext := newApplyContext(control, &act)
	codeVersion := crypto.NewSha256Byte([]byte(code))

	//ret := false
	defer try.HandleReturn()
	try.Try(func() {
		wasm.Apply(codeVersion, code, applyContext)
	}).Catch(func(e exception.Exception) {
		if e.Code() == errCode {
			fmt.Println(errMsg)
			ret = true
			try.Return()
		}
	}).End()

	ret = false
	return

}

func callTestFunction(code []byte, cls string, method string, payload []byte) *chain.ApplyContext {

	wasm := wasmgo.NewWasmGo()
	action := wasmTestAction(cls, method)

	control := chain.GetControllerInstance()
	blockTimeStamp := common.NewBlockTimeStamp(common.Now())
	control.StartBlock(blockTimeStamp, 0)

	createNewAccount(control, "testapi")

	act := types.Action{
		Account:       common.AccountName(common.N("testapi")),
		Name:          common.ActionName(action),
		Data:          payload,
		Authorization: []types.PermissionLevel{types.PermissionLevel{Actor: common.AccountName(common.N("testapi")), Permission: common.PermissionName(common.N("active"))}},
	}

	applyContext := newApplyContext(control, &act)

	fmt.Println(cls, method, action)
	codeVersion := crypto.NewSha256Byte([]byte(code))
	wasm.Apply(codeVersion, code, applyContext)

	control.Close()

	return applyContext

}

func callTestFunctionCheckException(code []byte, cls string, method string, payload []byte, errCode exception.ExcTypes, errMsg string) (ret bool) {

	wasm := wasmgo.NewWasmGo()
	action := wasmTestAction(cls, method)

	control := chain.GetControllerInstance()
	blockTimeStamp := common.NewBlockTimeStamp(common.Now())
	control.StartBlock(blockTimeStamp, 0)

	act := types.Action{
		Account:       common.AccountName(common.N("testapi")),
		Name:          common.ActionName(action),
		Data:          payload,
		Authorization: []types.PermissionLevel{types.PermissionLevel{Actor: common.AccountName(common.N("testapi")), Permission: common.PermissionName(common.N("active"))}},
	}

	applyContext := newApplyContext(control, &act)
	codeVersion := crypto.NewSha256Byte([]byte(code))

	//ret := false
	defer try.HandleReturn()
	try.Try(func() {
		wasm.Apply(codeVersion, code, applyContext)
	}).Catch(func(e exception.Exception) {
		if e.Code() == errCode {
			fmt.Println(errMsg)
			ret = true
			control.Close()
			try.Return()
		}
	}).End()

	control.Close()
	ret = false
	return

}

func sigDigest(chainID, payload []byte) []byte {
	h := sha256.New()
	_, _ = h.Write(chainID)
	_, _ = h.Write(payload)
	return h.Sum(nil)
}
