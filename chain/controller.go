package chain

import (
	"fmt"
	"time"


	"github.com/eosspark/eos-go/chain/config"
	"github.com/eosspark/eos-go/chain/types"
	"github.com/eosspark/eos-go/common"
	"github.com/eosspark/eos-go/cvm/exec"
	"github.com/eosspark/eos-go/db"
	"github.com/eosspark/eos-go/log"
	"github.com/eosspark/eos-go/rlp"
)

//var self *Controller

type DBReadMode int8

const (
	SPECULATIVE = DBReadMode(iota)
	HEADER      //HEAD
	READONLY
	IRREVERSIBLE
)

type ValidationMode int8

const (
	FULL = ValidationMode(iota)
	LIGHT
)

type HandlerKey struct {
	handKey map[common.AccountName]common.AccountName
}

type applyCon struct {
	handlerKey   map[common.AccountName]common.AccountName
	applyContext ApplyContext
}

//apply_context
type ApplyHandler struct {
	applyHandler map[common.AccountName]applyCon
	scopeName    common.AccountName
}

type Config struct {
	blocksDir           string
	stateDir            string
	stateSize           uint64
	stateGuardSize      uint64
	reversibleCacheSize uint64
	reversibleGuardSize uint64
	readOnly            bool
	forceAllChecks      bool
	disableReplayOpts   bool
	disableReplay       bool
	contractsConsole    bool
	genesis             types.GenesisState
	//vmType              exec.WasmInterface
	readMode            DBReadMode
	blockValidationMode ValidationMode
	resourceGreylist    []common.AccountName
}

var IsActive bool	//default value false ;Does the process include control ;

var instance	*Controller

type Controller struct {
	db           eosiodb.DataBase
	dbsession    *eosiodb.Session
	reversibledb eosiodb.DataBase
	//reversibleBlocks      *eosiodb.Session
	blog                  string //TODO
	pending               *types.PendingState
	head                  types.BlockState
	forkDB                types.ForkDatabase
	wasmif                exec.WasmInterface
	resourceLimist        ResourceLimitsManager
	authorization         AuthorizationManager
	config                Config //local	Config
	chainID               common.ChainIDType
	rePlaying             bool
	replayHeadTime        common.TimePoint //optional<common.Tstamp>
	readMode              DBReadMode
	inTrxRequiringChecks  bool                //if true, checks that are normally skipped on replay (e.g. auth checks) cannot be skipped
	subjectiveCupLeeway   common.Microseconds //optional<common.Tstamp>
	handlerKey            HandlerKey
	applyHandlers         ApplyHandler
	unappliedTransactions map[rlp.Sha256]types.TransactionMetadata

}

func GetControlInstance() *Controller{
	if !IsActive {
		instance = newController()
	}
	return instance
}

func newController() *Controller {

	db, err := eosiodb.NewDataBase("./", "shared_memory.bin", true)
	if err != nil {
		log.Error("pending NewPendingState is error detail:", err)
		return nil
	}
	defer db.Close()

	session := db.StartSession()

	if err != nil {
		log.Debug("db start session is error detail:", err.Error(), session)
		return nil
	}
	defer session.Undo()

	session.Commit()
	var con = &Controller{inTrxRequiringChecks: false, rePlaying: false}
	IsActive = true
	return con.initConfig()
}

func (self Controller) PopBlock() {

	prev := self.forkDB.GetBlock(self.head.Header.Previous)

	var r types.ReversibleBlockObject
	errs := self.reversibledb.Find("NUM", self.head.BlockNum, r)
	if errs != nil {
		log.Error("PopBlock ReversibleBlocks Find is error,detail:", errs)
	}
	if &r != nil {
		self.reversibledb.Remove(&r)
	}

	if self.readMode == SPECULATIVE {
		var trx []types.TransactionMetadata = self.head.Trxs
		step := 0
		for ; step < len(trx); step++ {
			self.unappliedTransactions[rlp.Sha256(trx[step].SignedID)] = trx[step]
		}
	}
	self.head = prev
	self.dbsession.Undo() //TODO
}

func newApplyCon(ac ApplyContext) *applyCon {
	a := applyCon{}
	a.applyContext = ac
	return &a
}
func (self Controller) SetApplayHandler(receiver common.AccountName, contract common.AccountName, action common.AccountName, handler ApplyContext) {
	h := make(map[common.AccountName]common.AccountName)
	h[receiver] = contract
	apply := newApplyCon(handler)
	apply.handlerKey = h
	t := make(map[common.AccountName]applyCon)
	t[receiver] = *apply
	//TODO common.types make_pair()
	self.applyHandlers = ApplyHandler{t, receiver}
	fmt.Println(self.applyHandlers)
}

func (self Controller) AbortBlock() {
	if self.pending != nil {
		if self.readMode == SPECULATIVE {
			trx := append(self.pending.PendingBlockState.Trxs)
			step := 0
			for ; step < len(trx); step++ {
				self.unappliedTransactions[rlp.Sha256(trx[step].SignedID)] = trx[step]
			}
		}
	}
}

func (self Controller) StartBlock(when common.BlockTimeStamp, confirmBlockCount uint16, s types.BlockStatus) {
	if self.pending != nil {
		fmt.Println("pending block already exists")
		return
	}
	// defer self.peding.reset()
	if self.skipDBSession(s) {
		self.pending = types.NewPendingState(self.db)
	} else {
		self.pending = types.GetInstance()
	}

	self.pending.BlockStatus = s

	self.pending.PendingBlockState = self.head
	self.pending.PendingBlockState.SignedBlock.Timestamp = when
	self.pending.PendingBlockState.InCurrentChain = true
	self.pending.PendingBlockState.SetConfirmed(confirmBlockCount)
	var wasPendingPromoted = self.pending.PendingBlockState.MaybePromotePending()
	log.Info("wasPendingPromoted", wasPendingPromoted)
	if self.readMode == DBReadMode(SPECULATIVE) || self.pending.BlockStatus != types.BlockStatus(types.Incomplete) {
		var gpo = types.GlobalPropertyObject{}
		err := self.db.ByIndex("ID", gpo)
		if err != nil {
			log.Error("Controller StartBlock find GlobalPropertyObject is error :", err)
		}
		//if(gpo.ProposedScheduleBlockNum.valid())
		if (gpo.ProposedScheduleBlockNum <= self.pending.PendingBlockState.DposIrreversibleBlocknum) &&
			(len(self.pending.PendingBlockState.PendingSchedule.Producers) == 0) &&
			(!wasPendingPromoted) {
			if !self.rePlaying {
				tmp := gpo.ProposedSchedule.ProducerScheduleType()
				ps := types.SharedProducerScheduleType{}
				ps.Version = tmp.Version
				ps.Producers = tmp.Producers
				self.pending.PendingBlockState.SetNewProducers(ps)
			}
			self.db.Update(&gpo, func(i interface{}) error {
				gpo.ProposedScheduleBlockNum = 1
				gpo.ProposedSchedule.Clear()
				return nil
			})
		}

		signedTransaction := self.GetOnBlockTransaction()
		onbtrx := types.TransactionMetadata{Trx: signedTransaction}
		onbtrx.Implicit = true
		//TODO defer
		self.inTrxRequiringChecks = true
		//PushTransaction()
		fmt.Println(onbtrx)
	}

}

func (self *Controller) PushTransaction(trx types.TransactionMetadata, deadLine common.TimePoint, billedCpuTimeUs uint32, explicitBilledCpuTime bool) (trxTrace types.TransactionTrace) {
	if deadLine == 0 {
		log.Error("deadline cannot be uninitialized")
		return
	}

	trxContext := TransactionContext{}
	trxContext = *trxContext.NewTransactionContext(self, &trx.Trx, trx.ID, common.Now())

	if self.subjectiveCupLeeway != 0 {
		if self.pending.BlockStatus == types.BlockStatus(types.Incomplete) {
			trxContext.Leeway = self.subjectiveCupLeeway
		}
	}
	trxContext.DeadLine = deadLine
	trxContext.ExplicitBilledCpuTime = explicitBilledCpuTime
	trxContext.BilledCpuTimeUs = int64(billedCpuTimeUs)

	trace := trxContext.Trace
	if trx.Implicit {
		trxContext.InitForImplicitTrx(0) //default value 0
		trxContext.CanSubjectivelyFail = false
	} else {
		/*skipRecording := (self.replayHeadTime !=0) && (common.TimePoint(trx.Trx.Expiration) <= self.replayHeadTime)
		trxContext.InitForInputTrx(uint64(trx.PackedTrx.GetUnprunableSize()),uint64(trx.PackedTrx.GetPrunableSize()), uint32(len(trx.Trx.Signatures)),skipRecording)*/
	}

	fmt.Println(trace)

	return
}

func (self *Controller) GetGlobalProperties() (gp *types.GlobalPropertyObject) {
	ggp := types.GlobalPropertyObject{}
	err := self.db.ByIndex("ID", ggp) //TODO
	if err != nil {
		log.Error("GetGlobalProperties is error detail:", err)
	}
	return
}

func (self *Controller) GetDynamicGlobalProperties() (dgp *types.DynamicGlobalPropertyObject) {
	gpo := types.DynamicGlobalPropertyObject{}
	err := self.db.ByIndex("ID", gpo) //TODO
	if err != nil {
		log.Error("GetGlobalProperties is error detail:", err)
	}
	return
}

func (self *Controller) GetMutableResourceLimitsManager() ResourceLimitsManager {
	return self.resourceLimist
}

func (self *Controller) GetOnBlockTransaction() types.SignedTransaction {
	var onBlockAction = types.Action{}
	onBlockAction.Account = common.AccountName(config.SystemAccountName)
	onBlockAction.Name = common.ActionName(common.StringToName("onblock"))
	onBlockAction.Authorization = []common.PermissionLevel{{common.AccountName(config.SystemAccountName), common.PermissionName(config.ActiveName)}}

	data, err := rlp.EncodeToBytes(self.head.Header)
	if err != nil {
		onBlockAction.Data = data
	}
	var trx = types.SignedTransaction{}
	trx.Actions = append(trx.Actions, &onBlockAction)
	trx.SetReferenceBlock(self.head.ID)
	in := self.pending.PendingBlockState.Header.Timestamp + 999999
	trx.Expiration = common.JSONTime{time.Now().UTC().Add(time.Duration(in))}
	log.Error("getOnBlockTransaction trx.Expiration:", trx)
	return trx
}
func (self *Controller) skipDBSession(bs types.BlockStatus) bool {
	var considerSkipping = (bs == types.BlockStatus(IRREVERSIBLE))
	log.Info("considerSkipping:", considerSkipping)
	return considerSkipping
}

func (self *Controller) SkipDBSessions() bool {
	if self.pending == nil {
		return self.skipDBSession(self.pending.BlockStatus)
	} else {
		return false
	}
}

func (self *Controller) SkipTrxChecks() (b bool) {
	b = self.LightValidationAllowed(self.config.disableReplayOpts)
	return
}

func (self *Controller) LightValidationAllowed(dro bool) (b bool) {
	if self.pending != nil || self.inTrxRequiringChecks {
		return false
	}

	pbStatus := self.pending.BlockStatus
	considerSkippingOnReplay := (pbStatus == types.Irreversible || pbStatus == types.Validated) && !dro

	considerSkippingOnvalidate := (pbStatus == types.Complete && self.config.blockValidationMode == LIGHT)

	return considerSkippingOnReplay || considerSkippingOnvalidate
}

func (self *Controller) isProducingBlock() bool {
	if self.pending == nil {
		return false
	}
	return self.pending.BlockStatus == types.Incomplete
}

func (self *Controller) IsResourceGreylisted(name *common.AccountName) bool {
	for _, account := range self.config.resourceGreylist {
		if &account == name {
			return true
		}
	}
	return false
}

func (self *Controller) PendingBlockState() types.BlockState {
	if self.pending != nil {
		return self.pending.PendingBlockState
	}
	return types.BlockState{}
}

func (self *Controller) PendingBlockTime() common.TimePoint {
	if self.pending == nil {
		log.Error("PendingBlockTime is error", "no pending block")
	}
	return self.pending.PendingBlockState.Header.Timestamp.ToTimePoint()
}

func Close(db eosiodb.DataBase, session eosiodb.Session) {
	//session.close()
	db.Close()
}

func (self *Controller) initConfig() *Controller {
	self.config = Config{
		blocksDir:           config.DefaultBlocksDirName,
		stateDir:            config.DefaultStateDirName,
		stateSize:           config.DefaultStateSize,
		stateGuardSize:      config.DefaultStateGuardSize,
		reversibleCacheSize: config.DefaultReversibleCacheSize,
		reversibleGuardSize: config.DefaultReversibleGuardSize,
		readOnly:            false,
		forceAllChecks:      false,
		disableReplayOpts:   false,
		contractsConsole:    false,
		//vmType:              config.DefaultWasmRuntime, //TODO
		readMode:            SPECULATIVE,
		blockValidationMode: FULL,
	}
	return self

}

func (self *Controller) HeadBlockId() common.BlockIDType{ return common.BlockIDType{}}

func (self *Controller) HeadBlockProducer() common.AccountName{return 0}

func (self *Controller) HeadBlockHeader() *types.BlockHeader{return nil}

func (self *Controller) HeadBlockState() types.BlockState{return types.BlockState{}}

func (self *Controller) ForkDbHeadBlockNum() uint32 { return 0}

func (self *Controller) ForkDbHeadBlockId() common.BlockIDType{ return common.BlockIDType{}}

func (self *Controller) ForkDbHeadBlockTime() common.TimePoint { return 0}

func (self *Controller) ForkDbHeadBlockProducer() common.AccountName{ return 0}

func (self *Controller) ActiveProducers() *types.ProducerScheduleType{ return nil}

func (self *Controller) PendingProducers() *types.ProducerScheduleType{return nil}

func (self *Controller) ProposedProducers() types.ProducerScheduleType{ return types.ProducerScheduleType{}}

func (self *Controller) LastIrreversibleBlockNum() uint32{ return 0}

func (self *Controller) LastIrreversibleBlockId() common.BlockIDType { return common.BlockIDType{}}

func (self *Controller) FetchBlockByNumber(blockNum uint32) types.SignedBlock { return types.SignedBlock{}}

func (self *Controller) FetchBlockById(id common.BlockIDType) types.SignedBlock{ return types.SignedBlock{}}

func (self *Controller) FetchBlockStateByNumber(blockNum uint32) types.BlockState{ return types.BlockState{}}

func (self *Controller) FetchBlockStateById(id common.BlockIDType) types.BlockState{ return types.BlockState{}}

func (self *Controller) GetBlcokIdForNum(blockNum uint32) common.BlockIDType { return common.BlockIDType{}}

func (self *Controller) CheckContractList(code common.AccountName){}

func (self *Controller) CheckActionList(code common.AccountName,action common.ActionName){}

func (self *Controller) CheckKeyList(key *common.PublicKeyType){}

func (self *Controller) IsProducing() bool { return false}

func (self *Controller) IsRamBillingInNotifyAllowed() bool { return false}

func (self *Controller) AddResourceGreyList(name *common.AccountName){}

func (self *Controller) RemoveResourceGreyList(name *common.AccountName){}

func (self *Controller) IsResourceGreyListed(name *common.AccountName) bool{ return false}

func (self *Controller) GetResourceGreyList() *map[common.AccountName]struct{} { return nil}

func (self *Controller) ValidateReferencedAccounts(t types.Transaction){}

func (self *Controller) ValidateExpiration(t types.Transaction){}

func (self *Controller) ValidateTapos(t types.Transaction){}

func (self *Controller) ValidateDbAvailableSize(){}

func (self *Controller) ValidateReversibleAvailableSize(){}

func (self *Controller) IsKnownUnexpiredTransaction(id common.TransactionIDType) bool { return false}

func (self *Controller) SetProposedProducers(producers []types.ProducerKey) int64{ return 0}

func (self *Controller) SkipAuthCheck() bool{ return false}

func (self *Controller) ContractsConsole() bool{ return false}

func (self *Controller) GetChainId() common.ChainIDType { return common.ChainIDType{}}

func (self *Controller) GetReadMode()DBReadMode{ return 0}

func (self *Controller) GetValidationMode()ValidationMode{ return 0}

func (self *Controller) SetSubjectiveCpuLeeway(leeway common.Microseconds){}

func (self *Controller) FindApplyHandler(contract common.AccountName,
	scope common.ScopeName,
	act common.ActionName	) *ApplyContext{
		return nil
}

func (self *Controller) GetWasmInterface() *exec.WasmInterface{return nil}

func (self *Controller) GetAbiSerializer(name common.AccountName,
	maxSerializationTime common.Microseconds) types.AbiSerializer{
	return types.AbiSerializer{}
}

func (self *Controller) ToVariantWithAbi(obj interface{},maxSerializationTime common.Microseconds){}







/*    about chan

signal<void(const signed_block_ptr&)>         pre_accepted_block;
signal<void(const block_state_ptr&)>          accepted_block_header;
signal<void(const block_state_ptr&)>          accepted_block;
signal<void(const block_state_ptr&)>          irreversible_block;
signal<void(const transaction_metadata_ptr&)> accepted_transaction;
signal<void(const transaction_trace_ptr&)>    applied_transaction;
signal<void(const header_confirmation&)>      accepted_confirmation;
signal<void(const int&)>                      bad_alloc;*/

/*func main(){
	c := new(Controller)

	fmt.Println("asdf",c)
}*/