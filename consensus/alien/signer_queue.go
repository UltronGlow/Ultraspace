// Copyright 2021 The utg Authors
// This file is part of the utg library.
//
// The utg library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The utg library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the utg library. If not, see <http://www.gnu.org/licenses/>.

// Package alien implements the delegated-proof-of-stake consensus engine.

package alien

import (
	"bytes"
	"github.com/UltronGlow/UltronGlow-Origin/log"
	"github.com/shopspring/decimal"
	"math"
	"math/big"
	"sort"

	"github.com/UltronGlow/UltronGlow-Origin/common"
)

type TallyItem struct {
	addr  common.Address
	stake *big.Int
}
type TallySlice []TallyItem

func (s TallySlice) Len() int      { return len(s) }
func (s TallySlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s TallySlice) Less(i, j int) bool {
	//we need sort reverse, so ...
	isLess := s[i].stake.Cmp(s[j].stake)
	if isLess > 0 {
		return true

	} else if isLess < 0 {
		return false
	}
	// if the stake equal
	return bytes.Compare(s[i].addr.Bytes(), s[j].addr.Bytes()) > 0
}

type SignerItem struct {
	addr common.Address
	hash common.Hash
}
type SignerSlice []SignerItem

func (s SignerSlice) Len() int      { return len(s) }
func (s SignerSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SignerSlice) Less(i, j int) bool {
	isLess :=  bytes.Compare(s[i].hash.Bytes(), s[j].hash.Bytes())
	if isLess > 0 {
		return true
	} else if isLess < 0 {
		return false
	}
	// if the hash equal
	return bytes.Compare(s[i].addr.Bytes(), s[j].addr.Bytes()) > 0
}

// verify the SignerQueue base on block hash
func (s *Snapshot) verifySignerQueue(signerQueue []common.Address) error {

	if len(signerQueue) > int(s.config.MaxSignerCount) {
		return errInvalidSignerQueue
	}
	sq, err := s.createSignerQueue()
	if err != nil {
		return err
	}
	if len(sq) == 0 || len(sq) != len(signerQueue) {
		return errInvalidSignerQueue
	}
	for i, signer := range signerQueue {
		if signer != sq[i] {
			log.Info("verifySignerQueue","sq",sq,"signerQueue",signerQueue)
			return errInvalidSignerQueue
		}
	}

	return nil
}

func (s *Snapshot) buildTallySlice() TallySlice {
	var tallySlice TallySlice
	if s.Number+1>= SigerElectNewEffectBlockNumber{
		for address, stake := range s.Tally {
			if !candidateNeedPD || s.isCandidate(address) {
				tallySlice = append(tallySlice, TallyItem{address, stake})
			}
		}
		return tallySlice
	}
	for address, stake := range s.Tally {
		if !candidateNeedPD || s.isCandidate(address) {
			if _, ok := s.Punished[address]; ok {
				var creditWeight uint64
				if s.Punished[address] > defaultFullCredit-minCalSignerQueueCredit {
					creditWeight = minCalSignerQueueCredit
				} else {
					creditWeight = defaultFullCredit - s.Punished[address]
				}
				tallySlice = append(tallySlice, TallyItem{address, new(big.Int).Mul(stake, big.NewInt(int64(creditWeight)))})
			} else {
				tallySlice = append(tallySlice, TallyItem{address, new(big.Int).Mul(stake, big.NewInt(defaultFullCredit))})
			}
		}
	}
	return tallySlice
}

func (s *Snapshot) buildTallyMiner() TallySlice {
	var tallySlice TallySlice
	if s.Number+1>= SigerElectNewEffectBlockNumber{
		for address, stake := range s.TallyMiner {
			if pledge, ok := s.CandidatePledge[address]; !ok || 0 < pledge.StartHigh || s.Punished[address] >= minCalSignerQueueCredit {
				continue
			}
			if s.Number+1> TallyPunishdFixBlockNumber {
				if _, isok := s.Tally[address]; isok{
					continue
				}
			}
			tallySlice = append(tallySlice, TallyItem{address, stake.Stake})
		}
		return tallySlice
	}
	for address, stake := range s.TallyMiner {
		if pledge, ok := s.CandidatePledge[address]; !ok || 0 < pledge.StartHigh || s.Punished[address] >= minCalSignerQueueCredit {
			continue
		}
		if _, ok := s.Punished[address]; ok {
			var creditWeight uint64
			if s.Punished[address] > defaultFullCredit-minCalSignerQueueCredit {
				creditWeight = minCalSignerQueueCredit
			} else {
				creditWeight = defaultFullCredit - s.Punished[address]
			}
			tallySlice = append(tallySlice, TallyItem{address, new(big.Int).Mul(stake.Stake, big.NewInt(int64(creditWeight)))})
		} else {
			tallySlice = append(tallySlice, TallyItem{address, new(big.Int).Mul(stake.Stake, big.NewInt(defaultFullCredit))})
		}
	}
	return tallySlice
}

func (s *Snapshot) rebuildTallyMiner(miners TallySlice) TallySlice {
	if s.Number+1>= SigerElectNewEffectBlockNumber{
		return s.reBuildMiner(miners)
	}
	var tallySlice TallySlice
	for _, item := range miners {
		if status, ok := s.TallyMiner[item.addr]; ok {
			tallySlice = append(tallySlice, TallyItem{item.addr, big.NewInt(int64(status.SignerNumber + 1))})
		}
	}
	sort.Sort(tallySlice)
	if (s.Number+1) >= SigerQueueFixBlockNumber {
		tallySlice=tsReverse(tallySlice)
	}
	return tallySlice
}
func  tsReverse(s TallySlice) TallySlice {
	var reverseSlice   TallySlice
	minnerlen :=len(s)
	for i, _ := range s {
		reverseSlice =append(reverseSlice,s[minnerlen-1-i])
	}
	return reverseSlice
}
func  (s *Snapshot)  reBuildMiner(miners TallySlice) TallySlice{
	totalAmount :=big.NewInt(0)
	for _, item := range miners {
		if _, ok := s.TallyMiner[item.addr]; ok {
			totalAmount= new(big.Int).Add(totalAmount,item.stake)
		}
	}
	var tallySlice TallySlice
	for _, item := range miners {
		signerNumber :=uint64(0)
		if status, ok := s.TallyMiner[item.addr]; ok {
			signerNumber = status.SignerNumber
		}
		selectParam:=s.calculateMinerState(item,totalAmount,signerNumber)
		tallySlice = append(tallySlice, TallyItem{item.addr,selectParam.BigInt() })
	}
	sort.Sort(tallySlice)
	return tallySlice
}
func  (s *Snapshot)  reBuildMainMiner(miners TallySlice) TallySlice{
	totalAmount :=big.NewInt(0)
	for _, item := range miners {
		totalAmount= new(big.Int).Add(totalAmount,item.stake)
	}
	var tallySlice TallySlice
	for _, item := range miners {
		signerNumber :=uint64(0)
		if  count,ok := s.TallySigner[item.addr];ok {
			signerNumber =count
		}
		selectParam:=s.calculateMinerState(item,totalAmount,signerNumber)
		tallySlice = append(tallySlice, TallyItem{item.addr,selectParam.BigInt() })

	}
	sort.Sort(tallySlice)
	return tallySlice
}
func (s *Snapshot) calculateMinerState(item TallyItem,totalAmount *big.Int,signerNumber uint64) decimal.Decimal{

	sigerIndex :=  float64(1) /float64(signerNumber+1)
	assetsRate :=decimal.NewFromBigInt(item.stake,0).Div(decimal.NewFromBigInt(totalAmount,0))
	creditWeight := uint64(defaultFullCredit)
	if _, ok := s.Punished[item.addr]; ok {
		creditWeight =defaultFullCredit - s.Punished[item.addr]
	}
	assetsRate=assetsRate.Mul(decimal.NewFromFloat(float64(creditWeight)))
	return decimal.NewFromFloat(math.Sqrt(sigerIndex )) .Mul(assetsRate).Mul(decimal.NewFromFloat(1e+18))
}
func (s *Snapshot) createSignerQueue() ([]common.Address, error) {

	if (s.Number+1)%s.config.MaxSignerCount != 0 || s.Hash != s.HistoryHash[len(s.HistoryHash)-1] {
		return nil, errCreateSignerQueueNotAllowed
	}

	var signerSlice SignerSlice
	var topStakeAddress []common.Address

	if (s.Number+1)%(s.config.MaxSignerCount*s.LCRS) == 0 {

		// before recalculate the signers, clear the candidate is not in snap.Candidates
		//log.Info("begin select node","blocknumbrt",s.Number)
		// only recalculate signers from to tally per 10 loop,
		// other loop end just reset the order of signers by block hash (nearly random)
		mainMinerSlice := s.buildTallySlice()
		sort.Sort(TallySlice(mainMinerSlice))
		secondMinerSlice := s.buildTallyMiner()
		sort.Sort(TallySlice(secondMinerSlice))
		queueLength := int(s.config.MaxSignerCount)
		mainSignerSliceLen := len(mainMinerSlice)

		if queueLength >= defaultOfficialMaxSignerCount {
			mainMinerNumber := (9*queueLength + defaultOfficialMaxSignerCount - 1) / defaultOfficialMaxSignerCount
			secondMinerNumber := 12 * queueLength / defaultOfficialMaxSignerCount

			if secondMinerNumber >= len(secondMinerSlice) {
				secondMinerNumber = len(secondMinerSlice)
				mainMinerNumber = queueLength - secondMinerNumber
				signerSlice = s.selectSecondMinerInsufficient(secondMinerSlice, signerSlice)
			} else {
				mainMinerNumber = queueLength - secondMinerNumber
				var candidatePledgeSlice TallySlice
				if s.Number+1>= SigerElectNewEffectBlockNumber{
					candidatePledgeSlice = secondMinerSlice
				}else{
					if len(secondMinerSlice)+mainSignerSliceLen >= maxCandidateMiner {
						for _, tallyItem := range secondMinerSlice[:maxCandidateMiner-mainSignerSliceLen] {
							candidatePledgeSlice = append(candidatePledgeSlice, TallyItem{tallyItem.addr, tallyItem.stake})
						}
					} else {
						candidatePledgeSlice = secondMinerSlice
					}
				}

				signerSlice = s.selectSecondMiner(candidatePledgeSlice, secondMinerNumber, signerSlice, queueLength)
			}
			// select Main Miner
			signerSlice = s.selectMainMiner(mainMinerNumber, mainSignerSliceLen, signerSlice, mainMinerSlice, secondMinerNumber)
		} else {
			if queueLength > len(mainMinerSlice) {
				queueLength = len(mainMinerSlice)
			}
			for i, tallyItem := range mainMinerSlice[:queueLength] {
				signerSlice = append(signerSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i]})
			}
		}
	} else {
		for i, signer := range s.Signers {
			signerSlice = append(signerSlice, SignerItem{*signer, s.HistoryHash[len(s.HistoryHash)-1-i]})
		}
	}
	// Set the top candidates in random order base on block hash
	sort.Sort(SignerSlice(signerSlice))
	if len(signerSlice) == 0 {
		return nil, errSignerQueueEmpty
	}
	for i := 0; i < int(s.config.MaxSignerCount); i++ {
		topStakeAddress = append(topStakeAddress, signerSlice[i%len(signerSlice)].addr)
	}
	return topStakeAddress, nil
}

func (s *Snapshot) selectMainMiner(mainMinerNumber int, mainSignerSliceLen int, signerSlice SignerSlice, mainMinerSlice TallySlice, secondMinerNumber int) SignerSlice {
	if s.Number+1>SigerElectNewEffectBlockNumber {
		mainMinerSlice=s.reBuildMainMiner(mainMinerSlice)
	}
	if mainMinerNumber > mainSignerSliceLen {
		//mainSignerSliceLen := len(mainMinerSlice)
		for i := 0; i < mainMinerNumber; i++ {
			signerSlice = append(signerSlice, SignerItem{mainMinerSlice[i%mainSignerSliceLen].addr, s.HistoryHash[len(s.HistoryHash)-1-i-secondMinerNumber]})
		}
	} else {
		for i := 0; i < mainMinerNumber; i++ {
			signerSlice = append(signerSlice, SignerItem{mainMinerSlice[i].addr, s.HistoryHash[len(s.HistoryHash)-1-i-secondMinerNumber]})
		}
	}
	return signerSlice
}

func (s *Snapshot) selectSecondMiner(candidatePledgeSlice TallySlice, secondMinerNumber int, signerSlice SignerSlice, queueLength int) SignerSlice {
	candidateLen := len(candidatePledgeSlice)
	if s.Number+1>= SigerElectNewEffectBlockNumber{
		return s.selectNewSecondMiner(candidatePledgeSlice,secondMinerNumber,signerSlice)
	}
	if candidateLen <= electionPartitionThreshold {
		candidatePledgeSlice = s.rebuildTallyMiner(candidatePledgeSlice)
		for i, tallyItem := range candidatePledgeSlice[:secondMinerNumber] {
			signerSlice = append(signerSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i]})
		}
	} else {
		var LevelSlice TallySlice
		index := int(0)
		firstNumber := 6 * queueLength / defaultOfficialMaxSignerCount
		//Proportion of the second step  20%
		firstTotal := candidateLen * 2 / 10
		for _, tallyItem := range candidatePledgeSlice[:firstTotal] {
			LevelSlice = append(LevelSlice, TallyItem{tallyItem.addr, tallyItem.stake})
		}
		LevelSlice = s.rebuildTallyMiner(LevelSlice)
		for i, tallyItem := range LevelSlice[:firstNumber] {
			signerSlice = append(signerSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i-index]})
		}
		index += firstNumber
		secondNumber := 4 * queueLength / 21
		//Proportion of the third step 30%  =secondTotal-firstTotal
		secondTotal := candidateLen * 3 / 10
		var secondLevelSlice TallySlice
		for _, tallyItem := range candidatePledgeSlice[firstTotal : firstTotal+secondTotal] {
			secondLevelSlice = append(secondLevelSlice, TallyItem{tallyItem.addr, tallyItem.stake})
		}
		secondLevelSlice = s.rebuildTallyMiner(secondLevelSlice)
		for i, tallyItem := range secondLevelSlice[:secondNumber] {
			signerSlice = append(signerSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i-index]})
		}
		index += secondNumber
		var lastLevelSlice TallySlice
		lastNumber := secondMinerNumber - index
		//Proportion of the fourth step 50%
		for _, tallyItem := range candidatePledgeSlice[firstTotal+secondTotal:] {
			lastLevelSlice = append(lastLevelSlice, TallyItem{tallyItem.addr, tallyItem.stake})
		}
		lastLevelSlice = s.rebuildTallyMiner(lastLevelSlice)
		for i, tallyItem := range lastLevelSlice[:lastNumber] {
			signerSlice = append(signerSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i-index]})
		}
	}
	return signerSlice
}

func (s *Snapshot) selectSecondMinerInsufficient(tallyMiner TallySlice, signerSlice SignerSlice) SignerSlice {
	for i, tallyItem := range tallyMiner {
		signerSlice = append(signerSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i]})
	}
	return signerSlice
}
func (s *Snapshot) selectNewSecondMiner(candidatePledgeSlice TallySlice, secondMinerNumber int, signerSlice SignerSlice) SignerSlice {
	candidatePledgeSlice = s.rebuildTallyMiner(candidatePledgeSlice)
	for i, tallyItem := range candidatePledgeSlice[:secondMinerNumber] {
		signerSlice = append(signerSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i]})
	}
	return signerSlice
}
