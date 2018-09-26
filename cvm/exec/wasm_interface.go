package exec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"reflect"

	"strings"

	"github.com/eosspark/eos-go/cvm/wasm"
)

var (
	envModule *wasm.Module
	wasmIf    *WasmInterface
	ignore    bool = false
)

type size_t int

func toString(name uint64) string {

	charmap := []byte(".12345abcdefghijklmnopqrstuvwxyz")
	tmp := name

	//var bytes [13]byte
	bytes := []byte{'.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.'}

	for i := 0; i <= 12; i++ {
		var c byte
		if i == 0 {
			c = charmap[tmp&0x0f]
		} else {
			c = charmap[tmp&0x1f]
		}

		bytes[12-i] = c

		if i == 0 {
			tmp >>= 4
		} else {
			tmp >>= 5
		}

	}

	str := string(bytes[:])

	return strings.Trim(str, ".")

}

func char2Symbol(c byte) uint64 {
	if c >= 'a' && c <= 'z' {
		return uint64((c - 'a') + 6)
	}
	if c >= '1' && c <= '5' {
		return uint64((c - '1') + 1)
	}

	return 0
}

func N(str string) uint64 {

	var name uint64
	var i int

	for i = 0; i < len(str) && i < 12; i++ {
		// NOTE: char_to_symbol() returns char type, and without this explicit
		// expansion to uint64 type, the compilation fails at the point of usage
		// of string_to_name(), where the usage requires constant (compile time) expression.
		name |= (char2Symbol(str[i]) & 0x1f) << uint(64-5*(i+1))
	}

	// The for-loop encoded up to 60 high bits into uint64 'name' variable,
	// if (strlen(str) > 12) then encode str[12] into the low (remaining)
	// 4 bits of 'name'
	if i == 12 {
		name |= char2Symbol(str[12]) & 0x0F
	}

	return name
}

type WasmInterface struct {
	context WasmContextInterface
	handles map[string]interface{}
	vm      *VM
}

func NewWasmInterface() *WasmInterface {

	if wasmIf != nil {
		return wasmIf
	}

	wasmInterface := WasmInterface{handles: make(map[string]interface{})}

	wasmInterface.Register("action_data_size", actionDataSize)
	wasmInterface.Register("read_action_data", readActionData)
	wasmInterface.Register("current_receiver", currentReceiver)

	wasmInterface.Register("require_authorization", requireAuthorization)
	wasmInterface.Register("has_authorization", hasAuthorization)
	wasmInterface.Register("require_auth2", requireAuth2)
	wasmInterface.Register("require_recipient", requireRecipient)
	wasmInterface.Register("is_account", isAccount)

	wasmInterface.Register("prints", prints)
	wasmInterface.Register("prints_l", printsl)
	wasmInterface.Register("printi", printi)
	wasmInterface.Register("printui", printui)
	wasmInterface.Register("printi128", printi128)
	wasmInterface.Register("printui128", printui128)
	wasmInterface.Register("printsf", printsf)
	wasmInterface.Register("printdf", printdf)
	wasmInterface.Register("printqf", printqf)
	wasmInterface.Register("printn", printn)
	wasmInterface.Register("printhex", printhex)

	wasmInterface.Register("assert_recover_key", assertRecoverKey)
	wasmInterface.Register("recover_key", recoverKey)
	wasmInterface.Register("assert_sha256", assertSha256)
	wasmInterface.Register("assert_sha1", assertSha1)
	wasmInterface.Register("assert_sha256", assertSha256)
	wasmInterface.Register("assert_sha512", assertSha512)
	wasmInterface.Register("assert_ripemd160", assertRipemd160)
	wasmInterface.Register("sha1", sha1)
	wasmInterface.Register("sha256", sha256)
	wasmInterface.Register("sha512", sha512)
	wasmInterface.Register("ripemd160", ripemd160)

	wasmInterface.Register("memcpy", memcpy)
	wasmInterface.Register("memmove", memmove)
	wasmInterface.Register("memcmp", memcmp)
	wasmInterface.Register("memset", memset)

	wasmInterface.Register("check_transaction_authorization", checkTransactionAuthorization)
	wasmInterface.Register("check_permission_authorization", checkPermissionAuthorization)
	wasmInterface.Register("get_permission_last_used", getPermissionLastUsed)
	wasmInterface.Register("get_account_creation_time", getAccountCreationTime)

	wasmInterface.Register("is_feature_active", isFeatureActive)
	wasmInterface.Register("activate_feature", activateFeature)
	wasmInterface.Register("set_resource_limits", setResourceLimits)
	wasmInterface.Register("get_resource_limits", getResourceLimits)
	wasmInterface.Register("get_blockchain_parameters_packed", getBlockchainParametersPacked)
	wasmInterface.Register("set_blockchain_parameters_packed", setBlockchainParametersPacked)
	wasmInterface.Register("is_privileged", isPrivileged)
	wasmInterface.Register("set_privileged", setPrivileged)

	wasmInterface.Register("set_proposed_producers", setProposedProducers)
	wasmInterface.Register("get_active_producers", getActiveProducers)

	wasmInterface.Register("checktime", checkTime)
	wasmInterface.Register("current_time", currentTime)
	wasmInterface.Register("publication_time", publicationTime)
	wasmInterface.Register("abort", abort)
	wasmInterface.Register("eosio_assert", eosioAssert)
	wasmInterface.Register("eosio_assert_message", eosioAssertMessage)
	wasmInterface.Register("eosio_assert_code", eosioAssertCode)
	wasmInterface.Register("eosio_exit", eosioExit)

	wasmInterface.Register("send_inline", sendInline)
	wasmInterface.Register("send_context_free_inline", sendContextFreeInline)
	wasmInterface.Register("send_deferred", sendDeferred)
	wasmInterface.Register("cancel_deferred", cancelDeferred)
	wasmInterface.Register("read_transaction", readTransaction)
	wasmInterface.Register("transaction_size", transactionSize)
	wasmInterface.Register("expiration", expiration)
	wasmInterface.Register("tapos_block_num", taposBlockNum)
	wasmInterface.Register("tapos_block_prefix", taposBlockPrefix)
	wasmInterface.Register("get_action", getAction)
	wasmInterface.Register("get_context_free_data", getContextFreeData)

	wasmIf = &wasmInterface

	return wasmIf
}

func (wasmInterface *WasmInterface) Apply(code_id string, code []byte, context WasmContextInterface) {
	wasmInterface.context = context

	bf := bytes.NewReader(code)

	m, err := wasm.ReadModule(bf, wasmInterface.importer)
	if err != nil {
		log.Fatalf("could not read module: %v", err)
	}

	// if *verify {
	// 	err = validate.VerifyModule(m)
	// 	if err != nil {
	// 		log.Fatalf("could not verify module: %v", err)
	// 	}
	// }

	if m.Export == nil {
		log.Fatalf("module has no export section")
	}

	vm, err := NewVM(m, wasmInterface)
	if err != nil {
		log.Fatalf("could not create VM: %v", err)
	}

	e, _ := m.Export.Entries["apply"]
	i := int64(e.Index)
	//fidx := m.Function.Types[int(i)]
	//ftype := m.Types.Entries[int(fidx)]

	wasmInterface.vm = vm

	args := make([]uint64, 3)
	args[0] = uint64(context.GetReceiver())
	args[1] = uint64(context.GetCode())
	args[2] = uint64(context.GetAct())

	//o, err := vm.ExecCode(i, args[0], args[1], args[2])
	o, err := vm.ExecCode(i, args[0], args[1], args[2])
	if err != nil {
		fmt.Printf("\n")
		log.Printf("err=%v", err)
	}
	//if len(ftype.ReturnTypes) == 0 {
	//	fmt.Printf("\n")
	//}
	if o != nil {
		fmt.Printf("%[1]v (%[1]T)\n", o)
	}
}

func (wasmInterface *WasmInterface) Register(name string, handler interface{}) bool {
	if _, ok := wasmInterface.handles[name]; ok {
		return false
	}

	wasmInterface.handles[name] = handler
	return true
}

func (wasmInterface *WasmInterface) Add(handles map[string]interface{}) bool {
	for k, v := range handles {
		if _, ok := wasmInterface.handles[k]; !ok {
			wasmInterface.handles[k] = v
		}
	}
	return true
}

func (wasmInterface *WasmInterface) GetHandles() map[string]interface{} {
	return wasmInterface.handles
}

func (wasmInterface *WasmInterface) GetHandle(name string) interface{} {

	if _, ok := wasmInterface.handles[name]; ok {
		return wasmInterface.handles[name]
	}

	return nil
}

// func importer(name string) (*wasm.Module, error) {
// 	f, err := os.Open(name + ".wasm")
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer f.Close()
// 	m, err := wasm.ReadModule(f, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// err = validate.VerifyModule(m)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	return m, nil
// }

func (wasmInterface *WasmInterface) importer(name string) (*wasm.Module, error) {

	if name == "env" {
		if envModule != nil {
			return envModule, nil
		}

		count := len(wasmInterface.handles)

		m := wasm.NewModule()
		m.Types.Entries = make([]wasm.FunctionSig, count)
		m.FunctionIndexSpace = make([]wasm.Function, count)
		m.Export.Entries = make(map[string]wasm.ExportEntry, count)

		i := 0
		for k, v := range wasmInterface.handles {

			// 1st param is *wasm_interface should be deleted
			numIn := reflect.TypeOf(v).NumIn() - 1
			args := make([]wasm.ValueType, numIn)
			for j := int(0); j < numIn; j++ {
				args[j] = reflect2wasm(reflect.TypeOf(v).In(j + 1).Kind())
			}

			numOut := reflect.TypeOf(v).NumOut()
			rtrns := make([]wasm.ValueType, numOut)
			for m := int(0); m < numOut; m++ {
				rtrns[m] = reflect2wasm(reflect.TypeOf(v).Out(m).Kind())
			}

			m.Types.Entries[i] = wasm.FunctionSig{
				//Form:        0,
				ParamTypes:  args,
				ReturnTypes: rtrns,
			}

			m.FunctionIndexSpace[i] = wasm.Function{
				Sig:  &m.Types.Entries[i],
				Host: reflect.ValueOf(v),
				Body: &wasm.FunctionBody{},
				Name: k,
			}

			m.Export.Entries[k] = wasm.ExportEntry{
				FieldStr: k,
				Kind:     wasm.ExternalFunction,
				Index:    uint32(i),
			}

			i++

		}

		envModule = m

		return envModule, nil

	}

	return nil, errors.New("Only env module availible")

}

// const (
// 	Invalid Kind = iota
// 	Bool
// 	Int
// 	Int8
// 	Int16
// 	Int32
// 	Int64
// 	Uint
// 	Uint8
// 	Uint16
// 	Uint32
// 	Uint64
// 	Uintptr
// 	Float32
// 	Float64
// 	Complex64
// 	Complex128
// 	Array
// 	Chan
// 	Func
// 	Interface
// 	Map
// 	Ptr
// 	Slice
// 	String
// 	Struct
// 	UnsafePointer
// )

func reflect2wasm(kind reflect.Kind) wasm.ValueType {

	switch kind {
	case reflect.Float64:
		return wasm.ValueTypeF64
	case reflect.Float32:
		return wasm.ValueTypeF32
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		return wasm.ValueTypeI32
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Struct:
		return wasm.ValueTypeI32
	case reflect.Ptr:
		return wasm.ValueTypeI64
	default:
		//panic(fmt.Sprintf("exec: return value %d invalid kind=%v", kind))
		return wasm.ValueTypeI64
	}
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
func i2b(i int) bool {
	if i > 0 {
		return true
	}
	return false
}
func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// func copyMemory(w *WasmInterface, dest int, src int, bufferSize int) {
// 	copy(w.vm.memory[dest:dest+bufferSize], w.vm.memory[src:src+bufferSize])
// }
func setMemory(w *WasmInterface, mIndex int, dIndex int, data []byte, bufferSize int) {
	copy(w.vm.memory[mIndex:mIndex+bufferSize], data[dIndex:dIndex+bufferSize])
}

//func getMemory(w *WasmInterface, mIndex int, dIndex int, data []byte, bufferSize int) {
func getMemory(w *WasmInterface, mIndex int, bufferSize int) []byte {
	data := make([]byte, bufferSize)
	copy(data[0:0+bufferSize], w.vm.memory[0:0+bufferSize])

	return data
}

func setUint64(w *WasmInterface, index int, val uint64) {
	c := make([]byte, 8)
	binary.LittleEndian.PutUint64(c, val)

	copy(w.vm.memory[index:index+8], c[:])
}

func getUint64(w *WasmInterface, index int) uint64 {
	c := make([]byte, 8)
	copy(c[:], w.vm.memory[index:index+8])

	return binary.LittleEndian.Uint64(c[:])

}

func getStringSize(w *WasmInterface, index int) int {
	var size int
	var i int
	for i = 0; i < 512; i++ {
		if w.vm.memory[index+i] == 0 {
			break
		}
		size++
	}

	return size
}

func getString(w *WasmInterface, index int) string {
	return string(w.vm.memory[index : index+getStringSize(w, index)])
}
func getBytes(w *WasmInterface, index int, datalen int) []byte {
	return w.vm.memory[index : index+datalen]
}
func setSha256(w *WasmInterface, index int, sha256 []byte) {
	copy(w.vm.memory[index:index+32], sha256[0:32])
}
func getSha256(w *WasmInterface, index int) []byte { return w.vm.memory[index : index+32] }
func setSha512(w *WasmInterface, index int, sha512 []byte) {
	copy(w.vm.memory[index:index+64], sha512[0:64])
}
func getSha512(w *WasmInterface, index int) []byte     { return w.vm.memory[index : index+64] }
func setSha1(w *WasmInterface, index int, sha1 []byte) { copy(w.vm.memory[index:index+20], sha1[0:20]) }
func getSha1(w *WasmInterface, index int) []byte       { return w.vm.memory[index : index+20] }
func setRipemd160(w *WasmInterface, index int, ripemd160 []byte) {
	copy(w.vm.memory[index:index+20], ripemd160[0:20])
}
func getRipemd160(w *WasmInterface, index int) []byte { return w.vm.memory[index : index+20] }