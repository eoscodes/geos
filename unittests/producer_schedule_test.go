package unittests

import (
	"fmt"
	. "github.com/eosspark/eos-go/chain"
	"github.com/eosspark/eos-go/chain/types"
	"github.com/eosspark/eos-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Calculate expected producer given the schedule and slot number
func getExpectedProducer(schedule []types.ProducerKey, slot uint64) common.AccountName {
	index := int(slot) % (len(schedule) * common.DefaultConfig.ProducerRepetitions) / common.DefaultConfig.ProducerRepetitions
	return schedule[index].ProducerName
}

// Check if two schedule is equal
func isScheduleEqual(first []types.ProducerKey, second []types.ProducerKey) bool {
	isEqual := len(first) == len(second)
	for i := 0; i < len(first); i++ {
		isEqual = isEqual && first[i] == second[i]
	}
	return isEqual
}

// Calculate the block num of the next round first block
// The new producer schedule will become effective when it's in the block of the next round first block
// However, it won't be applied until the effective block num is deemed irreversible
func calcBloclkNumOfNextRoundFirstBlock(control *Controller) uint64 {
	res := control.HeadBlockNum() + 1
	blocksPerRound := len(control.HeadBlockState().ActiveSchedule.Producers) * common.DefaultConfig.ProducerRepetitions
	for res % uint32(blocksPerRound) != 0 {
		res ++
	}
	return uint64(res)
}

func TestVerifyProducerSchedule(t *testing.T) {
	b := newBaseTester(true, SPECULATIVE)
	confirmScheduleCorrectness := func(newProdSchd []types.ProducerKey, effNewProdSchdBlockNum uint64) {
		checkDuration := uint32(1000)
		for i := uint32(0); i < checkDuration; i++ {
			currentSchedule := b.Control.HeadBlockState().ActiveSchedule.Producers
			currentAbsoluteSlot := b.Control.GetGlobalProperties().ProposedScheduleBlockNum
			expectedProducer := getExpectedProducer(currentSchedule, uint64(currentAbsoluteSlot + 1))
			isNewScheduleApplied := uint64(b.Control.LastIrreversibleBlockNum()) > effNewProdSchdBlockNum

			if isNewScheduleApplied {
				assert.True(t, isScheduleEqual(newProdSchd, currentSchedule))
			} else {
				assert.False(t, isScheduleEqual(newProdSchd, currentSchedule))
			}

			b.ProduceBlock(common.Milliseconds(common.DefaultConfig.BlockIntervalMs), 0)
			assert.True(t, b.Control.HeadBlockProducer() == expectedProducer)
		}
	}

	producers := []common.AccountName{
		common.N("inita"), common.N("initb"), common.N("initc"), common.N("initd"), common.N("inite"), common.N("initf"), common.N("initg"),
		common.N("inith"), common.N("initi"), common.N("initj"), common.N("initk"), common.N("initl"), common.N("initm"), common.N("initn"),
		common.N("inito"), common.N("initp"), common.N("initq"), common.N("initr"), common.N("inits"), common.N("initt"), common.N("initu"),
	}
	b.CreateAccounts(producers, false, true)

	// ---- Test first set of producers ----
	// Send set prods action and confirm schedule correctness
	fmt.Println("------------")
	b.SetProducers(&producers)
	fmt.Println("------------")

	firstProdSchd := b.GetProducerKeys(&producers)
	effFirstProdSchdBlockNum := calcBloclkNumOfNextRoundFirstBlock(b.Control)
	confirmScheduleCorrectness(firstProdSchd, effFirstProdSchdBlockNum)

	// ---- Test second set of producers ----
	secondSetOfProducers := []common.AccountName{
		producers[3], producers[6], producers[9], producers[12], producers[15], producers[18], producers[20],
	}
	b.SetProducers(&secondSetOfProducers)
	secondProdSchd := b.GetProducerKeys(&secondSetOfProducers)
	effSecondProdSchdBlockNum := calcBloclkNumOfNextRoundFirstBlock(b.Control)
	confirmScheduleCorrectness(secondProdSchd, effSecondProdSchdBlockNum)

	// ---- Test deliberately miss some blocks ----
	numOfMissedBlocks := int64(5000)
	b.ProduceBlock(common.Microseconds(500 * 1000 * numOfMissedBlocks), 0)
	confirmScheduleCorrectness(secondProdSchd, effSecondProdSchdBlockNum)
	b.ProduceBlock(common.Milliseconds(common.DefaultConfig.BlockIntervalMs), 0)

	// ---- Test third set of producers ----
	thirdSetOfProducer := []common.AccountName{
		producers[2], producers[5], producers[8], producers[11], producers[14], producers[17], producers[20],
		producers[0], producers[3], producers[6], producers[9], producers[12], producers[15], producers[18],
		producers[1], producers[4], producers[7], producers[10], producers[13], producers[16], producers[19],
	}
	b.SetProducers(&thirdSetOfProducer)
	thirdProSchd := b.GetProducerKeys(&thirdSetOfProducer)
	effThirdProdSchdBlockNum := calcBloclkNumOfNextRoundFirstBlock(b.Control)
	confirmScheduleCorrectness(thirdProSchd, effThirdProdSchdBlockNum)
	b.close()
}
