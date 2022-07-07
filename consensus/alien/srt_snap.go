package alien

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/UltronGlow/UltronGlow-Origin/common"
	"github.com/UltronGlow/UltronGlow-Origin/ethdb"
	"github.com/UltronGlow/UltronGlow-Origin/log"
	"github.com/UltronGlow/UltronGlow-Origin/rlp"
	"golang.org/x/crypto/sha3"
	"math/big"
	"sort"
)

type SRTState interface {
	Set(addr common.Address, amount *big.Int)
	Add(addr common.Address, amount *big.Int)
	Sub(addr common.Address, amount *big.Int) error
	Get(addr common.Address) *big.Int
	Del(addr common.Address)
	Copy() SRTState
	Load(db ethdb.Database, hash common.Hash) error
	Save(db ethdb.Database) (common.Hash, error)
	Root() common.Hash
	GetAll() map[common.Address]*big.Int
}

func NewSRT(root common.Hash,db ethdb.Database) (SRTState,error) {
	//state,err := NewDefaultSRTState()
	state ,err:= NewTrieSRTState(root,db)
	return state,err
}

func NewTrieSRTState(root common.Hash, db ethdb.Database) (SRTState,error) {
	return NewSrtTrie(root,db)
}

type DefaultSRTState struct {
	state       map[common.Address]*big.Int
	StorageHash common.Hash `json:"storage"`
	Hash        common.Hash `json:"root"`
}

func NewDefaultSRTState() (SRTState,error) {
	return &DefaultSRTState{
		state:       make(map[common.Address]*big.Int),
		StorageHash: common.Hash{},
		Hash:        common.Hash{},
	},nil
}


// TODO calc Hash
func (c *DefaultSRTState) calcHash() {
	srtBal:=c.state
	srtBalArr:=make([]SrtBalRecord,0)
	for addr,amount:=range srtBal{
		srtBalArr=append(srtBalArr,SrtBalRecord{
			Address:addr,
			Amount:amount,
		})
	}
	grantProfitSlice:=SrtBalSlice(srtBalArr)
	sort.Sort(grantProfitSlice)
	hasher := sha3.NewLegacyKeccak256()
	rlp.Encode(hasher, grantProfitSlice)
	var hash common.Hash
	hasher.Sum(hash[:0])
	c.Hash=hash
}
func (c *DefaultSRTState) Set(addr common.Address, amount *big.Int) {
	c.state[addr] =new(big.Int).Set(amount)
	c.calcHash()
}

func (c *DefaultSRTState) Add(addr common.Address, amount *big.Int)  {
	if _, ok := c.state[addr]; !ok {
		c.state[addr] = new(big.Int).Set(amount)
	} else {
		c.state[addr] = new(big.Int).Add(c.state[addr],amount)
	}
	c.calcHash()
}

func (c *DefaultSRTState) Sub(addr common.Address, amount *big.Int) error {
	if _, ok := c.state[addr]; !ok {
		return errors.New(fmt.Sprintf("No SRT. Address=%v", addr.String()))
	} else {
		if c.state[addr].Cmp(amount)<0 {
			return errors.New(fmt.Sprintf(" SRT. InsufficientAddress=%v. SRT=%v. Need=%v", addr.String(), c.state[addr], amount))
		}
		c.state[addr] = new(big.Int).Sub(c.state[addr],amount)
	}
	c.calcHash()
	return nil
}
func (c *DefaultSRTState) Get(addr common.Address) *big.Int {
	if _, ok := c.state[addr]; !ok {
		return common.Big0
	}
	return c.state[addr]
}
func (c *DefaultSRTState) Del(addr common.Address) {
	delete(c.state, addr)
}
func (c *DefaultSRTState) Copy() SRTState {
	state := &DefaultSRTState{state: make(map[common.Address]*big.Int), StorageHash: c.StorageHash, Hash: c.Hash}
	for addr, amount := range c.state {
		state.state[addr] = new(big.Int).Set(amount)
	}
	return state
}

type SRTItem struct {
	Addr   common.Address
	Amount *big.Int
}

func encodeSRT(items []*SRTItem) (error, []byte) {
	out := bytes.NewBuffer(make([]byte, 0, 255))
	err := rlp.Encode(out, items)
	if err != nil {
		return err, nil
	}
	return nil, out.Bytes()
}

func (c *DefaultSRTState) Load(db ethdb.Database, hash common.Hash) error {
	key := append([]byte("srt-"), hash[:]...)
	blob, err := db.Get(key)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(blob)
	items := []*SRTItem{}
	err = rlp.Decode(buf, &items)
	if err != nil {
		log.Warn("DefaultSRTState Load", "hash", hash, "error", err)
		return err
	}
	c.state = make(map[common.Address]*big.Int)
	for _, item := range items {
		c.state[item.Addr] = new(big.Int).Set(item.Amount)
	}
	c.StorageHash = hash
	c.Hash = hash
	return nil
}
func (c *DefaultSRTState) Save(db ethdb.Database) (common.Hash, error) {
	if c.Hash==c.StorageHash {
		log.Info("DefaultSRTState Save","SRT not change",c.Hash)
		return common.Hash{}, nil
	}
	items := []*SRTItem{}
	for addr, amount := range c.state {
		items = append(items, &SRTItem{Addr: addr, Amount: amount})
	}

	err, buf := encodeSRT(items)
	if err != nil {
		return common.Hash{}, err
	}
	err = db.Put(append([]byte("srt-"), c.Hash[:]...), buf)
	if err != nil {
		return common.Hash{}, err
	}
	c.StorageHash = c.Hash
	return c.StorageHash, nil
}
func (c *DefaultSRTState) Root() common.Hash {
	return c.Hash
}

func (c *DefaultSRTState) GetAll() map[common.Address]*big.Int {
	bals:=make(map[common.Address]*big.Int)
	for addr, amount := range c.state {
		bals[addr] = new(big.Int).Set(amount)
	}
	return bals
}

func (s *Snapshot) checkEnoughSRT(sRent []LeaseRequestRecord, rent LeaseRequestRecord, number uint64, db ethdb.Database) bool {
	if s.SRT==nil{
		return false
	}
	srtAmount:=new(big.Int).Mul(rent.Duration,rent.Price)
	srtAmount=new(big.Int).Mul(srtAmount,rent.Capacity)
	srtAmount=new(big.Int).Div(srtAmount,gbTob)
	for _, item := range sRent {
		if item.Tenant==rent.Tenant{
			itemSrtAmount:=new(big.Int).Mul(item.Duration,item.Price)
			itemSrtAmount=new(big.Int).Mul(itemSrtAmount,item.Capacity)
			itemSrtAmount=new(big.Int).Div(itemSrtAmount,gbTob)
			srtAmount=new(big.Int).Add(srtAmount,itemSrtAmount)
		}
	}
	balance:=s.SRT.Get(rent.Tenant)
	if balance.Cmp(srtAmount)>=0 {
		return true
	}
	return false
}


func (s *Snapshot) checkEnoughSRTPg(sRentPg []LeasePledgeRecord, rent LeasePledgeRecord,number uint64, db ethdb.Database) bool {
	if s.SRT==nil{
		return false
	}
	srtAmount:=rent.BurnSRTAmount
	for _, item := range sRentPg {
		if item.BurnSRTAddress==rent.BurnSRTAddress{
			srtAmount=new(big.Int).Add(srtAmount,item.BurnSRTAmount)
		}
	}
	balance:=s.SRT.Get(rent.BurnSRTAddress)
	if balance.Cmp(srtAmount)>=0 {
		return true
	}
	return false
}

func (s *Snapshot) checkEnoughSRTReNew(currentSRentReNew []LeaseRenewalRecord, sRentReNew LeaseRenewalRecord, number uint64, db ethdb.Database) bool {
	if s.SRT==nil{
		return false
	}
	srtAmount:=new(big.Int).Mul(sRentReNew.Duration,sRentReNew.Price)
	srtAmount=new(big.Int).Mul(srtAmount,sRentReNew.Capacity)
	srtAmount=new(big.Int).Div(srtAmount,gbTob)
	for _, item := range currentSRentReNew {
		if item.Tenant==sRentReNew.Tenant{
			itemSrtAmount:=new(big.Int).Mul(item.Duration,item.Price)
			itemSrtAmount=new(big.Int).Mul(itemSrtAmount,item.Capacity)
			itemSrtAmount=new(big.Int).Div(itemSrtAmount,gbTob)
			srtAmount=new(big.Int).Add(srtAmount,itemSrtAmount)
		}
	}
	balance:=s.SRT.Get(sRentReNew.Tenant)
	if balance.Cmp(srtAmount)>=0 {
		return true
	}
	return false

}

func (s *Snapshot) checkEnoughSRTReNewPg(sRentPg []LeaseRenewalPledgeRecord, rent LeaseRenewalPledgeRecord,number uint64, db ethdb.Database) bool {
	if s.SRT==nil{
		return false
	}
	srtAmount:=rent.BurnSRTAmount
	for _, item := range sRentPg {
		if item.BurnSRTAddress==rent.BurnSRTAddress{
			srtAmount=new(big.Int).Add(srtAmount,item.BurnSRTAmount)
		}
	}
	balance:=s.SRT.Get(rent.BurnSRTAddress)
	if balance.Cmp(srtAmount)>=0 {
		return true
	}
	return false
}

func (s *Snapshot) burnSRTAmount(pg []LeasePledgeRecord, number uint64, db ethdb.Database) {
	if s.SRT==nil{
		return
	}
	if pg!=nil&&len(pg)>0 {
		for _, item := range pg {
			balance:=s.SRT.Get(item.BurnSRTAddress)
			if balance.Cmp(item.BurnSRTAmount)>0 {
				s.SRT.Sub(item.BurnSRTAddress,item.BurnSRTAmount)
			}else{
				s.SRT.Del(item.BurnSRTAddress)
			}
		}
	}
}

func (s *Snapshot) burnSRTAmountReNew(pg []LeaseRenewalPledgeRecord, number uint64, db ethdb.Database) {
	if s.SRT==nil{
		return
	}
	if pg!=nil&&len(pg)>0 {
		for _, item := range pg {
			balance:=s.SRT.Get(item.BurnSRTAddress)
			if balance.Cmp(item.BurnSRTAmount)>0 {
				s.SRT.Sub(item.BurnSRTAddress,item.BurnSRTAmount)
			}else{
				s.SRT.Del(item.BurnSRTAddress)
			}
		}
	}
}

type SrtBalRecord struct {
	Address    common.Address
	Amount          *big.Int
}

type SrtBalSlice []SrtBalRecord

func (s SrtBalSlice) Len() int      { return len(s) }
func (s SrtBalSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SrtBalSlice) Less(i, j int) bool {
	isLess := s[i].Amount.Cmp(s[j].Amount)
	if isLess > 0 {
		return true
	} else if isLess < 0 {
		return false
	}
	return bytes.Compare(s[i].Address.Bytes(), s[j].Address.Bytes()) > 0
}