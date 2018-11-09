package types

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"github.com/eosspark/eos-go/common"
	arithmetic "github.com/eosspark/eos-go/common/arithmetic_types"
	"github.com/eosspark/eos-go/crypto"
	"github.com/eosspark/eos-go/crypto/ecc"
	"github.com/eosspark/eos-go/crypto/rlp"
	"github.com/eosspark/eos-go/exception"
	"io/ioutil"
	"math"
	"github.com/eosspark/eos-go/exception/try"
)

type Extension struct {
	Type uint16          `json:"type"`
	Data common.HexBytes `json:"data"`
}

func init() {

}

/**
 *  TransactionHeader contains the fixed-sized data
 *  associated with each transaction. It is separated from
 *  the transaction body to facilitate partial parsing of
 *  transactions without requiring dynamic memory allocation.
 *
 *  All transactions have an expiration time after which they
 *  may no longer be included in the blockchain. Once a block
 *  with a block_header::timestamp greater than expiration is
 *  deemed irreversible, then a user can safely trust the transaction
 *  will never be included.
 *

 *  Each region is an independent blockchain, it is included as routing
 *  information for inter-blockchain communication. A contract in this
 *  region might generate or authorize a transaction intended for a foreign
 *  region.
 */
type TransactionHeader struct {
	Expiration     common.TimePointSec `json:"expiration"`
	RefBlockNum    uint16              `json:"ref_block_num"`
	RefBlockPrefix uint32              `json:"ref_block_prefix"`

	MaxNetUsageWords uint32 `json:"max_net_usage_words" eos:"vuint32"`
	MaxCpuUsageMS    uint8  `json:"max_cpu_usage_ms"`
	DelaySec         uint32 `json:"delay_sec" eos:"vuint32"` // number of secs to delay, making it cancellable for that duration
}

func (t TransactionHeader) GetRefBlocknum(headBlocknum uint32) uint32 {
	return headBlocknum/0xffff*0xffff + headBlocknum%0xffff
}

func (t TransactionHeader) VerifyReferenceBlock(referenceBlock *common.BlockIdType) bool {
	return t.RefBlockNum == uint16(common.EndianReverseU32(uint32(referenceBlock.Hash[0]))) &&
		t.RefBlockPrefix == uint32(referenceBlock.Hash[1])
}

func (t TransactionHeader) Validate() {
	if t.MaxNetUsageWords >= uint32(0xffffffff)/8 {
		panic("declared max_net_usage_words overflows when expanded to max net usage")
	}
}

func (t *TransactionHeader) SetReferenceBlock(referenceBlock *common.BlockIdType) {
	first := common.EndianReverseU32(uint32(referenceBlock.Hash[0]))
	t.RefBlockNum = uint16(first)
	t.RefBlockPrefix = uint32(referenceBlock.Hash[1])
}

var recoveryCache = make(map[ecc.Signature]CachedPubKey)

type CachedPubKey struct {
	TrxID  common.TransactionIdType `json:"trx_id"`
	PubKey ecc.PublicKey            `json:"pub_key"`
	Sig    ecc.Signature            `json:"sig"`
}

//Transaction consits of a set of messages which must all be applied or
//all are rejected. These messages have access to data within the given
//read and write scopes.
type Transaction struct { // WARN: is a `variant` in C++, can be a SignedTransaction or a Transaction.
	TransactionHeader

	ContextFreeActions    []*Action    `json:"context_free_actions"`
	Actions               []*Action    `json:"actions"`
	TransactionExtensions []*Extension `json:"transaction_extensions"`
}

func (t *Transaction) ID() common.TransactionIdType {
	b, err := rlp.EncodeToBytes(t)
	if err != nil {
		fmt.Println("Transaction ID() is error :", err.Error()) //TODO
	}
	return common.TransactionIdType(crypto.Hash256(b))
}

func (t *Transaction) SigDigest(chainID *common.ChainIdType, cfd []common.HexBytes) []byte { //common.DigestType {
	enc := crypto.NewSha256()
	chainIDByte, err := rlp.EncodeToBytes(chainID)
	if err != nil {
		fmt.Println(err)
	}
	thByte, err := rlp.EncodeToBytes(t)
	if err != nil {
		fmt.Println(err)
	}

	enc.Write(chainIDByte)
	enc.Write(thByte)
	if len(cfd) > 0 {
		enc.Write(crypto.Hash256(cfd).Bytes())
	} else {
		enc.Write(crypto.NewSha256Nil().Bytes())
	}

	//hashed := enc.Sum(nil)   //TODO []byte or DigestType?????
	//out :=crypto.NewSha256Byte(hashed)
	//return common.DigestType(*out)

	return enc.Sum(nil)
}

//allowDuplicateKeys = false
//useCache= true
func (t *Transaction) GetSignatureKeys(signatures []ecc.Signature, chainID *common.ChainIdType, cfd []common.HexBytes,
	allowDuplicateKeys bool, useCache bool) common.FlatSet /*(recoveredPubKeys []*ecc.PublicKey)*/ {
	const recoveryCacheSize common.SizeT = 1000
	recoveredPubKeys := common.FlatSet{}
	recov := ecc.PublicKey{}
	digest := t.SigDigest(chainID, cfd)
	for _, sig := range signatures {
		if useCache {
			it, ok := recoveryCache[sig]
			if !ok || it.TrxID != t.ID() {
				recov, _ = sig.PublicKey(digest)
				recoveryCache[sig] = CachedPubKey{t.ID(), recov, sig} //could fail on dup signatures; not a problem
			} else {
				recov = it.PubKey
			}
		} else {
			recov, _ = sig.PublicKey(digest)
		}
		_, exist := recoveredPubKeys.Insert(&recov)
		try.EosAssert(allowDuplicateKeys || !exist, &exception.TxDuplicateSig{},
			"transaction includes more than one signature signed using the same key associated with public key: %s}", recov)
	}
	/*if useCache {
		for len(t.RecoveryCache) > int(recoveryCacheSize) {
			recovery_cache.erase( recovery_cache.begin() )
		}
	}*/
	return recoveredPubKeys
}

func (t *Transaction) TotalActions() uint32 {
	return uint32(len(t.ContextFreeActions) + len(t.Actions))
}

func (tx *Transaction) FirstAuthorizor() common.AccountName {
	for _, a := range tx.Actions {
		for _, auth := range a.Authorization {
			return auth.Actor
		}
	}
	return common.AccountName(0)
}

type SignedTransaction struct {
	Transaction

	Signatures      []ecc.Signature   `json:"signatures"`
	ContextFreeData []common.HexBytes `json:"context_free_data"`
}

func NewSignedTransaction(tx *Transaction, signature []ecc.Signature, contextFreeData []common.HexBytes) *SignedTransaction {
	return &SignedTransaction{
		Transaction:     *tx,
		Signatures:      signature,
		ContextFreeData: contextFreeData,
	}
}

func NewSignedTransactionNil() *SignedTransaction {
	return &SignedTransaction{
		Signatures:      make([]ecc.Signature, 0),
		ContextFreeData: make([]common.HexBytes, 0),
	}
}

func (s *SignedTransaction) Sign(key *ecc.PrivateKey, chainID *common.ChainIdType) ecc.Signature {
	signature, err := key.Sign(s.Transaction.SigDigest(chainID, s.ContextFreeData))
	if err != nil {
		fmt.Println(err) //TODO
	}
	s.Signatures = append(s.Signatures, signature)
	return signature
}
func (s *SignedTransaction) SignWithoutAppend(key ecc.PrivateKey, chainID *common.ChainIdType) ecc.Signature {
	signature, err := key.Sign(s.Transaction.SigDigest(chainID, s.ContextFreeData))
	if err != nil {
		fmt.Println(err) //TODO
	}
	return signature
}

//allowDeplicateKeys =false,useCache=true
func (st *SignedTransaction) GetSignatureKeys(chainID *common.ChainIdType, allowDeplicateKeys bool, useCache bool) common.FlatSet {
	return st.Transaction.GetSignatureKeys(st.Signatures, chainID, st.ContextFreeData, allowDeplicateKeys, useCache)
}

func (s *SignedTransaction) String() string {

	data, err := json.Marshal(s)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// PackedTransaction represents a fully packed transaction, with
// signatures, and all. They circulate like that on the P2P net, and
// that's how they are stored.
type PackedTransaction struct {
	Signatures            []ecc.Signature        `json:"signatures"`
	Compression           common.CompressionType `json:"compression"` // in C++, it's an enum, not sure how it Binary-marshals..
	PackedContextFreeData common.HexBytes        `json:"packed_context_free_data"`
	PackedTrx             common.HexBytes        `json:"packed_trx"`
	UnpackedTrx           *Transaction           `eos:"-"`
}

func NewPackedTransactionByTrx(t *Transaction, compression common.CompressionType) *PackedTransaction {
	ptrx := &PackedTransaction{}
	ptrx.SetTransaction(t, compression)
	return ptrx
}

//compression := common.CompressionNone
func NewPackedTransactionBySignedTrx(t *SignedTransaction, compression common.CompressionType) *PackedTransaction {
	ptrx := &PackedTransaction{
		Signatures: t.Signatures,
	}
	ptrx.SetTransactionWithCFD(t, &t.ContextFreeData, compression)
	return ptrx
}

//func (p *PackedTransaction) SetTransactionWithCFD(t *Transaction, cfd *[]common.HexBytes, compression common.CompressionType) {
func (p *PackedTransaction) SetTransactionWithCFD(t *SignedTransaction, cfd *[]common.HexBytes, compression common.CompressionType) {
	//try{}
	switch compression {
	case common.CompressionNone:
		p.PackedTrx = packTransaction(&t.Transaction)
		p.PackedContextFreeData = packContextFreeData(cfd)
	case common.CompressionZlib:
		p.PackedTrx = zlibCompressTransaction(&t.Transaction)
		p.PackedContextFreeData = zlibCompressContextFreeData(cfd)
	default:
		//EOS_THROW(unknown_transaction_compression, "Unknown transaction compression algorithm");
		panic("Unknown transaction compression algorithm")
	}

	p.Compression = compression
}
func (p *PackedTransaction) GetUnprunableSize() (size uint32) {
	size = common.DefaultConfig.FixedNetOverheadOfPackedTrx
	size += uint32(len(p.PackedTrx))
	//EOS_ASSERT( size <= std::numeric_limits<uint32_t>::max(), tx_too_big, "packed_transaction is too big" );
	if size > math.MaxUint32 {
		panic("packed_transaction is too big")
	}
	return
}

func (p *PackedTransaction) GetPrunableSize() uint32 {
	size, err := rlp.EncodeSize(p.Signatures)
	if err != nil {
		panic(err)
	}
	size += len(p.PackedContextFreeData)
	//EOS_ASSERT( size <= std::numeric_limits<uint32_t>::max(), tx_too_big, "packed_transaction is too big" );
	if size > math.MaxUint32 {
		panic("packed_transaction is too big")
	}
	return uint32(size)
}

func (p *PackedTransaction) PackedDigest() common.DigestType {
	prunable := crypto.NewSha256()
	result, err := rlp.EncodeToBytes(p.Signatures)
	if err != nil {
		errout := fmt.Sprintf("PackedDigest:Signatures error:%s", err)
		panic(errout)
	}
	prunable.Write(result)
	result, err = rlp.EncodeToBytes(p.PackedContextFreeData)
	if err != nil {
		errout := fmt.Sprintf("PackedDigest:PackedContextFreeData error:%s", err)
		panic(errout)
	}
	prunable.Write(result)

	prunableResult := prunable.Sum(nil)

	enc := crypto.NewSha256()
	result, err = rlp.EncodeToBytes(p.Compression)
	if err != nil {
		errout := fmt.Sprintf("PackedDigest:Compression error:%s", err)
		panic(errout)
	}
	enc.Write(result)

	result, err = rlp.EncodeToBytes(p.PackedTrx)
	if err != nil {
		errout := fmt.Sprintf("PackedDigest:PackedTrx error:%s", err)
		panic(errout)
	}
	enc.Write(result)
	enc.Write(prunableResult)

	hashed := enc.Sum(nil) //TODO []byte or DigestType?????
	out := crypto.NewSha256Byte(hashed)
	return common.DigestType(*out)
}

func (p *PackedTransaction) GetRawTransaction() common.HexBytes {
	//try{}
	switch p.Compression {
	case common.CompressionNone:
		return p.PackedTrx
	case common.CompressionZlib:
		return zlibDecompress(&p.PackedTrx)
	default:
		//EOS_THROW(unknown_transaction_compression, "Unknown transaction compression algorithm");
		panic("Unknown transaction compression algorithm")
	}
}

func (p *PackedTransaction) GetContextFreeData() []common.HexBytes {
	//try{}
	switch p.Compression {
	case common.CompressionNone:
		return unpackContextFreeData(&p.PackedContextFreeData)
	case common.CompressionZlib:
		return zlibDecompressContextFreeData(&p.PackedContextFreeData)
	default:
		//EOS_THROW(unknown_transaction_compression, "Unknown transaction compression algorithm");
		panic("Unknown transaction compression algorithm")
	}
}

func (p *PackedTransaction) Expiration() common.TimePointSec {
	p.localUnpack()
	return p.UnpackedTrx.Expiration
}

func (p *PackedTransaction) ID() common.TransactionIdType {
	p.localUnpack()
	return p.GetTransaction().ID()
}

func (p *PackedTransaction) GetUncachedID() common.TransactionIdType {
	raw := p.GetRawTransaction()
	tx := Transaction{}
	err := rlp.DecodeBytes([]byte(raw), &tx)
	if err != nil {
		panic(err)
	}
	return tx.ID()
}

func (p *PackedTransaction) localUnpack() {
	//try{}//TODO
	if p.UnpackedTrx == nil {
		switch p.Compression {
		case common.CompressionNone:
			p.UnpackedTrx = unpackTransaction(p.PackedTrx)
		case common.CompressionZlib:
			p.UnpackedTrx = zlibDecompressTransaction(&p.PackedTrx)
		default:
			//EOS_THROW(unknown_transaction_compression, "Unknown transaction compression algorithm");
			panic("Unknown transaction compression algorim")
		}
	}
}

func (p *PackedTransaction) GetTransaction() *Transaction {
	p.localUnpack()
	return p.UnpackedTrx
}

func (p *PackedTransaction) GetSignedTransaction() *SignedTransaction {
	//try{}
	switch p.Compression {
	case common.CompressionNone:
		return NewSignedTransaction(p.GetTransaction(), p.Signatures, unpackContextFreeData(&p.PackedContextFreeData))
	case common.CompressionZlib:
		return NewSignedTransaction(p.GetTransaction(), p.Signatures, zlibDecompressContextFreeData(&p.PackedContextFreeData))
	default:
		//EOS_THROW(unknown_transaction_compression, "Unknown transaction compression algorithm");
		panic("Unknown transaction compression algorithm")
	}
}

func (p *PackedTransaction) SetTransaction(t *Transaction, compression common.CompressionType) {
	//try{}
	switch compression {
	case common.CompressionNone:
		p.PackedTrx = packTransaction(t)
	case common.CompressionZlib:
		p.PackedTrx = zlibCompressTransaction(t)
	default:
		//EOS_THROW(unknown_transaction_compression, "Unknown transaction compression algorithm");
		panic("Unknown transaction compression algorithm")
	}

	p.PackedContextFreeData = nil //TODO clear()
	p.Compression = compression
}

func unpackContextFreeData(data *common.HexBytes) (out []common.HexBytes) {
	if len(*data) == 0 {
		return
	}
	err := rlp.DecodeBytes([]byte(*data), out)
	if err != nil {
		panic(err)
	}
	return
}
func unpackTransaction(data common.HexBytes) (tx *Transaction) {
	err := rlp.DecodeBytes(data, &tx)
	if err != nil {
		panic(err)
	}
	return
}

func zlibDecompress(data *common.HexBytes) common.HexBytes { //TODO
	in := bytes.NewReader(*data)
	r, err := zlib.NewReader(in)
	if err != nil {
		panic(err)
	}
	result, _ := ioutil.ReadAll(r)
	r.Close()
	return result
}

func zlibDecompressContextFreeData(data *common.HexBytes) []common.HexBytes {
	if len(*data) == 0 {
		return []common.HexBytes{}
	}
	packedData := zlibDecompress(data)
	return unpackContextFreeData(&packedData)
}

func zlibDecompressTransaction(data *common.HexBytes) *Transaction {
	packedTrax := zlibDecompress(data)
	return unpackTransaction(packedTrax)
}

func packTransaction(t *Transaction) (out []byte) { //Bytes
	out, err := rlp.EncodeToBytes(t)
	if err != nil {
		panic(err)
	}
	return
}

func packContextFreeData(cfd *[]common.HexBytes) (out []byte) {
	if len(*cfd) == 0 {
		return []byte{}
	}
	out, err := rlp.EncodeToBytes(cfd)
	if err != nil {
		panic(err)
	}
	return
}

func zlibCompressContextFreeData(cfd *[]common.HexBytes) (out []byte) {
	if len(*cfd) == 0 {
		return
	}
	in := packContextFreeData(cfd)

	return zlibCompress(in)
}

func zlibCompressTransaction(t *Transaction) []byte {
	in := packTransaction(t)
	return zlibCompress(in)
}

func zlibCompress(data []byte) []byte {
	var in bytes.Buffer
	w, err := zlib.NewWriterLevel(&in, zlib.BestCompression)
	if err != nil {
		panic(err)
	}
	w.Write(data)
	w.Close()
	return in.Bytes()
}

/**
 *  When a transaction is generated it can be scheduled to occur
 *  in the future. It may also fail to execute for some reason in
 *  which case the sender needs to be notified. When the sender
 *  sends a transaction they will assign it an ID which will be
 *  passed back to the sender if the transaction fails for some
 *  reason.
 */
type DeferredTransaction struct {
	*SignedTransaction

	SenderID     arithmetic.Uint128  `json:"sender_id"` // ID assigned by sender of generated, accessible via WASM api when executing normal or error
	Sender       common.AccountName  `json:"sender"`    // receives error handler callback
	Payer        common.AccountName  `json:"payer"`
	ExecuteAfter common.TimePointSec `json::execute_after` // delayed execution
}

func NewDeferredTransaction(senderID arithmetic.Uint128, sender common.AccountName, payer common.AccountName,
	executeAfter common.TimePointSec, txn *SignedTransaction) *DeferredTransaction {
	return &DeferredTransaction{
		SignedTransaction: txn,
		SenderID:          senderID,
		Sender:            sender,
		Payer:             payer,
		ExecuteAfter:      executeAfter,
	}
}

type DeferredReference struct {
	Sender   common.AccountName `json:"sender"`
	SenderID arithmetic.Uint128 `json:"sender_id"`
}

func NewDeferredReference(sender common.AccountName, senderID arithmetic.Uint128) *DeferredReference {
	return &DeferredReference{
		Sender:   sender,
		SenderID: senderID,
	}
}

func TransactionIDtoSenderID(tid common.TransactionIdType) arithmetic.Uint128 {
	return arithmetic.Uint128{tid.Hash[3], tid.Hash[2]}
}
