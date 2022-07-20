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
	"github.com/UltronGlow/UltronGlow-Origin/common"
	"github.com/UltronGlow/UltronGlow-Origin/consensus"
	"github.com/UltronGlow/UltronGlow-Origin/consensus/alien/extrastate"
	"github.com/UltronGlow/UltronGlow-Origin/core/types"
	"github.com/UltronGlow/UltronGlow-Origin/ethdb"
	"github.com/UltronGlow/UltronGlow-Origin/log"
	"github.com/UltronGlow/UltronGlow-Origin/rlp"
	"github.com/UltronGlow/UltronGlow-Origin/rpc"
	"github.com/shopspring/decimal"
	"math/big"
)

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the delegated-proof-of-stake scheme.
type API struct {
	chain consensus.ChainHeaderReader
	alien *Alien
}

// GetSnapshot retrieves the state snapshot at a given block.
func (api *API) GetSnapshot(number *rpc.BlockNumber) (*Snapshot, error) {
	// Retrieve the requested block number (or current if none requested)
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	// Ensure we have an actually valid block and return its snapshot
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)

}

// GetSnapshotAtHash retrieves the state snapshot at a given block.
func (api *API) GetSnapshotAtHash(hash common.Hash) (*Snapshot, error) {
	header := api.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
}

// GetSnapshotAtNumber retrieves the state snapshot at a given block.
func (api *API) GetSnapshotAtNumber(number uint64) (*Snapshot, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
}

// GetSnapshotByHeaderTime retrieves the state snapshot by timestamp of header.
// snapshot.header.time <= targetTime < snapshot.header.time + period
// todo: add confirm headertime in return snapshot, to minimize the request from side chain
func (api *API) GetSnapshotByHeaderTime(targetTime uint64, scHash common.Hash) (*Snapshot, error) {
	header := api.chain.CurrentHeader()
	if header == nil {
		return nil, errUnknownBlock
	}
	period := new(big.Int).SetUint64(api.chain.Config().Alien.Period)
	target := new(big.Int).SetUint64(targetTime)
	ceil := new(big.Int).Add(new(big.Int).SetUint64(header.Time), period)
	if target.Cmp(ceil) > 0 {
		target = new(big.Int).SetUint64(header.Time)
	}

	minN := new(big.Int).SetUint64(api.chain.Config().Alien.MaxSignerCount)
	maxN := new(big.Int).Set(header.Number)
	nextN := new(big.Int).SetInt64(0)
	isNext := false
	for {
		ceil = new(big.Int).Add(new(big.Int).SetUint64(header.Time), period)
		if target.Cmp(new(big.Int).SetUint64(header.Time)) >= 0 && target.Cmp(ceil) < 0 {
			snap, err := api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)

			// replace coinbase by signer settings
			var scSigners    []*common.Address
			//for _, signer := range snap.Signers {
			//	replaced := false
			//	if _, ok := snap.SCCoinbase[*signer]; ok {
			//		if addr, ok := snap.SCCoinbase[*signer][scHash]; ok {
			//			replaced = true
			//			scSigners = append(scSigners, &addr)
			//		}
			//	}
			//	if !replaced {
			//		scSigners = append(scSigners, signer)
			//	}
			//}
			for signer, _ := range snap.SCCoinbase[scHash] {
				scSigners = append(scSigners, &signer)
			}
			mcs := Snapshot{
				LoopStartTime: snap.LoopStartTime,
				Period: snap.Period,
				Signers: scSigners,
				Number: snap.Number,
				SCFULBalance: make(map[common.Address]*big.Int),
				SCMinerRevenue: make(map[common.Address]common.Address),
				SCFlowPledge: make(map[common.Address]bool),
			}
			for address, item := range snap.FULBalance {
				balance := new(big.Int).Set(item.Balance)
				for sc, cost := range item.CostTotal {
					if sc.String() == scHash.String() {
						continue
					}
					balance = new(big.Int).Sub(balance,cost)
					if 0 >= balance.Cmp(big.NewInt(0)) {
						break
					}
				}
				mcs.SCFULBalance[address] = balance
			}
			for address, revenue := range snap.RevenueFlow {
				mcs.SCMinerRevenue[address] = revenue.RevenueAddress
			}
			for address, pledge := range snap.FlowPledge {
				if 0 == pledge.StartHigh {
					mcs.SCFlowPledge[address] = true
				}
			}
			if _, ok := snap.SCNoticeMap[scHash]; ok {
				mcs.SCNoticeMap = make(map[common.Hash]*CCNotice)
				mcs.SCNoticeMap[scHash] = snap.SCNoticeMap[scHash]
			}
			return &mcs, err
		} else {
			if minNext := new(big.Int).Add(minN, big.NewInt(1)); maxN.Cmp(minN) == 0 || maxN.Cmp(minNext) == 0 {
				if !isNext && maxN.Cmp(minNext) == 0 {
					var maxHeaderTime, minHeaderTime *big.Int
					maxH := api.chain.GetHeaderByNumber(maxN.Uint64())
					if maxH != nil {
						maxHeaderTime = new(big.Int).SetUint64(maxH.Time)
					} else {
						break
					}
					minH := api.chain.GetHeaderByNumber(minN.Uint64())
					if minH != nil {
						minHeaderTime = new(big.Int).SetUint64(minH.Time)
					} else {
						break
					}
					period = period.Sub(maxHeaderTime, minHeaderTime)
					isNext = true
				} else {
					break
				}
			}
			// calculate next number
			nextN.Sub(target, new(big.Int).SetUint64(header.Time))
			nextN.Div(nextN, period)
			nextN.Add(nextN, header.Number)

			// if nextN beyond the [minN,maxN] then set nextN = (min+max)/2
			if nextN.Cmp(maxN) >= 0 || nextN.Cmp(minN) <= 0 {
				nextN.Add(maxN, minN)
				nextN.Div(nextN, big.NewInt(2))
			}
			// get new header
			header = api.chain.GetHeaderByNumber(nextN.Uint64())
			if header == nil {
				break
			}
			// update maxN & minN
			if new(big.Int).SetUint64(header.Time).Cmp(target) >= 0 {
				if header.Number.Cmp(maxN) < 0 {
					maxN.Set(header.Number)
				}
			} else if new(big.Int).SetUint64(header.Time).Cmp(target) <= 0 {
				if header.Number.Cmp(minN) > 0 {
					minN.Set(header.Number)
				}
			}

		}
	}
	return nil, errUnknownBlock
}

//y add method
func (api *API) GetSnapshotSignerAtNumber(number uint64) (*SnapshotSign, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	snapshot,err:= api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		log.Warn("Fail to GetSnapshotSignAtNumber", "err", err)
		return nil, errUnknownBlock
	}
	snapshotSign := &SnapshotSign{
		LoopStartTime:snapshot.LoopStartTime,
		Signers: snapshot.Signers,
		Punished: snapshot.Punished,
	}
	return snapshotSign, err
}


type SnapshotSign struct {
	LoopStartTime   uint64                                              `json:"loopStartTime"`
	Signers         []*common.Address                                   `json:"signers"`
	Punished        map[common.Address]uint64                           `json:"punished"`
}


func (api *API) GetSnapshotReleaseAtNumber(number uint64,part string) (*SnapshotRelease, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	snapshot,err:= api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		log.Warn("Fail to GetSnapshotSignAtNumber", "err", err)
		return nil, errUnknownBlock
	}
	snapshotRelease := &SnapshotRelease{
		CandidatePledge:make(map[common.Address]*PledgeItem),
		FlowPledge: make(map[common.Address]*PledgeItem),
		FlowRevenue: make(map[common.Address]*LockBalanceData),
	}
	if part!=""{
		if part =="candidatepledge"{
			snapshotRelease.CandidatePledge=snapshot.CandidatePledge
		}else if part =="flowminerpledge"{
			if number < PledgeRevertLockEffectNumber{
				snapshotRelease.FlowPledge=snapshot.FlowPledge
			}
		}else if part =="rewardlock"{
			snapshotRelease.appendFRlockData(snapshot.FlowRevenue.RewardLock,api.alien.db)
		}else if part =="flowlock"{
			snapshotRelease.appendFRlockData(snapshot.FlowRevenue.FlowLock,api.alien.db)
		}else if part =="bandwidthlock"{
			snapshotRelease.appendFRlockData(snapshot.FlowRevenue.BandwidthLock,api.alien.db)
		}else if part =="posplexit"{
			if snapshot.FlowRevenue.PosPgExitLock!=nil {
				snapshotRelease.appendFRlockData(snapshot.FlowRevenue.PosPgExitLock,api.alien.db)
			}

		}
	}else{
		snapshotRelease.CandidatePledge=snapshot.CandidatePledge
		if number < PledgeRevertLockEffectNumber{
			snapshotRelease.FlowPledge=snapshot.FlowPledge
		}
		snapshotRelease.appendFRlockData(snapshot.FlowRevenue.RewardLock,api.alien.db)
		snapshotRelease.appendFRlockData(snapshot.FlowRevenue.FlowLock,api.alien.db)
		snapshotRelease.appendFRlockData(snapshot.FlowRevenue.BandwidthLock,api.alien.db)
		if number >= PledgeRevertLockEffectNumber{
			snapshotRelease.appendFRlockData(snapshot.FlowRevenue.PosPgExitLock,api.alien.db)
		}

	}
	return snapshotRelease, err
}

func (s *SnapshotRelease) appendFRItems(items []*PledgeItem) {
	for _, item := range items {
		if _, ok := s.FlowRevenue[item.TargetAddress]; !ok {
			s.FlowRevenue[item.TargetAddress] = &LockBalanceData{
				RewardBalance:make(map[uint32]*big.Int),
				LockBalance: make(map[uint64]map[uint32]*PledgeItem),
			}
		}
		flowRevenusTarget := s.FlowRevenue[item.TargetAddress]
		if _, ok := flowRevenusTarget.LockBalance[item.StartHigh]; !ok {
			flowRevenusTarget.LockBalance[item.StartHigh] = make(map[uint32]*PledgeItem)
		}
		lockBalance := flowRevenusTarget.LockBalance[item.StartHigh]
		lockBalance[item.PledgeType] = item
	}
}

func (sr *SnapshotRelease) appendFR(FlowRevenue map[common.Address]*LockBalanceData) (error) {
	fr1:=FlowRevenue
	for t1, item1 := range fr1 {
		if _, ok := sr.FlowRevenue[t1]; !ok {
			sr.FlowRevenue[t1] = &LockBalanceData{
				RewardBalance:make(map[uint32]*big.Int),
				LockBalance: make(map[uint64]map[uint32]*PledgeItem),
			}
		}
		rewardBalance:=item1.RewardBalance
		for t2, item2 := range rewardBalance {
			sr.FlowRevenue[t1].RewardBalance[t2]=item2
		}
		lockBalance:=item1.LockBalance
		for t3, item3 := range lockBalance {
			if _, ok := sr.FlowRevenue[t1].LockBalance[t3]; !ok {
				sr.FlowRevenue[t1].LockBalance[t3] = make(map[uint32]*PledgeItem)
			}
			t3LockBalance:=sr.FlowRevenue[t1].LockBalance[t3]
			for t4,item4:=range item3{
				if _, ok := t3LockBalance[t4]; !ok {
					t3LockBalance[t4] = item4
				}
			}
		}
	}
	return nil
}


func (sr *SnapshotRelease) appendFRlockData(lockData *LockData,db ethdb.Database) (error) {
	sr.appendFR(lockData.FlowRevenue)
	items, err := lockData.loadCacheL1(db)
	if err == nil {
		sr.appendFRItems(items)
	}
	items, err = lockData.loadCacheL2(db)
	if err == nil {
		sr.appendFRItems(items)
	}
	return nil
}


type SnapshotRelease struct {
	CandidatePledge map[common.Address]*PledgeItem                      `json:"candidatepledge"`
	FlowPledge      map[common.Address]*PledgeItem                      `json:"flowminerpledge"`
	FlowRevenue     map[common.Address]*LockBalanceData                 `json:"flowrevenve"`
}

func (api *API) GetSnapshotFlowAtNumber(number uint64) (*SnapshotFlow, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	headerExtra := HeaderExtra{}
	err := rlp.DecodeBytes(header.Extra[extraVanity:len(header.Extra)-extraSeal], &headerExtra)
	if err != nil {
		log.Info("Fail to decode header Extra", "err", err)
		return nil,err
	}
	lockReward:=make([]FlowRecord,0)
	if len(headerExtra.LockReward)>0 {
		for _, item := range headerExtra.LockReward {
			if(item.IsReward==sscEnumFlwReward){
				lockReward=append(lockReward,FlowRecord{
					Target: item.Target,
					Amount: item.Amount,
					FlowValue1: item.FlowValue1,
					FlowValue2: item.FlowValue2,
				})
			}
		}
	}
	snapshotFlow := &SnapshotFlow{
		LockReward: lockReward,
	}
	return snapshotFlow, err
}

type SnapshotFlow struct {
	LockReward  []FlowRecord `json:"flowrecords"`
}

type FlowRecord struct {
	Target   common.Address
	Amount   *big.Int
	FlowValue1 uint64 `json:"realFlowvalue"`
	FlowValue2 uint64 `json:"validFlowvalue"`
}

func (api *API) GetSnapshotFlowMinerAtNumber(number uint64) (*SnapshotFlowMiner, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	snapshot,err:= api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		log.Warn("Fail to GetSnapshotFlowMinerAtNumber", "err", err)
		return nil, errUnknownBlock
	}
	flowMiner := &SnapshotFlowMiner{
		DayStartTime:snapshot.FlowMiner.DayStartTime,
		FlowMinerPrevTotal: snapshot.FlowMiner.FlowMinerPrevTotal,
		FlowMiner: snapshot.FlowMiner.FlowMiner,
		FlowMinerPrev:snapshot.FlowMiner.FlowMinerPrev,
		FlowMinerReport:[]*FlowMinerReport{},
		FlowMinerPrevReport:[]*FlowMinerReport{},
	}
	fMiner:=snapshot.FlowMiner
	db:=api.alien.db
	items:=flowMiner.loadFlowMinerCache(fMiner,fMiner.FlowMinerCache,db)
	flowMiner.FlowMinerReport=append(flowMiner.FlowMinerReport,items...)
	items=flowMiner.loadFlowMinerCache(fMiner,fMiner.FlowMinerPrevCache,db)
	flowMiner.FlowMinerPrevReport=append(flowMiner.FlowMinerPrevReport,items...)
	return flowMiner, err
}


type SnapshotFlowMiner struct {
	DayStartTime       uint64                                              `json:"dayStartTime"`
	FlowMinerPrevTotal uint64                                              `json:"flowminerPrevTotal"`
	FlowMiner          map[common.Address]map[common.Hash]*FlowMinerReport `json:"flowminerCurr"`
	FlowMinerReport    []*FlowMinerReport `json:"flowminerReport"`
	FlowMinerPrev      map[common.Address]map[common.Hash]*FlowMinerReport `json:"flowminerPrev"`
	FlowMinerPrevReport    []*FlowMinerReport `json:"flowminerPrevReport"`
}

func (sf *SnapshotFlowMiner) loadFlowMinerCache(fMiner *FlowMinerSnap,flowMinerCache []string,db ethdb.Database) ([]*FlowMinerReport) {
	item:=[]*FlowMinerReport{}
	for _, key := range flowMinerCache {
		flows, err := fMiner.load(db, key)
		if err != nil {
			log.Warn("appendFlowMinerCache load cache error", "key", key, "err", err)
			continue
		}
		item=append(item,flows...)
	}
	return item
}



func (api *API) GetSnapshotFlowReportAtNumber(number uint64) (*SnapshotFlowReport, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	headerExtra := HeaderExtra{}
	err := rlp.DecodeBytes(header.Extra[extraVanity:len(header.Extra)-extraSeal], &headerExtra)
	if err != nil {
		log.Info("Fail to decode header Extra", "err", err)
		return nil,err
	}
	flowReport:=make([]MinerFlowReportRecord,0)
	if len(headerExtra.FlowReport)>0 {
		flowReport=append(flowReport,headerExtra.FlowReport...)
	}
	snapshotFlowReport := &SnapshotFlowReport{
		FlowReport: flowReport,
	}
	return snapshotFlowReport, err
}

type SnapshotFlowReport struct {
	FlowReport []MinerFlowReportRecord `json:"flowreport"`
}


func (api *API) GetLockRewardAtNumber(number uint64) ([]LockRewardRecord, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	headerExtra := HeaderExtra{}
	err := rlp.DecodeBytes(header.Extra[extraVanity:len(header.Extra)-extraSeal], &headerExtra)
	if err != nil {
		log.Info("Fail to decode header Extra", "err", err)
		return nil,err
	}
	LockReward:=make([]LockRewardRecord,0)
	if len(headerExtra.LockReward)>0 {
		LockReward=append(LockReward,headerExtra.LockReward...)
	}
	return LockReward, err
}

func (api *API) GetSRTBalAtNumber(number uint64) (*SnapshotSRT, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	snapshot,err:= api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		log.Warn("Fail to GetSRTBalAtNumber", "err", err)
		return nil, errUnknownBlock
	}

	snapshotSRT:=&SnapshotSRT{
		SrtBal:make(map[common.Address]*big.Int),
	}
	if snapshot.SRT!=nil{
		srtBal:= snapshot.SRT.GetAll()
		if err==nil{
			snapshotSRT.SrtBal=srtBal
		}
	}
	return snapshotSRT, err
}
type SnapshotSRT struct {
	SrtBal map[common.Address]*big.Int `json:"srtbal"`
}

func (api *API) GetSPledgeAtNumber(number uint64) (*SnapshotSPledge, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	snapshot,err:= api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		log.Warn("Fail to GetSPledgeAtNumber", "err", err)
		return nil, errUnknownBlock
	}
	snapshotSPledge := &SnapshotSPledge{
		StoragePledge: make(map[common.Address]*SPledge2),
	}

	for pledgeAddr,sPledge := range snapshot.StorageData.StoragePledge {
		snapshotSPledge.StoragePledge[pledgeAddr]=&SPledge2{
			PledgeStatus:sPledge.PledgeStatus,
			StorageCapacity:sPledge.StorageSpaces.StorageCapacity,
			Lease:make(map[common.Hash]*Lease2),
			LastVerificationTime:sPledge.LastVerificationTime,
			LastVerificationSuccessTime:sPledge.LastVerificationSuccessTime,
			ValidationFailureTotalTime:sPledge.ValidationFailureTotalTime,
		}
		lease:=sPledge.Lease
		for hash,l:=range lease {
			lease2:=&Lease2{
				Address:l.Address,
				Status:l.Status,
				LastVerificationTime:l.LastVerificationTime,
				LastVerificationSuccessTime:l.LastVerificationSuccessTime,
				ValidationFailureTotalTime:l.ValidationFailureTotalTime,
				LeaseList:make(map[common.Hash]*LeaseDetail2),
			}
			ll:=l.LeaseList
			for lhash,item:=range ll{
				lease2.LeaseList[lhash]=&LeaseDetail2{
					Deposit: item.Deposit,
				}
			}
			snapshotSPledge.StoragePledge[pledgeAddr].Lease[hash]=lease2
		}
	}
	return snapshotSPledge, err
}

type SnapshotSPledge struct {
	StoragePledge map[common.Address]*SPledge2 `json:"spledge"`
}

type SPledge2 struct {
	PledgeStatus  *big.Int `json:"pledgeStatus"`
	StorageCapacity *big.Int `json:"storagecapacity"`
	Lease map[common.Hash]*Lease2 `json:"lease"`
	LastVerificationTime  *big.Int `json:"lastverificationtime"`
	LastVerificationSuccessTime  *big.Int `json:"lastverificationsuccesstime"`
	ValidationFailureTotalTime *big.Int `json:"validationfailuretotaltime"`
}
type Lease2 struct {
	Address common.Address `json:"address"`
	Status int `json:"status"`
	LastVerificationTime  *big.Int `json:"lastverificationtime"`
	LastVerificationSuccessTime  *big.Int `json:"lastverificationsuccesstime"`
	ValidationFailureTotalTime *big.Int `json:"validationfailuretotaltime"`
	LeaseList map[common.Hash]*LeaseDetail2 `json:"leaselist"`
}
type LeaseDetail2 struct {
	Deposit                    *big.Int    `json:"deposit"`
}

func (api *API) GetStorageRewardAtNumber(number uint64,part string) (*SnapshotStorageReward, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	snapshot,err:= api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		log.Warn("Fail to GetStoragePledgeRewardAtNumber", "err", err)
		return nil, errUnknownBlock
	}
	snapshotStorageReward := &SnapshotStorageReward{
		StorageReward:StorageReward{
			Reward: make([]SpaceRewardRecord,0),
			LockPeriod:snapshot.SystemConfig.LockParameters[sscEnumRwdLock].LockPeriod,
			RlsPeriod:snapshot.SystemConfig.LockParameters[sscEnumRwdLock].RlsPeriod,
			Interval:snapshot.SystemConfig.LockParameters[sscEnumRwdLock].Interval,
		},
	}
	if part =="spaceLock"||part==""{
		reward,err2:=NewStorageSnap().loadLockReward(api.alien.db,number,storagePledgeRewardkey)
		if err2==nil&&reward!=nil&&len(reward)>0{
			snapshotStorageReward.StorageReward.Reward=append(snapshotStorageReward.StorageReward.Reward,reward...)
		}
	}
	if part =="leaseLock"{
		reward,err2:=NewStorageSnap().loadLockReward(api.alien.db,number,storageLeaseRewardkey)
		if err2==nil&&reward!=nil&&len(reward)>0{
			snapshotStorageReward.StorageReward.Reward=append(snapshotStorageReward.StorageReward.Reward,reward...)
		}
	}
	if part =="revertLock"{
		reward,err2:=NewStorageSnap().loadLockReward(api.alien.db,number,revertSpaceLockRewardkey)
		if err2==nil&&reward!=nil&&len(reward)>0{
			snapshotStorageReward.StorageReward.Reward=append(snapshotStorageReward.StorageReward.Reward,reward...)
		}
	}
	if part =="blockLock"{
		if number >= StorageEffectBlockNumber {
			headerExtra := HeaderExtra{}
			err3 := rlp.DecodeBytes(header.Extra[extraVanity:len(header.Extra)-extraSeal], &headerExtra)
			if err3 != nil {
				log.Info("Fail to decode header Extra", "err", err3)
				return nil,err3
			}
			if len(headerExtra.LockReward)>0 {
				for _,item:=range headerExtra.LockReward{
					if sscEnumSignerReward == item.IsReward {
						revenueAddress:=item.Target
						if revenue, ok := snapshot.RevenueNormal[item.Target]; ok {
							revenueAddress = revenue.RevenueAddress
						}
						spaceRewardRecord:=SpaceRewardRecord{
							Target:item.Target,
							Amount:item.Amount,
							Revenue:revenueAddress,
						}
						snapshotStorageReward.StorageReward.Reward=append(snapshotStorageReward.StorageReward.Reward,spaceRewardRecord)
					}
				}
			}


		reward, err2 := NewStorageSnap().loadLockReward(api.alien.db, number, signerRewardKey)
		if err2 == nil && reward != nil && len(reward) > 0 {
			snapshotStorageReward.StorageReward.Reward = append(snapshotStorageReward.StorageReward.Reward, reward...)
		}
	}
	}
	return snapshotStorageReward, err
}

type SnapshotStorageReward struct {
	StorageReward StorageReward `json:"storagereward"`
}

type StorageReward struct {
	Reward []SpaceRewardRecord `json:"reward"`
	LockPeriod uint32 `json:"LockPeriod"`
	RlsPeriod  uint32 `json:"ReleasePeriod"`
	Interval   uint32 `json:"ReleaseInterval"`
}

func (api *API) GetStorageRatiosAtNumber(number uint64) (*SnapshotStorageRatios, error) {
	snapshotStorageRatios := &SnapshotStorageRatios{
		Ratios:make(map[common.Address]*StorageRatio),
	}
	ratios,err:=NewStorageSnap().lockStorageRatios(api.alien.db,number)
	if err==nil&&ratios!=nil&&len(ratios)>0{
		snapshotStorageRatios.Ratios=ratios
	}
	return snapshotStorageRatios, err
}

type SnapshotStorageRatios struct {
	Ratios map[common.Address]*StorageRatio `json:"ratios"`
}

type SnapshotRevertSRT struct {
	RevertSRT []ExchangeSRTRecord `json:"revertsrt"`
}

func (api *API) GetRevertSRTAtNumber(number uint64) (*SnapshotRevertSRT, error) {
	revertSRT,err:=NewStorageSnap().lockRevertSRT(api.alien.db,number)
	if err != nil {
		log.Info("Fail to decode header Extra", "err", err)
		return nil,err
	}
	snapshotRevertSRT:=&SnapshotRevertSRT{
		RevertSRT:revertSRT,
	}
	return snapshotRevertSRT,nil
}

func (api *API) GetPaysAtNumber(number uint64) (*SnapshotPay) {
	snapshotPay := &SnapshotPay{
		Pays:make([]PayRecard2,0),
	}
	payRecards:=extrastate.LoadPayRecords(number)
	if payRecards!=nil&&len(payRecards)>0{
		for _,pay:=range payRecards{
			snapshotPay.Pays=append(snapshotPay.Pays,PayRecard2{
				Address:pay.Address,
				Amount:pay.Amount,
			})
		}
	}
	return snapshotPay
}

type SnapshotPay struct {
	Pays []PayRecard2 `json:"pays"`
}

type PayRecard2 struct {
	Address common.Address `json:"address"`
	Amount  *big.Int `json:"amount"`
}

type SnapshotAddrSRT struct {
	AddrSrtBal *big.Int `json:"addrsrtbal"`
}

func (api *API) GetSRTBalanceAtNumber(address common.Address,number uint64) (*SnapshotAddrSRT,error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	snapshot,err:= api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		log.Warn("Fail to GetSRTBalanceAtNumber", "err", err)
		return nil, errUnknownBlock
	}

	snapshotAddrSRT:=&SnapshotAddrSRT{
		AddrSrtBal:common.Big0,
	}
	if snapshot.SRT!=nil{
		snapshotAddrSRT.AddrSrtBal= snapshot.SRT.Get(address)
	}

	return snapshotAddrSRT,nil
}

func (api *API) GetSRTBalance(address common.Address) (*SnapshotAddrSRT,error) {
	header := api.chain.CurrentHeader()
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.GetSRTBalanceAtNumber(address,header.Number.Uint64())
}

func (api *API) GetSPledgeInfoByAddr(address common.Address) (*SnapshotSPledgeInfo,error) {
	header := api.chain.CurrentHeader()
	if header == nil {
		return nil, errUnknownBlock
	}
	snapshot,err:= api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		log.Warn("Fail to GetSPledgeInfoByAddr", "err", err)
		return nil, errUnknownBlock
	}
	snapshotSPledgeInfo := &SnapshotSPledgeInfo{
		SPledgeInfo: make(map[common.Address]*SPledge3),
	}
	for pledgeAddr,sPledge := range snapshot.StorageData.StoragePledge {
		if pledgeAddr==address{
			leftCapacity:=snapshot.StorageData.StoragePledge[pledgeAddr].StorageSpaces.StorageCapacity
			snapshotSPledgeInfo.SPledgeInfo[pledgeAddr]=&SPledge3{
				PledgeStatus:sPledge.PledgeStatus,
				TotalCapacity:new(big.Int).Set(sPledge.TotalCapacity),
				LeftCapacity:new(big.Int).Set(leftCapacity),
				Lease:make([]Lease3,0),
			}
			lease:=sPledge.Lease
			for hash,l:=range lease {
				snapshotSPledgeInfo.SPledgeInfo[pledgeAddr].Lease=append(snapshotSPledgeInfo.SPledgeInfo[pledgeAddr].Lease,Lease3{
					Status:l.Status,
					Hash: hash,
				})
			}
		}
	}
	return snapshotSPledgeInfo, err
}

type SnapshotSPledgeInfo struct {
	SPledgeInfo map[common.Address]*SPledge3 `json:"spledgeinfo"`
}

type SPledge3 struct {
	PledgeStatus  *big.Int `json:"pledgeStatus"`
	TotalCapacity *big.Int               `json:"totalcapacity"`
	LeftCapacity  *big.Int               `json:"leftcapacity"`
	Lease []Lease3 `json:"lease"`
}

type Lease3 struct {
	Hash common.Hash `json:"hash"`
	Status int `json:"status"`
}

func (api *API) GetSPledgeCapVerAtNumber(number uint64) (*SnapshotSPledgeCapVer, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	snapshot,err:= api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		log.Warn("Fail to GetSPledgeCapVerAtNumber", "err", err)
		return nil, errUnknownBlock
	}
	snapshotSPledgeCapVer := &SnapshotSPledgeCapVer{
		SpledgeCapVer: api.calStorageVerifyPercentage(number,snapshot.getBlockPreDay(),snapshot.StorageData.copy()),
	}
    return snapshotSPledgeCapVer, err
}

type SnapshotSPledgeCapVer struct {
	SpledgeCapVer map[common.Address]*big.Int `json:"spledgecapver"`
}

func (api *API) calStorageVerifyPercentage(number uint64, blockPerday uint64,s *StorageData) (map[common.Address]*big.Int) {
	capSuccPer := make(map[common.Address]*big.Int, 0)
	bigNumber := new(big.Int).SetUint64(number)
	bigblockPerDay := new(big.Int).SetUint64(blockPerday)
	zeroTime := new(big.Int).Mul(new(big.Int).Div(bigNumber, bigblockPerDay), bigblockPerDay) //0:00 every day
	beforeZeroTime := new(big.Int).Set(zeroTime)
	for pledgeAddr, sPledge := range s.StoragePledge {
		capSucc := big.NewInt(0)
		storagespaces := s.StoragePledge[pledgeAddr].StorageSpaces
		sfiles := storagespaces.StorageFile
		for _, sfile := range sfiles {
			lastVerSuccTime := sfile.LastVerificationSuccessTime
			if lastVerSuccTime.Cmp(beforeZeroTime) < 0 {

			} else {
				capSucc = new(big.Int).Add(capSucc, sfile.Capacity)
			}
		}
		leases := make(map[common.Hash]*Lease)
		for lhash, l := range sPledge.Lease {
			if l.Status == LeaseNormal || l.Status == LeaseBreach {
				leases[lhash] = l
			}
		}
		for _, lease := range leases {
			storageFile := lease.StorageFile
			for _, file := range storageFile {
				lastVerSuccTime := file.LastVerificationSuccessTime
				if lastVerSuccTime.Cmp(beforeZeroTime) < 0 {

				} else {
					capSucc = new(big.Int).Add(capSucc, file.Capacity)
				}
			}
		}
		per := new(big.Int).Mul(capSucc, big.NewInt(100))
		per = new(big.Int).Div(per,  sPledge.TotalCapacity)
		capSuccPer[pledgeAddr]=per
	}
	return capSuccPer
}

type SnapshotSPledgeValue struct {
	SpledgeValue *big.Int `json:"spledgevalue"`
}

func (api *API) GetStorageValueAtNumber(number uint64,part string) (*SnapshotSPledgeValue, error) {
	snapshotStorage := &SnapshotSPledgeValue{
		SpledgeValue:common.Big0,
	}
	key:=originalTotalCapacityKey
	var err error
	var v *big.Int
	if part =="totalPledgeReward"{
		key=totalPledgeRewardKey
	}
	if part =="storageHarvest"{
		key=storageHarvestKey
	}
	if part =="leaseHarvest"{
		key=leaseHarvestKey
	}
	v,err=NewStorageSnap().loadSpledgeValue(api.alien.db,number,key)
	if err==nil&&v!=nil{
		snapshotStorage.SpledgeValue=v
	}
	return snapshotStorage, err
}

type SnapshotSPledgeDecimalValue struct {
	SpledgeDecimalValue decimal.Decimal `json:"spledgedecimalvalue"`
}


func (api *API) GetStorageDecimalValueAtNumber(number uint64,part string) (*SnapshotSPledgeDecimalValue, error) {
	snapshotStorage := &SnapshotSPledgeDecimalValue{
		SpledgeDecimalValue:decimal.Zero,
	}
	key:=totalLeaseSpaceKey
	var err error
	var v decimal.Decimal
	v,err=NewStorageSnap().loadSpledgeDecimalValue(api.alien.db,number,key)
	if err==nil{
		snapshotStorage.SpledgeDecimalValue=v
	}
	return snapshotStorage, err
}


type SnapshotSPledgeRatioValue struct {
	SpledgeRatioValue decimal.Decimal `json:"spledgeratiovalue"`
}

func (api *API) GetStorageRatioValueAtNumber(number uint64,value *big.Int,part string) (*SnapshotSPledgeRatioValue, error) {
	snapshotStorage := &SnapshotSPledgeRatioValue{
		SpledgeRatioValue:decimal.Zero,
	}
	var v decimal.Decimal
	if part =="Bandwidth"{
		v=getBandwaith(value,number)
	}
	if part =="StorageRatio"{
		v=NewStorageSnap().calStorageRatio(value)
	}
	snapshotStorage.SpledgeRatioValue=v
	return snapshotStorage, nil
}

type SnapshotSucSPledge struct {
	SucSPledge []common.Address `json:"sucspledge"`
}

func (api *API) GetSucSPledgeAtNumber(number uint64) (*SnapshotSucSPledge, error) {
	snapshotSucSPledge := &SnapshotSucSPledge{
		SucSPledge:make([]common.Address,0),
	}
	sucspledge,err:=NewStorageSnap().loadSPledgeSucc(api.alien.db,number)
	if err==nil&&sucspledge!=nil&&len(sucspledge)>0{
		snapshotSucSPledge.SucSPledge=sucspledge
	}
	return snapshotSucSPledge, err
}

type SnapshotRentSuc struct {
	RentSuc []common.Hash `json:"rentsuc"`
}

func (api *API) GetRentSucAtNumber(number uint64) (*SnapshotRentSuc, error) {
	snapshotRentSuc := &SnapshotRentSuc{
		RentSuc:make([]common.Hash,0),
	}
	rentSuc,err:=NewStorageSnap().loadRentSucc(api.alien.db,number)
	if err==nil&&rentSuc!=nil&&len(rentSuc)>0{
		snapshotRentSuc.RentSuc=rentSuc
	}
	return snapshotRentSuc, err
}


type SnapshotCapSuccAddrs struct {
	CapSuccAddrs map[common.Address]*big.Int `json:"capsuccaddrs"`
}

func (api *API) GetCapSuccAddrsAtNumber(number uint64) (*SnapshotCapSuccAddrs, error) {
	snapshotCapSuccAddrs := &SnapshotCapSuccAddrs{
		CapSuccAddrs:make(map[common.Address]*big.Int),
	}
	capSuccAddrs,err:=NewStorageSnap().loadCapSuccAddrs(api.alien.db,number)
	if err==nil&&capSuccAddrs!=nil&&len(capSuccAddrs)>0{
		snapshotCapSuccAddrs.CapSuccAddrs=capSuccAddrs
	}
	return snapshotCapSuccAddrs, err
}

func (api *API) GetGrantProfitAtNumber(number uint64) ([]consensus.GrantProfitRecord, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	headerExtra := HeaderExtra{}
	err := rlp.DecodeBytes(header.Extra[extraVanity:len(header.Extra)-extraSeal], &headerExtra)
	if err != nil {
		log.Info("Fail to decode header Extra", "err", err)
		return nil,err
	}
	grantProfit:=make([]consensus.GrantProfitRecord,0)
	if len(headerExtra.GrantProfit)>0 {
		grantProfit=append(grantProfit,headerExtra.GrantProfit...)
	}
	return grantProfit, err
}

type SnapshotSTGbwMakeup struct {
	STGBandwidthMakeup map[common.Address]*BandwidthMakeup `json:"stgbandwidthmakeup"`
}

func (api *API) GetSTGBandwidthMakeup() (*SnapshotSTGbwMakeup, error) {
	header := api.chain.GetHeaderByNumber(StoragePledgeOptEffectNumber)
	if header == nil {
		return nil, errUnknownBlock
	}
	snapshot,err:= api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		log.Warn("Fail to GetSPledgeCapVerAtNumber", "err", err)
		return nil, errUnknownBlock
	}
	snapshotSTGbwMakeup := &SnapshotSTGbwMakeup{
		STGBandwidthMakeup: snapshot.STGBandwidthMakeup,
	}
	return snapshotSTGbwMakeup, err
}