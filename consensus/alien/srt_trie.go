package alien

import (
	"errors"
	"github.com/UltronGlow/UltronGlow-Origin/common"
	"github.com/UltronGlow/UltronGlow-Origin/ethdb"
	"github.com/UltronGlow/UltronGlow-Origin/log"
	"github.com/UltronGlow/UltronGlow-Origin/rlp"
	"github.com/UltronGlow/UltronGlow-Origin/trie"
	"math/big"
)

var (
	errNotEnoughSrt = errors.New("not enough srt")
)

type SrtAccount struct {
	Address  common.Address
	Balance  *big.Int
}

func (s *SrtAccount) Encode() ([]byte, error) {
	return rlp.EncodeToBytes(s)
}

func decodeSrtAccount(buf []byte) *SrtAccount {
	s := &SrtAccount{}
		err := rlp.DecodeBytes(buf, s)
		if err != nil {
			//fmt.Println("decode srtaccoutn failed", "error", err)
			return nil
		}else {
			return s
		}
}

func (s *SrtAccount) GetBalance() *big.Int {
	return s.Balance
}

func (s *SrtAccount) SetBalance(amount *big.Int) {
	s.Balance = amount
}

func (s *SrtAccount) AddBalance(amount *big.Int) {
	s.Balance = new(big.Int).Add(s.Balance, amount)
}

func (s *SrtAccount) SubBalance(amount *big.Int) error {
	if s.Balance.Cmp(amount) < 0 {
		return errNotEnoughSrt
	}
	s.Balance = new(big.Int).Sub(s.Balance, amount)
	return nil
}

//=========================================================================
type SrtTrie struct {
	trie 	*trie.SecureTrie
	db      ethdb.Database
	triedb 	*trie.Database
}

func (s *SrtTrie) GetOrNewAccount(addr common.Address) *SrtAccount {
	var obj *SrtAccount
	objData := s.trie.Get(addr.Bytes())
	obj = decodeSrtAccount(objData)
	if obj != nil {
		return obj
	}
	obj = &SrtAccount {
		Address: addr,
		Balance: common.Big0,
	}
	return obj
}

func (s *SrtTrie) TireDB() *trie.Database {
	return s.triedb
}

func (s *SrtTrie) getBalance(addr common.Address) *big.Int {
	obj := s.GetOrNewAccount(addr)
	if obj == nil {
		return common.Big0
	}
	return obj.GetBalance()
}

func (s *SrtTrie) setBalance(addr common.Address, amount *big.Int) {
	obj := s.GetOrNewAccount(addr)
	if obj == nil {
		log.Warn("srttrie setbalance", "result", "failed")
		return
	}
	obj.SetBalance(amount)
	value, _ := obj.Encode()
	s.trie.Update(addr.Bytes(), value)
}

func (s *SrtTrie) addBalance(addr common.Address, amount *big.Int) {
	obj := s.GetOrNewAccount(addr)
	if obj == nil {
		log.Warn("srttrie addbalance", "result", "failed")
		return
	}
	obj.AddBalance(amount)
	value, _ := obj.Encode()
	s.trie.Update(addr.Bytes(), value)
}

func (s *SrtTrie) subBalance(addr common.Address, amount *big.Int) error{
	obj := s.GetOrNewAccount(addr)
	if obj == nil {
		log.Warn("srttrie subbalance", "result", "failed")
		return errNotEnoughSrt
	}
	obj.SubBalance(amount)
	value, _ := obj.Encode()
	s.trie.Update(addr.Bytes(), value)
	return nil
}

func (s *SrtTrie) cmpBalance(addr common.Address, amount *big.Int) int{
	obj := s.GetOrNewAccount(addr)
	if obj == nil {
		log.Warn("srttrie cmpbalance", "error", "load account failed")
		return -1
	}

	return obj.Balance.Cmp(amount)
}

func (s *SrtTrie) Hash() common.Hash {
	return s.trie.Hash()
}

func (s *SrtTrie) commit() (root common.Hash, err error){
	hash, err := s.trie.Commit(nil)
	if err != nil {
		return common.Hash{}, err
	}
	s.triedb.Commit(hash, true, nil)
	return hash, nil
}

//====================================================================================
func NewSrtTrie(root common.Hash, db ethdb.Database) (*SrtTrie, error) {
	triedb := trie.NewDatabase(db)
	tr, err := trie.NewSecure(root, triedb)
	if err != nil {
		log.Warn("srttrie open srt trie failed", "root", root)
		return nil, err
	}

	return &SrtTrie{
		trie: tr,
		db: db,
		triedb: triedb,
	}, nil
}

func (s *SrtTrie) Get (addr common.Address) *big.Int{
	return s.getBalance(addr)
}

func (s *SrtTrie) Set (addr common.Address, amount *big.Int) {
	s.setBalance(addr, amount)
}

func (s *SrtTrie) Add (addr common.Address, amount *big.Int) {
	s.addBalance(addr, amount)
}

func (s *SrtTrie) Sub (addr common.Address, amount *big.Int) error{
	return s.subBalance(addr, amount)
}

func (s *SrtTrie) Del(addr common.Address) {
	s.setBalance(addr, common.Big0)
	s.trie.Delete(addr.Bytes())
}

func (s *SrtTrie) Copy() SRTState{
	root, _ := s.Save(nil)
	trie, _ := NewTrieSRTState(root, s.db)
	return trie
}

func (s *SrtTrie) Load(db ethdb.Database, hash common.Hash) error{
	return nil
}

func (s *SrtTrie) Save(db ethdb.Database) (common.Hash, error){
	return s.commit()
}

func (s *SrtTrie) Root() common.Hash {
	return s.Hash()
}

func (s *SrtTrie) GetAll() map[common.Address]*big.Int {
	found := make(map[common.Address]*big.Int)
	it := trie.NewIterator(s.trie.NodeIterator(nil))
	for it.Next() {
		acc := decodeSrtAccount(it.Value)
		if nil != acc {
			found[acc.Address] = acc.Balance
		}
	}
	return found
}
