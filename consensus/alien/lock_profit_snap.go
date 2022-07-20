package alien

import (
	"bytes"
	"github.com/UltronGlow/UltronGlow-Origin/common"
	"github.com/UltronGlow/UltronGlow-Origin/consensus"
	"github.com/UltronGlow/UltronGlow-Origin/core/state"
	"github.com/UltronGlow/UltronGlow-Origin/core/types"
	"github.com/UltronGlow/UltronGlow-Origin/ethdb"
	"github.com/UltronGlow/UltronGlow-Origin/log"
	"github.com/UltronGlow/UltronGlow-Origin/rlp"
	"github.com/shopspring/decimal"
	"math/big"
	"time"
)

const (
	LOCKREWARDDATA    = "reward"
	LOCKFLOWDATA      = "flow"
	LOCKBANDWIDTHDATA = "bandwidth"
	LOCKPOSEXITDATA = "posplexit"
)

type RlsLockData struct {
	LockBalance map[uint64]map[uint32]*PledgeItem // The primary key is lock number, The second key is pledge type
}

type LockData struct {
	FlowRevenue map[common.Address]*LockBalanceData `json:"flowrevenve"`
	CacheL1     []common.Hash                       `json:"cachel1"` // Store chceckout data
	CacheL2     common.Hash                         `json:"cachel2"` //Store data of the previous day
	//rlsLockBalance map[common.Address]*RlsLockData     // The release lock data
	Locktype string `json:"Locktype"`
}

func NewLockData(t string) *LockData {
	return &LockData{
		FlowRevenue: make(map[common.Address]*LockBalanceData),
		CacheL1:     []common.Hash{},
		CacheL2:     common.Hash{},
		Locktype:    t,
	}
}

func (l *LockData) copy() *LockData {
	clone := &LockData{
		FlowRevenue: make(map[common.Address]*LockBalanceData),
		CacheL1:     []common.Hash{},
		CacheL2:     l.CacheL2,
		//rlsLockBalance: nil,
		Locktype: l.Locktype,
	}
	clone.CacheL1 = make([]common.Hash, len(l.CacheL1))
	copy(clone.CacheL1, l.CacheL1)
	for who, pledges := range l.FlowRevenue {
		clone.FlowRevenue[who] = &LockBalanceData{
			RewardBalance: make(map[uint32]*big.Int),
			LockBalance:   make(map[uint64]map[uint32]*PledgeItem),
		}
		for which, balance := range l.FlowRevenue[who].RewardBalance {
			clone.FlowRevenue[who].RewardBalance[which] = new(big.Int).Set(balance)
		}
		for when, pledge1 := range pledges.LockBalance {
			clone.FlowRevenue[who].LockBalance[when] = make(map[uint32]*PledgeItem)
			for which, pledge := range pledge1 {
				clone.FlowRevenue[who].LockBalance[when][which] = pledge.copy()
			}
		}
	}
	return clone
}

func (s *LockData) addLockData(snap *Snapshot, item LockRewardRecord, headerNumber *big.Int) {
	if _, ok := s.FlowRevenue[item.Target]; !ok {
		s.FlowRevenue[item.Target] = &LockBalanceData{
			RewardBalance: make(map[uint32]*big.Int),
			LockBalance:   make(map[uint64]map[uint32]*PledgeItem),
		}
	}
	flowRevenusTarget := s.FlowRevenue[item.Target]
	if _, ok := flowRevenusTarget.RewardBalance[item.IsReward]; !ok {
		flowRevenusTarget.RewardBalance[item.IsReward] = new(big.Int).Set(item.Amount)
	} else {
		flowRevenusTarget.RewardBalance[item.IsReward] = new(big.Int).Add(flowRevenusTarget.RewardBalance[item.IsReward], item.Amount)
	}
}

func (s *LockData) updateAllLockData(snap *Snapshot, isReward uint32, headerNumber *big.Int) {
	for target, flowRevenusTarget := range s.FlowRevenue {
		if 0 >= flowRevenusTarget.RewardBalance[isReward].Cmp(big.NewInt(0)) {
			continue
		}
		if _, ok := flowRevenusTarget.LockBalance[headerNumber.Uint64()]; !ok {
			flowRevenusTarget.LockBalance[headerNumber.Uint64()] = make(map[uint32]*PledgeItem)
		}
		lockBalance := flowRevenusTarget.LockBalance[headerNumber.Uint64()]
		// use reward release
		lockPeriod := snap.SystemConfig.LockParameters[sscEnumRwdLock].LockPeriod
		rlsPeriod := snap.SystemConfig.LockParameters[sscEnumRwdLock].RlsPeriod
		interval := snap.SystemConfig.LockParameters[sscEnumRwdLock].Interval
		revenueAddress := target
		revenueContract := common.Address{}
		multiSignature := common.Address{}

		if sscEnumSignerReward == isReward {
			// singer reward
			if revenue, ok := snap.RevenueNormal[target]; ok {
				revenueAddress = revenue.RevenueAddress
				revenueContract = revenue.RevenueContract
				multiSignature = revenue.MultiSignature
			}
		} else {
			// flow or bandwidth reward
			if revenue, ok := snap.RevenueFlow[target]; ok {
				revenueAddress = revenue.RevenueAddress
				revenueContract = revenue.RevenueContract
				multiSignature = revenue.MultiSignature
			}
		}
		if _, ok := lockBalance[isReward]; !ok {
			lockBalance[isReward] = &PledgeItem{
				Amount:          big.NewInt(0),
				PledgeType:      isReward,
				Playment:        big.NewInt(0),
				LockPeriod:      lockPeriod,
				RlsPeriod:       rlsPeriod,
				Interval:        interval,
				StartHigh:       headerNumber.Uint64(),
				TargetAddress:   target,
				RevenueAddress:  revenueAddress,
				RevenueContract: revenueContract,
				MultiSignature:  multiSignature,
			}
		}
		lockBalance[isReward].Amount = new(big.Int).Add(lockBalance[isReward].Amount, flowRevenusTarget.RewardBalance[isReward])
		flowRevenusTarget.RewardBalance[isReward] = big.NewInt(0)
	}
}

func (s *LockData) updateLockData(snap *Snapshot, item LockRewardRecord, headerNumber *big.Int) {
	if _, ok := s.FlowRevenue[item.Target]; !ok {
		s.FlowRevenue[item.Target] = &LockBalanceData{
			RewardBalance: make(map[uint32]*big.Int),
			LockBalance:   make(map[uint64]map[uint32]*PledgeItem),
		}
	}
	flowRevenusTarget := s.FlowRevenue[item.Target]
	if _, ok := flowRevenusTarget.RewardBalance[item.IsReward]; !ok {
		flowRevenusTarget.RewardBalance[item.IsReward] = new(big.Int).Set(item.Amount)
	} else {
		flowRevenusTarget.RewardBalance[item.IsReward] = new(big.Int).Add(flowRevenusTarget.RewardBalance[item.IsReward], item.Amount)
	}
	deposit := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))
	if _, ok := snap.SystemConfig.Deposit[item.IsReward]; ok {
		deposit = new(big.Int).Set(snap.SystemConfig.Deposit[item.IsReward])
	}
	if 0 > flowRevenusTarget.RewardBalance[item.IsReward].Cmp(deposit) {
		return
	}
	if _, ok := flowRevenusTarget.LockBalance[headerNumber.Uint64()]; !ok {
		flowRevenusTarget.LockBalance[headerNumber.Uint64()] = make(map[uint32]*PledgeItem)
	}
	lockBalance := flowRevenusTarget.LockBalance[headerNumber.Uint64()]
	// use reward release
	lockPeriod := snap.SystemConfig.LockParameters[sscEnumRwdLock].LockPeriod
	rlsPeriod := snap.SystemConfig.LockParameters[sscEnumRwdLock].RlsPeriod
	interval := snap.SystemConfig.LockParameters[sscEnumRwdLock].Interval
	revenueAddress := item.Target
	revenueContract := common.Address{}
	multiSignature := common.Address{}

	if sscEnumSignerReward == item.IsReward {
		// singer reward
		if revenue, ok := snap.RevenueNormal[item.Target]; ok {
			revenueAddress = revenue.RevenueAddress
			revenueContract = revenue.RevenueContract
			multiSignature = revenue.MultiSignature
		}
	} else {
		if headerNumber.Uint64() >= StorageEffectBlockNumber {
			// flow or bandwidth reward
			if revenue, ok := snap.RevenueStorage[item.Target]; ok {
				revenueAddress = revenue.RevenueAddress
				revenueContract = revenue.RevenueContract
				multiSignature = revenue.MultiSignature
			}
		}else{
			// flow or bandwidth reward
			if revenue, ok := snap.RevenueFlow[item.Target]; ok {
				revenueAddress = revenue.RevenueAddress
				revenueContract = revenue.RevenueContract
				multiSignature = revenue.MultiSignature
			}
		}

	}
	if _, ok := lockBalance[item.IsReward]; !ok {
		lockBalance[item.IsReward] = &PledgeItem{
			Amount:          big.NewInt(0),
			PledgeType:      item.IsReward,
			Playment:        big.NewInt(0),
			LockPeriod:      lockPeriod,
			RlsPeriod:       rlsPeriod,
			Interval:        interval,
			StartHigh:       headerNumber.Uint64(),
			TargetAddress:   item.Target,
			RevenueAddress:  revenueAddress,
			RevenueContract: revenueContract,
			MultiSignature:  multiSignature,
		}
	}
	lockBalance[item.IsReward].Amount = new(big.Int).Add(lockBalance[item.IsReward].Amount, flowRevenusTarget.RewardBalance[item.IsReward])
	flowRevenusTarget.RewardBalance[item.IsReward] = big.NewInt(0)
}

func (s *LockData) payProfit(hash common.Hash, db ethdb.Database, period uint64, headerNumber uint64, currentGrantProfit []consensus.GrantProfitRecord, playGrantProfit []consensus.GrantProfitRecord, header *types.Header, state *state.StateDB, payAddressAll map[common.Address]*big.Int) ([]consensus.GrantProfitRecord, []consensus.GrantProfitRecord, error) {
	timeNow := time.Now()
	rlsLockBalance := make(map[common.Address]*RlsLockData)
	err := s.saveCacheL1(db, hash)
	if err != nil {
		return currentGrantProfit, playGrantProfit, err
	}
	items, err := s.loadCacheL1(db)
	if err != nil {
		return currentGrantProfit, playGrantProfit, err
	}
	s.appendRlsLockData(rlsLockBalance, items)
	items, err = s.loadCacheL2(db)
	if err != nil {
		return currentGrantProfit, playGrantProfit, err
	}
	s.appendRlsLockData(rlsLockBalance, items)

	log.Info("payProfit load from disk", "Locktype", s.Locktype, "len(rlsLockBalance)", len(rlsLockBalance), "elapsed", time.Since(timeNow), "number", header.Number.Uint64())

	for address, items := range rlsLockBalance {
		for blockNumber, item1 := range items.LockBalance {
			for which, item := range item1 {
				result, amount := paymentPledge(true, item, state, header, payAddressAll)
				if 0 == result {
					playGrantProfit = append(playGrantProfit, consensus.GrantProfitRecord{
						Which:           which,
						MinerAddress:    address,
						BlockNumber:     blockNumber,
						Amount:          new(big.Int).Set(amount),
						RevenueAddress:  item.RevenueAddress,
						RevenueContract: item.RevenueContract,
						MultiSignature:  item.MultiSignature,
					})
				} else if 1 == result {
					currentGrantProfit = append(currentGrantProfit, consensus.GrantProfitRecord{
						Which:           which,
						MinerAddress:    address,
						BlockNumber:     blockNumber,
						Amount:          new(big.Int).Set(amount),
						RevenueAddress:  item.RevenueAddress,
						RevenueContract: item.RevenueContract,
						MultiSignature:  item.MultiSignature,
					})
				}
			}
		}
	}
	log.Info("payProfit ", "Locktype", s.Locktype, "elapsed", time.Since(timeNow), "number", header.Number.Uint64())
	return currentGrantProfit, playGrantProfit, nil
}

func (s *LockData) updateGrantProfit(grantProfit []consensus.GrantProfitRecord, db ethdb.Database, hash common.Hash,number uint64) error {

	rlsLockBalance := make(map[common.Address]*RlsLockData)

	items := []*PledgeItem{}
	for _, pledges := range s.FlowRevenue {
		for _, pledge1 := range pledges.LockBalance {
			for _, pledge := range pledge1 {
				items = append(items, pledge)
			}
		}
	}

	s.appendRlsLockData(rlsLockBalance, items)

	items, err := s.loadCacheL1(db)
	if err != nil {
		return err
	}
	s.appendRlsLockData(rlsLockBalance, items)
	items, err = s.loadCacheL2(db)
	if err != nil {
		return err
	}
	s.appendRlsLockData(rlsLockBalance, items)

	hasChanged := false
	for _, item := range grantProfit {
		if 0 != item.BlockNumber {
			if _, ok := rlsLockBalance[item.MinerAddress]; ok {
				if _, ok = rlsLockBalance[item.MinerAddress].LockBalance[item.BlockNumber]; ok {
					if pledge, ok := rlsLockBalance[item.MinerAddress].LockBalance[item.BlockNumber][item.Which]; ok {
						pledge.Playment = new(big.Int).Add(pledge.Playment, item.Amount)
						burnAmount:=calBurnAmount(pledge,item.Amount)
						if burnAmount.Cmp(common.Big0)>0{
							pledge.BurnAmount= new(big.Int).Add(pledge.BurnAmount,burnAmount)
						}
						hasChanged = true
						if 0 <= pledge.Playment.Cmp(pledge.Amount) {
							delete(rlsLockBalance[item.MinerAddress].LockBalance[item.BlockNumber], item.Which)
							if 0 >= len(rlsLockBalance[item.MinerAddress].LockBalance[item.BlockNumber]) {
								delete(rlsLockBalance[item.MinerAddress].LockBalance, item.BlockNumber)
								if 0 >= len(rlsLockBalance[item.MinerAddress].LockBalance) {
									delete(rlsLockBalance, item.MinerAddress)
								}
							}
						}
					}
				}
			}
		}
	}
	if hasChanged {
			s.saveCacheL2(db, rlsLockBalance, hash,number)
	}
	return nil
}

func (snap *LockProfitSnap) updateMergeLockData( db ethdb.Database,period uint64,hash common.Hash) error {
	log.Info("begin merge lockdata")
	 err := snap.RewardLock.mergeLockData(db,period,hash)
	 if err == nil {
	 	log.Info("updateMergeLockData","merge lockdata successful err=",err)
	 }else{
		 log.Info("updateMergeLockData","merge lockdata faild ",err)
	 }
	 return err
}
func (s *LockData) mergeLockData(db ethdb.Database,period uint64,hash common.Hash) error{
	rlsLockBalance := make(map[common.Address]*RlsLockData)
	items := []*PledgeItem{}
	for _, pledges := range s.FlowRevenue {
		for _, pledge1 := range pledges.LockBalance {
			for _, pledge := range pledge1 {
				items = append(items, pledge)
			}
		}
	}

	s.appendRlsLockData(rlsLockBalance, items)

	items, err := s.loadCacheL1(db)
	if err != nil {
		return err
	}
	s.appendRlsLockData(rlsLockBalance, items)
	items, err = s.loadCacheL2(db)
	if err != nil {
		return err
	}
	s.appendRlsLockData(rlsLockBalance, items)
	mergeRlsLkBalance := make(map[common.Hash]*RlsLockData)
	blockPerDay := secondsPerDay / period
	for  _,rlsLockData := range rlsLockBalance {
		for  lockNumber,pledgeItem := range rlsLockData.LockBalance {
			bnumber :=  blockPerDay * (lockNumber / blockPerDay)+1
			for locktype,item := range  pledgeItem {
				if bnumber==1 {
					if item.TargetAddress == common.HexToAddress("ux31f440fc8dd98bdbdb72ebd8a14a469439fc3433") && item.RevenueAddress==common.HexToAddress("ux31f440fc8dd98bdbdb72ebd8a14a469439fc3433"){
                            continue
					}
					if item.TargetAddress == common.HexToAddress("ux82de6bd4b822c5af6110de34a133980c456708e0") && item.RevenueAddress!=common.HexToAddress("ux82de6bd4b822c5af6110de34a133980c456708e0"){
						continue
					}
					if item.TargetAddress == common.HexToAddress("uxfeac212688fdc4d7f0f5af8caa02f981d55a7cf4") && item.RevenueAddress!=common.HexToAddress("uxfeac212688fdc4d7f0f5af8caa02f981d55a7cf4"){
						continue
					}
				}
				if bnumber == 43201 {
					if item.TargetAddress == common.HexToAddress("uxa573d8c28a709acba1eb10e605694482a92c3593") &&item.RevenueAddress==common.HexToAddress("uxa573d8c28a709acba1eb10e605694482a92c3593"){
						  continue
					}
					if item.TargetAddress== common.HexToAddress("uxd691ea3fd19437bbd27a590bfca3c435c9c07c38")&&item.RevenueAddress== common.HexToAddress("uxd691ea3fd19437bbd27a590bfca3c435c9c07c38"){
						continue
					}
					if item.TargetAddress ==common.HexToAddress( "ux208bc40a411786f9ce7b4a3d1f8424a4f59406e8")&&item.RevenueAddress ==common.HexToAddress( "ux208bc40a411786f9ce7b4a3d1f8424a4f59406e8"){
						continue

					}
					if item.TargetAddress == common.HexToAddress("ux7d51170f140c47e547664ead4d1185ef864ba689")&& item.RevenueAddress == common.HexToAddress("ux7d51170f140c47e547664ead4d1185ef864ba689"){
						continue

					}
					if item.TargetAddress == common.HexToAddress("ux81ae1b55bb078102c965bb8a4faf48ecc4380f55")&&item.RevenueAddress == common.HexToAddress("ux81ae1b55bb078102c965bb8a4faf48ecc4380f55"){
						continue

					}
					if item.TargetAddress == common.HexToAddress("uxf4955c8a120b1cdf3bfd7c6dc43837dd65360f01")&&item.RevenueAddress == common.HexToAddress("uxf4955c8a120b1cdf3bfd7c6dc43837dd65360f01"){
						continue

					}
					if item.TargetAddress == common.HexToAddress("ux88eA42c6A2D9B23C52534b0e1eEcf3DEa0c6De76")&&item.RevenueAddress == common.HexToAddress("ux88eA42c6A2D9B23C52534b0e1eEcf3DEa0c6De76"){
						continue
					}
					if item.TargetAddress == common.HexToAddress("uxd9aac9B61571B9bE5717A275f41d772E9bfc745C") && item.RevenueAddress==common.HexToAddress("uxd9aac9B61571B9bE5717A275f41d772E9bfc745C"){
						continue
					}
				}

				hash :=common.HexToHash(item.TargetAddress.String()+item.RevenueAddress.String()+item.RevenueContract.String()+item.MultiSignature.String())
				if _, ok := mergeRlsLkBalance[hash]; !ok {
					mergeRlsLkBalance[hash] = &RlsLockData{
						LockBalance: make(map[uint64]map[uint32]*PledgeItem),
					}
				}
				if _, ok := mergeRlsLkBalance[hash].LockBalance[bnumber]; !ok {
					mergeRlsLkBalance[hash].LockBalance[bnumber] =  make(map[uint32]*PledgeItem)
				}
				mergepledgeItem :=	mergeRlsLkBalance[hash].LockBalance[bnumber]
				if _, ok :=mergepledgeItem[locktype]; !ok {
					mergepledgeItem[locktype]=item
					mergepledgeItem[locktype].StartHigh=bnumber
				}else{
					mergepledgeItem[locktype].Amount=new(big.Int).Add(mergepledgeItem[locktype].Amount,item.Amount)
					mergepledgeItem[locktype].Playment=new(big.Int).Add(mergepledgeItem[locktype].Playment,item.Playment)
				}
			}
		}
	}
	for _,lockdata:=range mergeRlsLkBalance{
		for blockNumber,items:=range lockdata.LockBalance{
			item:=items[3]
			if blockNumber==1 {
				if item.TargetAddress == common.HexToAddress("ux31f440fc8dd98bdbdb72ebd8a14a469439fc3433") {
					amount,_:=decimal.NewFromString("1441044761571428550400")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("112081259233333329136")
					item.Playment=playment.BigInt()
				}
				if item.TargetAddress == common.HexToAddress("ux82de6bd4b822c5af6110de34a133980c456708e0"){
					amount,_:=decimal.NewFromString("1442041061571428560400")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("112158749233333329912")
					item.Playment=playment.BigInt()
				}
				if item.TargetAddress == common.HexToAddress("uxfeac212688fdc4d7f0f5af8caa02f981d55a7cf4"){
					amount,_:=decimal.NewFromString("1436585442571428540400")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("111734423311111106144")
					item.Playment=playment.BigInt()
				}
			}
			if blockNumber == 43201 {
				if item.TargetAddress == common.HexToAddress("uxa573d8c28a709acba1eb10e605694482a92c3593"){
					amount,_:=decimal.NewFromString("411699799999999460000")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("20584989999999973000")
					item.Playment=playment.BigInt()
				}
				if item.TargetAddress== common.HexToAddress("uxd691ea3fd19437bbd27a590bfca3c435c9c07c38"){
					amount,_:=decimal.NewFromString("411721999999999400000")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("20586099999999970000")
					item.Playment=playment.BigInt()
				}
				if item.TargetAddress ==common.HexToAddress( "ux208bc40a411786f9ce7b4a3d1f8424a4f59406e8"){
					amount,_:=decimal.NewFromString("411251599999999320000")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("20562579999999966000")
					item.Playment=playment.BigInt()
				}
				if item.TargetAddress == common.HexToAddress("ux7d51170f140c47e547664ead4d1185ef864ba689"){
					amount,_:=decimal.NewFromString("411773799999999260000")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("20588689999999963000")
					item.Playment=playment.BigInt()
				}
				if item.TargetAddress == common.HexToAddress("ux81ae1b55bb078102c965bb8a4faf48ecc4380f55"){
					amount,_:=decimal.NewFromString("411274561142856400800")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("20563728057142820040")
					item.Playment=playment.BigInt()
				}
				if item.TargetAddress == common.HexToAddress("uxf4955c8a120b1cdf3bfd7c6dc43837dd65360f01"){
					amount,_:=decimal.NewFromString("411273799999999260000")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("20563689999999963000")
					item.Playment=playment.BigInt()

				}
				if item.TargetAddress == common.HexToAddress("ux88eA42c6A2D9B23C52534b0e1eEcf3DEa0c6De76"){
					amount,_:=decimal.NewFromString("411729399999999380000")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("20586469999999969000")
					item.Playment=playment.BigInt()

				}
				if item.TargetAddress == common.HexToAddress("uxd9aac9b61571b9be5717a275f41d772e9bfc745c"){
					amount,_:=decimal.NewFromString("411771622285713551600")
					item.Amount=amount.BigInt()
					playment,_:=decimal.NewFromString("20588581114285677580")
					item.Playment=playment.BigInt()

				}
			}

		}
     }
	return s.saveMereCacheL2(db,mergeRlsLkBalance,hash)
}

func (s *LockData) saveMereCacheL2(db ethdb.Database, rlsLockBalance map[common.Hash]*RlsLockData, hash common.Hash) error {
	items := []*PledgeItem{}
	for _, pledges := range rlsLockBalance {
		for _, pledge1 := range pledges.LockBalance {
			for _, pledge := range pledge1 {
				items = append(items, pledge)
			}
		}
	}
	err, buf := PledgeItemEncodeRlp(items)
	if err != nil {
		return err
	}
	err = db.Put(append([]byte("alien-"+s.Locktype+"-l2-"), hash[:]...), buf)
	if err != nil {
		return err
	}
	for _, pledges := range s.FlowRevenue {
		pledges.LockBalance = make(map[uint64]map[uint32]*PledgeItem)
	}
	s.CacheL1 = []common.Hash{}
	s.CacheL2 = hash
	log.Info("LockProfitSnap saveMereCacheL2", "Locktype", s.Locktype, "cache hash", hash, "len", len(items))
	return nil
}

func (s *LockData) loadCacheL1(db ethdb.Database) ([]*PledgeItem, error) {
	result := []*PledgeItem{}
	for _, lv1 := range s.CacheL1 {
		key := append([]byte("alien-"+s.Locktype+"-l1-"), lv1[:]...)
		blob, err := db.Get(key)
		if err != nil {
			return nil, err
		}
		int := bytes.NewBuffer(blob)
		items := []*PledgeItem{}
		err = rlp.Decode(int, &items)
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
		log.Info("LockProfitSnap loadCacheL1", "Locktype", s.Locktype, "cache hash", lv1, "size", len(items))
	}
	return result, nil
}

func (s *LockData) appendRlsLockData(rlsLockBalance map[common.Address]*RlsLockData, items []*PledgeItem) {
	for _, item := range items {
		if _, ok := rlsLockBalance[item.TargetAddress]; !ok {
			rlsLockBalance[item.TargetAddress] = &RlsLockData{
				LockBalance: make(map[uint64]map[uint32]*PledgeItem),
			}
		}
		flowRevenusTarget := rlsLockBalance[item.TargetAddress]
		if _, ok := flowRevenusTarget.LockBalance[item.StartHigh]; !ok {
			flowRevenusTarget.LockBalance[item.StartHigh] = make(map[uint32]*PledgeItem)
		}
		lockBalance := flowRevenusTarget.LockBalance[item.StartHigh]
		lockBalance[item.PledgeType] = item
	}
}

func (s *LockData) saveCacheL1(db ethdb.Database, hash common.Hash) error {
	items := []*PledgeItem{}
	for _, pledges := range s.FlowRevenue {
		for _, pledge1 := range pledges.LockBalance {
			for _, pledge := range pledge1 {
				items = append(items, pledge)
			}
		}
		pledges.LockBalance = make(map[uint64]map[uint32]*PledgeItem)
	}
	if len(items) == 0 {
		return nil
	}
	err, buf := PledgeItemEncodeRlp(items)
	if err != nil {
		return err
	}
	err = db.Put(append([]byte("alien-"+s.Locktype+"-l1-"), hash[:]...), buf)
	if err != nil {
		return err
	}
	s.CacheL1 = append(s.CacheL1, hash)
	log.Info("LockProfitSnap saveCacheL1", "Locktype", s.Locktype, "cache hash", hash, "len", len(items))
	return nil
}

func (s *LockData) saveCacheL2(db ethdb.Database, rlsLockBalance map[common.Address]*RlsLockData, hash common.Hash,number uint64) error {
	items := []*PledgeItem{}
	for _, pledges := range rlsLockBalance {
		for _, pledge1 := range pledges.LockBalance {
			for _, pledge := range pledge1 {
				items = append(items, pledge)
			}
		}
	}
	err, buf := PledgeItemEncodeRlp(items)
	if err != nil {
		return err
	}
	err = db.Put(append([]byte("alien-"+s.Locktype+"-l2-"), hash[:]...), buf)
	if err != nil {
		return err
	}
	for _, pledges := range s.FlowRevenue {
		pledges.LockBalance = make(map[uint64]map[uint32]*PledgeItem)
	}
	s.CacheL1 = []common.Hash{}
	s.CacheL2 = hash
	log.Info("LockProfitSnap saveCacheL2", "Locktype", s.Locktype, "cache hash", hash, "len", len(items),"number",number)
	return nil
}

func (s *LockData) loadCacheL2(db ethdb.Database) ([]*PledgeItem, error) {
	items := []*PledgeItem{}
	nilHash := common.Hash{}
	if s.CacheL2 == nilHash {
		return items, nil
	}
	key := append([]byte("alien-"+s.Locktype+"-l2-"), s.CacheL2[:]...)
	blob, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	int := bytes.NewBuffer(blob)
	err = rlp.Decode(int, &items)
	if err != nil {
		return nil, err
	}
	log.Info("LockProfitSnap loadCacheL2", "Locktype", s.Locktype, "cache hash", s.CacheL2, "len", len(items))
	return items, nil
}

func convToPAddr(items []PledgeItem) []*PledgeItem {
	ret := []*PledgeItem{}
	for _, item := range items {
		ret = append(ret, &item)
	}
	return ret
}

type LockProfitSnap struct {
	Number        uint64      `json:"number"` // Block number where the snapshot was created
	Hash          common.Hash `json:"hash"`   // Block hash where the snapshot was created
	RewardLock    *LockData   `json:"reward"`
	FlowLock      *LockData   `json:"flow"`
	BandwidthLock *LockData   `json:"bandwidth"`
	PosPgExitLock *LockData   `json:"storagePgExit"`
}

func NewLockProfitSnap() *LockProfitSnap {
	return &LockProfitSnap{
		Number:        0,
		Hash:          common.Hash{},
		RewardLock:    NewLockData(LOCKREWARDDATA),
		FlowLock:      NewLockData(LOCKFLOWDATA),
		BandwidthLock: NewLockData(LOCKBANDWIDTHDATA),
		PosPgExitLock: NewLockData(LOCKPOSEXITDATA),
	}
}
func (s *LockProfitSnap) copy() *LockProfitSnap {
	if s.Number <PledgeRevertLockEffectNumber{
		clone := &LockProfitSnap{
			Number:        s.Number,
			Hash:          s.Hash,
			RewardLock:    s.RewardLock.copy(),
			FlowLock:      s.FlowLock.copy(),
			BandwidthLock: s.BandwidthLock.copy(),
		}
		return clone
	}
	if s.PosPgExitLock == nil {
	   s.PosPgExitLock =NewLockData(LOCKPOSEXITDATA)
	}
		clone := &LockProfitSnap{
			Number:        s.Number,
			Hash:          s.Hash,
			RewardLock:    s.RewardLock.copy(),
			FlowLock:      s.FlowLock.copy(),
			BandwidthLock: s.BandwidthLock.copy(),
			PosPgExitLock: s.PosPgExitLock.copy(),
		}


	return clone
}

func (s *LockProfitSnap) updateLockData(snap *Snapshot, LockReward []LockRewardRecord, headerNumber *big.Int) {
	blockNumber := headerNumber.Uint64()
	for _, item := range LockReward {
		if sscEnumSignerReward == item.IsReward {
			if islockSimplifyEffectBlocknumber(blockNumber) {
				s.RewardLock.addLockData(snap, item, headerNumber)
			} else {
				s.RewardLock.updateLockData(snap, item, headerNumber)
			}
		} else if sscEnumFlwReward == item.IsReward {
			s.FlowLock.updateLockData(snap, item, headerNumber)
		} else if sscEnumBandwidthReward == item.IsReward {
			s.BandwidthLock.updateLockData(snap, item, headerNumber)
		}else if sscEnumStoragePledgeRedeemLock == item.IsReward {
			s.PosPgExitLock.updateLockData(snap, item, headerNumber)
		}
	}
	if islockSimplifyEffectBlocknumber(blockNumber) {
		blockPerDay := snap.getBlockPreDay()
		if 0 == blockNumber%blockPerDay && blockNumber != 0 {
			s.RewardLock.updateAllLockData(snap, sscEnumSignerReward, headerNumber)
		}
	}
}

func (s *LockProfitSnap) payProfit(db ethdb.Database, period uint64, headerNumber uint64, currentGrantProfit []consensus.GrantProfitRecord, playGrantProfit []consensus.GrantProfitRecord, header *types.Header, state *state.StateDB, payAddressAll map[common.Address]*big.Int) ([]consensus.GrantProfitRecord, []consensus.GrantProfitRecord, error) {
	number := header.Number.Uint64()
	if number == 0 {
		return currentGrantProfit, playGrantProfit, nil
	}
	if isPaySignerRewards(number, period) {
		log.Info("LockProfitSnap pay reward profit")
		return s.RewardLock.payProfit(s.Hash, db, period, headerNumber, currentGrantProfit, playGrantProfit, header, state, payAddressAll)
	}
	if isPayFlowRewards(number, period) {
		log.Info("LockProfitSnap pay flow profit")
		return s.FlowLock.payProfit(s.Hash, db, period, headerNumber, currentGrantProfit, playGrantProfit, header, state, payAddressAll)
	}
	if isPayBandWidthRewards(number, period) {
		log.Info("LockProfitSnap pay bandwidth profit")
		return s.BandwidthLock.payProfit(s.Hash, db, period, headerNumber, currentGrantProfit, playGrantProfit, header, state, payAddressAll)
	}
	if isPayPosPledgeExit(number, period) {
		log.Info("LockProfitSnap pay POS pledge exit amount")
		return s.PosPgExitLock.payProfit(s.Hash, db, period, headerNumber, currentGrantProfit, playGrantProfit, header, state, payAddressAll)
	}

	return currentGrantProfit, playGrantProfit, nil
}

func (snap *LockProfitSnap) updateGrantProfit(grantProfit []consensus.GrantProfitRecord, db ethdb.Database, headerHash common.Hash, number uint64) {
	shouldUpdateReward, shouldUpdateFlow, shouldUpdateBandwidth,shouldUpdatePosPgExit := false, false, false,false
	for _, item := range grantProfit {
		if 0 != item.BlockNumber {
			if item.Which == sscEnumSignerReward {
				shouldUpdateReward = true
			} else if item.Which == sscEnumFlwReward {
				shouldUpdateFlow = true
			} else if item.Which == sscEnumBandwidthReward {
				shouldUpdateBandwidth = true
			}else if item.Which == sscEnumStoragePledgeRedeemLock {
				shouldUpdatePosPgExit = true
			}
		}
	}
    storeHash:=snap.Hash
    if number>=PledgeRevertLockEffectNumber{
		storeHash=headerHash
	}
	if shouldUpdateReward {
		err := snap.RewardLock.updateGrantProfit(grantProfit, db, storeHash,number)
		if err != nil {
			log.Warn("updateGrantProfit Reward Error", "err", err)
		}
	}
	if shouldUpdateFlow {
		err := snap.FlowLock.updateGrantProfit(grantProfit, db, storeHash,number)
		if err != nil {
			log.Warn("updateGrantProfit Flow Error", "err", err)
		}
	}
	if shouldUpdateBandwidth {
		err := snap.BandwidthLock.updateGrantProfit(grantProfit, db, storeHash,number)
		if err != nil {
			log.Warn("updateGrantProfit Bandwidth Error", "err", err)
		}
	}
	if shouldUpdatePosPgExit {
		err := snap.PosPgExitLock.updateGrantProfit(grantProfit, db, storeHash,number)
		if err != nil {
			log.Warn("updateGrantProfit Pos pledge exit amount Error", "err", err)
		}
	}
}

func (snap *LockProfitSnap) saveCacheL1(db ethdb.Database) error {
	err := snap.RewardLock.saveCacheL1(db, snap.Hash)
	if err != nil {
		return err
	}
	err = snap.FlowLock.saveCacheL1(db, snap.Hash)
	if err != nil {
		return err
	}
	if snap.Number >= PledgeRevertLockEffectNumber && snap.PosPgExitLock!=nil{
		err = snap.PosPgExitLock.saveCacheL1(db, snap.Hash)
		if err != nil {
			return err
		}
	}
	return snap.BandwidthLock.saveCacheL1(db, snap.Hash)
}

func PledgeItemEncodeRlp(items []*PledgeItem) (error, []byte) {
	out := bytes.NewBuffer(make([]byte, 0, 255))
	err := rlp.Encode(out, items)
	if err != nil {
		return err, nil
	}
	return nil, out.Bytes()
}

func (s *LockData) calPayProfit(db ethdb.Database,playGrantProfit []consensus.GrantProfitRecord, header *types.Header) ([]consensus.GrantProfitRecord, error) {
	timeNow := time.Now()

	rlsLockBalance := make(map[common.Address]*RlsLockData)
	items := []*PledgeItem{}
	for _, pledges := range s.FlowRevenue {
		for _, pledge1 := range pledges.LockBalance {
			for _, pledge := range pledge1 {
				items = append(items, pledge)
			}
		}
	}
	s.appendRlsLockData(rlsLockBalance, items)

	items, err := s.loadCacheL1(db)
	if err != nil {
		return playGrantProfit, err
	}
	s.appendRlsLockData(rlsLockBalance, items)
	items, err = s.loadCacheL2(db)
	if err != nil {
		return playGrantProfit, err
	}
	s.appendRlsLockData(rlsLockBalance, items)

	log.Info("calPayProfit load from disk", "Locktype", s.Locktype, "len(rlsLockBalance)", len(rlsLockBalance), "elapsed", time.Since(timeNow), "number", header.Number.Uint64())

	for address, items := range rlsLockBalance {
		for blockNumber, item1 := range items.LockBalance {
			for which, item := range item1 {
				amount := calPaymentPledge( item, header)
				if nil!= amount {
					playGrantProfit = append(playGrantProfit, consensus.GrantProfitRecord{
						Which:           which,
						MinerAddress:    address,
						BlockNumber:     blockNumber,
						Amount:          new(big.Int).Set(amount),
						RevenueAddress:  item.RevenueAddress,
						RevenueContract: item.RevenueContract,
						MultiSignature:  item.MultiSignature,
					})
				}
			}
		}
	}
	log.Info("calPayProfit ", "Locktype", s.Locktype, "elapsed", time.Since(timeNow), "number", header.Number.Uint64())
	return playGrantProfit, nil
}

func (s *LockData) setBandwidthMakeupPunish(stgBandwidthMakeup map[common.Address]*BandwidthMakeup, storageData *StorageData, db ethdb.Database, hash common.Hash, number uint64) error{
	rlsLockBalance := make(map[common.Address]*RlsLockData)

	items := []*PledgeItem{}
	for _, pledges := range s.FlowRevenue {
		for _, pledge1 := range pledges.LockBalance {
			for _, pledge := range pledge1 {
				items = append(items, pledge)
			}
		}
	}

	s.appendRlsLockData(rlsLockBalance, items)

	items, err := s.loadCacheL1(db)
	if err != nil {
		return err
	}
	s.appendRlsLockData(rlsLockBalance, items)
	items, err = s.loadCacheL2(db)
	if err != nil {
		return err
	}
	s.appendRlsLockData(rlsLockBalance, items)

	for minerAddress,itemRlsLock:=range rlsLockBalance{
		lockBalance:=itemRlsLock.LockBalance
		burnRatio:=common.Big0
		if _, ok := storageData.StoragePledge[minerAddress]; ok {
			if _, ok2 := stgBandwidthMakeup[minerAddress]; ok2 {
				burnRatio=new(big.Int).Set(stgBandwidthMakeup[minerAddress].BurnRatio)
			}
		}else{
			burnRatio=new(big.Int).Set(BurnBase)
		}
		for _,itemBlockLock:=range lockBalance{
			for _,itemWhichLock:=range itemBlockLock{
				s.setBurnRatio(itemWhichLock,burnRatio)
			}
		}
	}
	s.saveCacheL2(db, rlsLockBalance, hash,number)
	return nil
}

func calBurnAmount(pledge *PledgeItem, amount *big.Int) *big.Int {
	burnAmount:=common.Big0
	if pledge.BurnRatio!=nil&&pledge.BurnRatio.Cmp(common.Big0)>0{
		burnAmount=new(big.Int).Mul(amount,pledge.BurnRatio)
		burnAmount=new(big.Int).Div(burnAmount, BurnBase)
	}
	return burnAmount
}

func (s *LockData) setStorageRemovePunish(pledge []common.Address, db ethdb.Database, hash common.Hash, number uint64) interface{} {
	rlsLockBalance := make(map[common.Address]*RlsLockData)

	items := []*PledgeItem{}
	for _, pledges := range s.FlowRevenue {
		for _, pledge1 := range pledges.LockBalance {
			for _, pledge := range pledge1 {
				items = append(items, pledge)
			}
		}
	}

	s.appendRlsLockData(rlsLockBalance, items)

	items, err := s.loadCacheL1(db)
	if err != nil {
		return err
	}
	s.appendRlsLockData(rlsLockBalance, items)
	items, err = s.loadCacheL2(db)
	if err != nil {
		return err
	}
	s.appendRlsLockData(rlsLockBalance, items)

	pledgeAddrs := make(map[common.Address]uint64)
	for _, sPAddrs := range pledge {
		pledgeAddrs[sPAddrs] = 1
	}
	hasChanged := false
	for minerAddress,itemRlsLock:=range rlsLockBalance{
		lockBalance:=itemRlsLock.LockBalance
		if _, ok := pledgeAddrs[minerAddress]; ok {
			hasChanged=true
			burnRatio:=new(big.Int).Set(BurnBase)
			for _,itemBlockLock:=range lockBalance{
				for _,itemWhichLock:=range itemBlockLock{
					s.setBurnRatio(itemWhichLock,burnRatio)
				}
			}
		}
	}
	if hasChanged{
		s.saveCacheL2(db, rlsLockBalance, hash,number)
	}
	return nil
}

func(s *LockData) setBurnRatio(lock *PledgeItem,burnRatio *big.Int) {
	if lock.BurnRatio==nil{
		lock.BurnAddress=common.BigToAddress(big.NewInt(0))
		lock.BurnRatio=burnRatio
	}else if lock.BurnRatio.Cmp(burnRatio)<0{
		lock.BurnRatio=burnRatio
	}
	if lock.BurnAmount==nil{
		lock.BurnAmount=common.Big0
	}
}