package alien

import (
	"bytes"
	"fmt"
	"github.com/UltronGlow/UltronGlow-Origin/common"
	"github.com/UltronGlow/UltronGlow-Origin/core/types"
	"github.com/UltronGlow/UltronGlow-Origin/ethdb"
	"github.com/UltronGlow/UltronGlow-Origin/log"
	"github.com/UltronGlow/UltronGlow-Origin/rlp"
	"math/big"
)

type FlowMinerSnap struct {
	DayStartTime       uint64                                              `json:"dayStartTime"`
	FlowMinerPrevTotal uint64                                              `json:"flowminerPrevTotal"`
	FlowMiner          map[common.Address]map[common.Hash]*FlowMinerReport `json:"flowminerCurr"`
	FlowMinerPrev      map[common.Address]map[common.Hash]*FlowMinerReport `json:"flowminerPrev"`
	FlowMinerCache     []string                                            `json:"flowminerCurCache"`
	FlowMinerPrevCache []string                                            `json:"flowminerPrevCache"`
}

func NewFlowMinerSnap(dayStartTime uint64) *FlowMinerSnap {
	return &FlowMinerSnap{
		DayStartTime:       dayStartTime,
		FlowMinerPrevTotal: 0,
		FlowMiner:          make(map[common.Address]map[common.Hash]*FlowMinerReport),
		FlowMinerPrev:      make(map[common.Address]map[common.Hash]*FlowMinerReport),
		FlowMinerCache:     []string{},
		FlowMinerPrevCache: []string{},
	}
}

func (s *FlowMinerSnap) copy() *FlowMinerSnap {
	clone := &FlowMinerSnap{
		DayStartTime:       s.DayStartTime,
		FlowMinerPrevTotal: s.FlowMinerPrevTotal,
		FlowMiner:          make(map[common.Address]map[common.Hash]*FlowMinerReport),
		FlowMinerPrev:      make(map[common.Address]map[common.Hash]*FlowMinerReport),
		FlowMinerCache:     nil,
		FlowMinerPrevCache: nil,
	}
	for who, item := range s.FlowMiner {
		clone.FlowMiner[who] = make(map[common.Hash]*FlowMinerReport)
		for chain, report := range item {
			clone.FlowMiner[who][chain] = report.copy()
		}
	}
	for who, item := range s.FlowMinerPrev {
		clone.FlowMinerPrev[who] = make(map[common.Hash]*FlowMinerReport)
		for chain, report := range item {
			clone.FlowMinerPrev[who][chain] = report.copy()
		}
	}
	clone.FlowMinerCache = make([]string, len(s.FlowMinerCache))
	copy(clone.FlowMinerCache, s.FlowMinerCache)
	clone.FlowMinerPrevCache = make([]string, len(s.FlowMinerPrevCache))
	copy(clone.FlowMinerPrevCache, s.FlowMinerPrevCache)
	return clone
}

func (s *FlowMinerSnap) updateFlowReport(rewardBlock uint64, blockPerDay uint64, flowReport []MinerFlowReportRecord, headerNumber *big.Int) {
	for _, items := range flowReport {
		chain := items.ChainHash
		for _, item := range items.ReportContent {
			//if items.ReportTime < s.DayStartTime {
			//	if rewardBlock > headerNumber.Uint64()%blockPerDay {
			//		if _, ok := s.FlowMinerPrev[item.Target]; !ok {
			//			s.FlowMinerPrev[item.Target] = make(map[common.Hash]*FlowMinerReport)
			//		}
			//		s.FlowMinerPrev[item.Target][chain] = &FlowMinerReport{
			//			Target:       item.Target,
			//			Hash:         chain,
			//			ReportNumber: item.ReportNumber,
			//			FlowValue1:   item.FlowValue1,
			//			FlowValue2:   item.FlowValue2,
			//		}
			//	}
			//} else {
			if _, ok := s.FlowMiner[item.Target]; !ok {
				s.FlowMiner[item.Target] = make(map[common.Hash]*FlowMinerReport)
			}
			if _, ok := s.FlowMiner[item.Target][chain]; !ok {
				s.FlowMiner[item.Target][chain] = &FlowMinerReport{
					Target:       item.Target,
					Hash:         chain,
					ReportNumber: item.ReportNumber,
					FlowValue1:   item.FlowValue1,
					FlowValue2:   item.FlowValue2,
				}
			}else {
				s.FlowMiner[item.Target][chain].ReportNumber += item.ReportNumber
				s.FlowMiner[item.Target][chain].FlowValue1 += item.FlowValue1
				s.FlowMiner[item.Target][chain].FlowValue2 += item.FlowValue2
			}
		}
		//}
	}
}

func (s *FlowMinerSnap) updateFlowMinerDaily(blockPerDay uint64, header *types.Header) {
	if 0 == header.Number.Uint64()%blockPerDay && 0 != header.Number.Uint64() {
		s.DayStartTime = header.Time
		s.FlowMinerPrev = make(map[common.Address]map[common.Hash]*FlowMinerReport)
		for address, item := range s.FlowMiner {
			s.FlowMinerPrev[address] = make(map[common.Hash]*FlowMinerReport)
			for chain, report := range item {
				s.FlowMinerPrev[address][chain] = &FlowMinerReport{
					Target:       report.Target,
					Hash:         report.Hash,
					ReportNumber: report.ReportNumber,
					FlowValue1:   report.FlowValue1,
					FlowValue2:   report.FlowValue2,
				}
			}
		}
		s.FlowMinerPrevCache = make([]string, len(s.FlowMinerCache))
		copy(s.FlowMinerPrevCache, s.FlowMinerCache)
		s.FlowMiner = make(map[common.Address]map[common.Hash]*FlowMinerReport)
		s.FlowMinerCache = []string{}
	}
}

func (s *FlowMinerSnap) cleanPrevFlow() {
	s.FlowMinerPrev = make(map[common.Address]map[common.Hash]*FlowMinerReport)
	s.FlowMinerPrevCache = []string{}
	s.FlowMinerPrevTotal = 0
}

func (s *FlowMinerSnap) setFlowPrevTotal(total uint64) {
	s.FlowMinerPrevTotal = total
}

func (s *FlowMinerSnap) accumulateFlows(db ethdb.Database) map[common.Address]*FlowMinerReport {
	flowcensus := make(map[common.Address]*FlowMinerReport)
	for minerAddress, item := range s.FlowMinerPrev {
		for _, bandwidth := range item {
			if _, ok := flowcensus[minerAddress]; !ok {
				flowcensus[minerAddress] = &FlowMinerReport{
					ReportNumber: bandwidth.ReportNumber,
					FlowValue1:   bandwidth.FlowValue1,
					FlowValue2:   bandwidth.FlowValue2,
				}
			} else {
				flowcensus[minerAddress].ReportNumber += bandwidth.ReportNumber
				flowcensus[minerAddress].FlowValue1 += bandwidth.FlowValue1
				flowcensus[minerAddress].FlowValue2 += bandwidth.FlowValue2
			}
		}
	}
	for _, key := range s.FlowMinerPrevCache {
		flows, err := s.load(db, key)
		if err != nil {
			log.Warn("accumulateFlows load cache error", "key", key, "err", err)
			continue
		}
		for _, flow := range flows {
			target := flow.Target
			if _, ok := flowcensus[target]; !ok {
				flowcensus[target] = &FlowMinerReport{
					ReportNumber: flow.ReportNumber,
					FlowValue1:   flow.FlowValue1,
					FlowValue2:   flow.FlowValue2,
				}
			} else {
				flowcensus[target].ReportNumber += flow.ReportNumber
				flowcensus[target].FlowValue1 += flow.FlowValue1
				flowcensus[target].FlowValue2 += flow.FlowValue2
			}
		}
	}
	return flowcensus
}

func (s *FlowMinerSnap) load(db ethdb.Database, key string) ([]*FlowMinerReport, error) {
	items := []*FlowMinerReport{}
	blob, err := db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	int := bytes.NewBuffer(blob)
	err = rlp.Decode(int, &items)
	if err != nil {
		return nil, err
	}
	log.Info("LockProfitSnap load", "key", key, "size", len(items))
	return items, nil
}

func (s *FlowMinerSnap) store(db ethdb.Database, number uint64) error {
	items := []*FlowMinerReport{}
	for _, flows := range s.FlowMiner {
		for _, flow := range flows {
			items = append(items, flow)
		}
	}
	if len(items) == 0 {
		return nil
	}
	err, buf := FlowMinerReportEncodeRlp(items)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("flow-%d", number)
	err = db.Put([]byte(key), buf)
	if err != nil {
		return err
	}
	s.FlowMinerCache = append(s.FlowMinerCache, key)
	s.FlowMiner = make(map[common.Address]map[common.Hash]*FlowMinerReport)
	log.Info("FlowMinerSnap store", "key", key, "len", len(items))
	return nil
}

func FlowMinerReportEncodeRlp(items []*FlowMinerReport) (error, []byte) {
	out := bytes.NewBuffer(make([]byte, 0, 255))
	err := rlp.Encode(out, items)
	if err != nil {
		return err, nil
	}
	return nil, out.Bytes()
}
