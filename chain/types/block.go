package types

import (
	"encoding/json"
	"github.com/eosspark/eos-go/common"
	"github.com/eosspark/eos-go/crypto"
	"github.com/eosspark/eos-go/crypto/ecc"
	"github.com/eosspark/eos-go/crypto/rlp"
)

type TransactionStatus uint8

const (
	TransactionStatusExecuted TransactionStatus = iota ///< succeed, no error handler executed
	TransactionStatusSoftFail                          ///< objectively failed (not executed), error handler executed
	TransactionStatusHardFail                          ///< objectively failed and error handler objectively failed thus no state change
	TransactionStatusDelayed                           ///< transaction delayed
	TransactionStatusExpired
	TransactionStatusUnknown = TransactionStatus(255)
)

type BlockStatus uint8

const (
	Irreversible BlockStatus = iota ///< this block has already been applied before by this node and is considered irreversible
	Validated                       ///< this is a complete block signed by a valid producer and has been previously applied by this node and therefore validated but it is not yet irreversible
	Complete                        ///< this is a complete block signed by a valid producer but is not yet irreversible nor has it yet been applied by this node
	Incomplete                      ///< this is an incomplete block (either being produced by a producer or speculatively produced by a node)
)

func (s *TransactionStatus) UnmarshalJSON(data []byte) error {
	var decoded string
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	switch decoded {
	case "executed":
		*s = TransactionStatusExecuted
	case "soft_fail":
		*s = TransactionStatusSoftFail

	case "hard_fail":
		*s = TransactionStatusHardFail
	case "delayed":
		*s = TransactionStatusDelayed
	default:
		*s = TransactionStatusUnknown
	}
	return nil
}

func (s TransactionStatus) MarshalJSON() (data []byte, err error) {
	out := "unknown"
	switch s {
	case TransactionStatusExecuted:
		out = "executed"
	case TransactionStatusSoftFail:
		out = "soft_fail"
	case TransactionStatusHardFail:
		out = "hard_fail"
	case TransactionStatusDelayed:
		out = "delayed"
	}
	return json.Marshal(out)
}
func (s TransactionStatus) String() string {

	switch s {
	case TransactionStatusExecuted:
		return "executed"
	case TransactionStatusSoftFail:
		return "soft fail"
	case TransactionStatusHardFail:
		return "hard fail"
	case TransactionStatusDelayed:
		return "delayed"
	default:
		return "unknown"
	}

}

// type ShardLock struct {
// 	AccountName common.AccountName `json:"account_name"`
// 	ScopeName   common.ScopeName   `json:"scope_name"`
// }

// type ShardSummary struct {
// 	ReadLocks    []ShardLock          `json:"read_locks"`
// 	WriteLocks   []ShardLock          `json:"write_locks"`
// 	Transactions []TransactionReceipt `json:"transactions"`
// }

// type Cycles []ShardSummary
// type RegionSummary struct {
// 	Region        uint16   `json:"region"`
// 	CyclesSummary []Cycles `json:"cycles_summary"`
// }

type TransactionReceiptHeader struct {
	Status        TransactionStatus `json:"status"`
	CpuUsageUs    uint32            `json:"cpu_usage_us"`
	NetUsageWords uint32            `json:"net_usage_words" eos:"vuint32"`
}

type TransactionReceipt struct {
	TransactionReceiptHeader
	Trx TransactionWithID `json:"trx" eos:"trxID"`
}

type SignedBlock struct {
	SignedBlockHeader `multiIndex:"inline"`
	Transactions      []TransactionReceipt `json:"transactions"`
	BlockExtensions   []Extension          `json:"block_extensions"`
}

func NewSignedBlock() *SignedBlock {
	return &SignedBlock{SignedBlockHeader: *NewSignedBlockHeader()}
}

func NewSignedBlock1(h *SignedBlockHeader) *SignedBlock {
	return &SignedBlock{SignedBlockHeader: *h}
}

/*func (m *SignedBlock) String() string {
	return "SignedBlock"
}*/

type ProducerConfirmation struct {
	BlockID     common.BlockIdType
	BlockDigest [4]uint64
	Producer    common.AccountName
	Sig         ecc.Signature
}

type Optional struct {
	Valid bool
	Pair  map[common.ChainIdType][]ecc.PublicKey
}

type TransactionWithID struct {
	PackedTransaction *PackedTransaction       `json:"packed_transaction" eos:"tag0"`
	TransactionID     common.TransactionIdType `json:"transaction_id" eos:"tag1"`
}

func (t TransactionWithID) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{
		t.PackedTransaction,
		t.TransactionID,
	})
}

func (t *TransactionWithID) UnmarshalJSON(data []byte) error {
	var packed PackedTransaction
	if data[0] == '{' {
		if err := json.Unmarshal(data, &packed); err != nil {
			return err
		}
		*t = TransactionWithID{
			PackedTransaction: &packed,
		}
		return nil
	} else if data[0] == '"' {
		var id common.TransactionIdType
		err := json.Unmarshal(data, &id)
		if err != nil {
			return err
		}
		*t = TransactionWithID{
			TransactionID: id,
		}
		return nil
	}
	return nil
}

func NewTransactionReceiptHeader() *TransactionReceiptHeader {
	trh := TransactionReceiptHeader{}
	trh.Status = TransactionStatusHardFail
	return &trh
}

func NewTransactionReceiptHeader2(status TransactionStatus) *TransactionReceiptHeader {
	trh := TransactionReceiptHeader{}
	trh.Status = status
	return &trh
}

func NewTransactionReceipt() *TransactionReceipt {
	tr := TransactionReceipt{}
	tr.TransactionReceiptHeader = *NewTransactionReceiptHeader()
	return &tr
}

func NewTransactionReceipt2(tid common.TransactionIdType) *TransactionReceipt {
	tr := TransactionReceipt{}
	tr.TransactionReceiptHeader = *NewTransactionReceiptHeader2(TransactionStatusExecuted)
	tr.Trx.TransactionID = tid
	return &tr
}

func NewTransactionReceipt3(ptrx PackedTransaction) *TransactionReceipt {
	tr := TransactionReceipt{}
	tr.TransactionReceiptHeader = *NewTransactionReceiptHeader2(TransactionStatusExecuted)
	tr.Trx.PackedTransaction = &ptrx
	return &tr
}

func (t *TransactionReceipt) Digest() common.DigestType { //TODO crypto.Sha256??
	enc := crypto.NewSha256()
	status, _ := rlp.EncodeToBytes(t.Status)
	cpuUsageUs, _ := rlp.EncodeToBytes(t.CpuUsageUs)
	netUsageWords, _ := rlp.EncodeToBytes(t.NetUsageWords)

	enc.Write(status)
	enc.Write(cpuUsageUs)
	enc.Write(netUsageWords)

	if t.Trx.TransactionID != common.TransactionIdType(*crypto.NewSha256Nil()) {
		trxID, _ := rlp.EncodeToBytes(t.Trx.TransactionID)
		enc.Write(trxID)
	} else {
		packedTrx, _ := rlp.EncodeToBytes(t.Trx.PackedTransaction.PackedDigest())
		enc.Write(packedTrx)
	}

	out := crypto.NewSha256Byte(enc.Sum(nil))
	return common.DigestType(*out)
}
