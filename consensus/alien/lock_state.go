package alien

import (
	"encoding/json"
	"fmt"
	"github.com/UltronGlow/UltronGlow-Origin/common"
	"github.com/UltronGlow/UltronGlow-Origin/consensus/alien/extrastate"
	"github.com/UltronGlow/UltronGlow-Origin/core/state"
	"github.com/UltronGlow/UltronGlow-Origin/ethdb"
	"github.com/UltronGlow/UltronGlow-Origin/log"
)

const (
	signerRewardKey        = "signerReward-%d"
)
type LockState interface {
	//New(root common.Hash) (*LockData, error)
	PayLockReward(LockAccountsRoot common.Hash, number uint64, state *state.StateDB) error
	CommitData() (common.Hash, common.Hash, error)
	AddLockReward(LockReward []LockRewardRecord, snap *Snapshot, db ethdb.Database, number uint64) ([]LockRewardRecord,error)
	//GetPaysAtNumber(number uint64) (*SnapshotPay)
}

func NewLockState(root, lockaccounts common.Hash,number uint64) (LockState, error) {
	return NewDefaultLockState(root)
	//return NewExtraLockState(root, lockaccounts)
}

/**
 * ExtraStateDB
 */
type DefaultLockState struct {
}

func NewDefaultLockState(root common.Hash) (*DefaultLockState, error) {
	return &DefaultLockState{}, nil
}

func (c *DefaultLockState) PayLockReward(LockAccountsRoot common.Hash, number uint64, state *state.StateDB) error {
	return nil
}

func (c *DefaultLockState) CommitData() (common.Hash, common.Hash, error) {
	return common.Hash{}, common.Hash{}, nil
}

func (c *DefaultLockState) AddLockReward(LockReward []LockRewardRecord, snap *Snapshot, db ethdb.Database, number uint64) ([]LockRewardRecord,error) {
	return LockReward,nil
}

type ExtraLockState struct {
	es *extrastate.ExtraStateDB
}

func NewExtraLockState(root, lockaccounts common.Hash) (*ExtraLockState, error) {
	es, err := extrastate.ExtraStateAt(root)
	if err != nil {
		log.Error("extrastate open failed", "root", root, "err", err)
		return nil, err
	}
	_, err = es.LoadLockAccounts(lockaccounts)
	if err != nil {
		return nil, err
	}
	return &ExtraLockState{es: es}, nil
}

func (c *ExtraLockState) PayLockReward(LockAccountsRoot common.Hash, number uint64, state *state.StateDB) error {
	/*
	_, err := c.es.LoadLockAccounts(LockAccountsRoot)
	if err != nil {
		return err
	}
	 */

	// process extrastate grantlist
	c.es.PayLockReward(number, state)
	return nil
}

func (c *ExtraLockState) CommitData() (common.Hash, common.Hash, error) {
	return extrastate.CommitData(c.es)
	/*
	stateRoot, err := c.es.Commit(true)
	if err != nil {
		return common.Hash{}, common.Hash{}, nil
	}
	err = c.es.Database().TrieDB().Commit(stateRoot, true, nil)
	if err != nil {
		return common.Hash{}, common.Hash{}, nil
	}
	//lockAccountsRoot := c.es.GetGrantListHash()
	lockAccountsRoot, err := c.es.PutLockAccounts()
	if err != nil {
		return common.Hash{}, common.Hash{}, nil
	}
	return stateRoot, lockAccountsRoot, nil
	 */
}

func (c *ExtraLockState) AddLockReward(LockReward []LockRewardRecord, snap *Snapshot, db ethdb.Database, number uint64) ([]LockRewardRecord,error) {
	if nil != LockReward {
		signerReward:=make([]SpaceRewardRecord,0)
		for _, item := range LockReward {
			revenueAddress := item.Target
			if sscEnumSignerReward == item.IsReward {
				if revenue, ok := snap.RevenueNormal[item.Target]; ok {
					revenueAddress = revenue.RevenueAddress
				}
				signerReward=append(signerReward,SpaceRewardRecord{
					Target:item.Target,
					Amount:item.Amount,
					Revenue:revenueAddress,
				})
			}else{
				if revenue, ok := snap.RevenueStorage[item.Target]; ok {
					revenueAddress = revenue.RevenueAddress
				}
			}
			c.es.AddLockReward(revenueAddress, item.Amount)
		}
        err:=c.SaveSignerRewardTodb(signerReward,db,number)
        if err!=nil {
        	return make([]LockRewardRecord, 0),err
		}
	}
	return make([]LockRewardRecord, 0),nil
}

func (c *ExtraLockState) SaveSignerRewardTodb(signerReward []SpaceRewardRecord, db ethdb.Database, number uint64) error {
	key := fmt.Sprintf(signerRewardKey, number)
	blob, err := json.Marshal(signerReward)
	if err != nil {
		return err
	}
	err = db.Put([]byte(key), blob)
	if err != nil {
		return err
	}
	return nil
}
