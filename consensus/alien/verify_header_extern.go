package alien

import (
	"errors"
	"fmt"
	"github.com/UltronGlow/UltronGlow-Origin/common"
	"github.com/UltronGlow/UltronGlow-Origin/consensus"
	"math/big"
	"reflect"
	"strconv"
)

const (
	lr_s   = "LockReward"
	en_s   = "ExchangeNFC"
	db_s   = "DeviceBind"
	cpl_s  = "CandidatePledge"
	cp_s   = "CandidatePunish"
	ms_s   = "MinerStake"
	cb_s   = "ClaimedBandwidth"
	bp_s   = "BandwidthPunish"
	cd_s   = "ConfigDeposit"
	ci_s   = "ConfigISPQOS"
	lp_s   = "LockParameters"
	ma_s   = "ManagerAddress"
	gp_s   = "GrantProfit"
	fr_s   = "FlowReport"
	mfrt_s = "MinerFlowReportItem"

	sp_s      = "StoragePledge"
	spe_s     = "StoragePledgeExit"
	sr_s      = "LeaseRequest"
	esrt_s    = "ExchangeSRT"
	esrpg_s   = "LeasePledge"
	esrrn_s   = "LeaseRenewal"
	esrrnpg_s = "LeaseRenewalPledge"
	esrc_s    = "LeaseRescind"
	esrd_s    = "StorageRecoveryData"
	espr_s    = "StorageProofRecord"
	esep_s    = "StorageExchangePrice"
	esbp_s    = "StorageBwPay"
)

func verifyHeaderExtern(currentExtra *HeaderExtra, verifyExtra *HeaderExtra) error {

	//ExchangeNFC               []ExchangeNFCRecord
	err := verifyExchangeNFC(currentExtra.ExchangeNFC, verifyExtra.ExchangeNFC)
	if err != nil {
		return err
	}
	//LockReward                []LockRewardRecord
	err = verifyLockReward(currentExtra.LockReward, verifyExtra.LockReward)
	if err != nil {
		return err
	}

	//DeviceBind                []DeviceBindRecord
	err = verifyDeviceBind(currentExtra.DeviceBind, verifyExtra.DeviceBind)
	if err != nil {
		return err
	}

	//CandidatePledge           []CandidatePledgeRecord
	err = verifyCandidatePledge(currentExtra.CandidatePledge, verifyExtra.CandidatePledge)
	if err != nil {
		return err
	}
	//CandidatePunish           []CandidatePunishRecord
	err = verifyCandidatePunish(currentExtra.CandidatePunish, verifyExtra.CandidatePunish)
	if err != nil {
		return err
	}
	//MinerStake                []MinerStakeRecord
	err = verifyMinerStake(currentExtra.MinerStake, verifyExtra.MinerStake)
	if err != nil {
		return err
	}

	//CandidateExit             []common.Address
	err = verifyExit(currentExtra.CandidateExit, verifyExtra.CandidateExit, "CandidateExit")
	if err != nil {
		return err
	}

	//ClaimedBandwidth          []ClaimedBandwidthRecord
	err = verifyClaimedBandwidth(currentExtra.ClaimedBandwidth, verifyExtra.ClaimedBandwidth)
	if err != nil {
		return err
	}

	//FlowMinerExit             []common.Address
	err = verifyExit(currentExtra.FlowMinerExit, verifyExtra.FlowMinerExit, "FlowMinerExit")
	if err != nil {
		return err
	}

	//BandwidthPunish           []BandwidthPunishRecord
	err = verifyBandwidthPunish(currentExtra.BandwidthPunish, verifyExtra.BandwidthPunish)
	if err != nil {
		return err
	}

	//ConfigExchRate            uint32
	err = verifyUint32Config(currentExtra.ConfigExchRate, verifyExtra.ConfigExchRate, "ConfigExchRate")
	if err != nil {
		return err
	}
	//ConfigOffLine             uint32
	err = verifyUint32Config(currentExtra.ConfigOffLine, verifyExtra.ConfigOffLine, "ConfigOffLine")
	if err != nil {
		return err
	}

	//ConfigDeposit             []ConfigDepositRecord
	err = verifyConfigDeposit(currentExtra.ConfigDeposit, verifyExtra.ConfigDeposit)
	if err != nil {
		return err
	}

	//ConfigISPQOS              []ISPQOSRecord
	err = verifyConfigISPQOS(currentExtra.ConfigISPQOS, verifyExtra.ConfigISPQOS)
	if err != nil {
		return err
	}

	//LockParameters            []LockParameterRecord
	err = verifyLockParameters(currentExtra.LockParameters, verifyExtra.LockParameters)
	if err != nil {
		return err
	}

	//ManagerAddress            []ManagerAddressRecord
	err = verifyManagerAddress(currentExtra.ManagerAddress, verifyExtra.ManagerAddress)
	if err != nil {
		return err
	}
	//FlowHarvest               *big.Int
	err = verifyFlowHarvest(currentExtra.FlowHarvest, verifyExtra.FlowHarvest)
	if err != nil {
		return err
	}
	//GrantProfit               []consensus.GrantProfitRecord
	err = verifyGrantProfit(currentExtra.GrantProfit, verifyExtra.GrantProfit)
	if err != nil {
		return err
	}

	//FlowReport                []MinerFlowReportRecord
	err = verifyFlowReport(currentExtra.FlowReport, verifyExtra.FlowReport)
	if err != nil {
		return err
	}
	//StoragePledge
	err = verifyStoragePledge(currentExtra.StoragePledge, verifyExtra.StoragePledge)
	if err != nil {
		return err
	}
	//StoragePledgeExit
	err = verifyStoragePledgeExit(currentExtra.StoragePledgeExit, verifyExtra.StoragePledgeExit)
	if err != nil {
		return err
	}

	//LeaseRequest
	err = verifyLeaseRequest(currentExtra.LeaseRequest, verifyExtra.LeaseRequest)
	if err != nil {
		return err
	}
	//	esrt_s="ExchangeSRT"
	err = verifyExchangeSRT(currentExtra.ExchangeSRT, verifyExtra.ExchangeSRT)
	if err != nil {
		return err
	}
	//esrpg_s="LeasePledge"
	err = verifyLeasePledge(currentExtra.LeasePledge, verifyExtra.LeasePledge)
	if err != nil {
		return err
	}
	//esrrn_s="LeaseRenewal"
	err = verifyLeaseRenewal(currentExtra.LeaseRenewal, verifyExtra.LeaseRenewal)
	if err != nil {
		return err
	}
	//esrrnpg_s="LeaseRenewalPledge"
	err = verifyLeaseRenewalPledge(currentExtra.LeaseRenewalPledge, verifyExtra.LeaseRenewalPledge)
	if err != nil {
		return err
	}
	//esrc_s="LeaseRescind"
	err = verifyLeaseRescind(currentExtra.LeaseRescind, verifyExtra.LeaseRescind)
	if err != nil {
		return err
	}
	//esrd_s="StorageRecoveryData"
	err = verifyStorageRecoveryData(currentExtra.StorageRecoveryData, verifyExtra.StorageRecoveryData)
	if err != nil {
		return err
	}
	//espr_s="StorageProofRecord"
	err = verifyStorageProofRecord(currentExtra.StorageProofRecord, verifyExtra.StorageProofRecord)
	if err != nil {
		return err
	}
	//esep_s="StorageExchangePrice"
	err = verifyStorageExchangePrice(currentExtra.StorageExchangePrice, verifyExtra.StorageExchangePrice)
	if err != nil {
		return err
	}
	if currentExtra.StorageDataRoot != verifyExtra.StorageDataRoot {
		return errors.New("Compare StorageDataRoot, current is " + currentExtra.StorageDataRoot.String() + ". but verify is " + verifyExtra.StorageDataRoot.String())
	}
	//esr_s="ExtraStateRoot"
	if currentExtra.ExtraStateRoot != verifyExtra.ExtraStateRoot {
		return errors.New("Compare ExtraStateRoot, current is " + currentExtra.ExtraStateRoot.String() + ". but verify is " + verifyExtra.ExtraStateRoot.String())
	}
	//elar_s="LockAccountsRoot"
	if currentExtra.LockAccountsRoot != verifyExtra.LockAccountsRoot {
		return errors.New("Compare LockAccountsRoot, current is " + currentExtra.LockAccountsRoot.String() + ". but verify is " + verifyExtra.LockAccountsRoot.String())
	}
	//SRTDataRoot
	if currentExtra.SRTDataRoot != verifyExtra.SRTDataRoot {
		return errors.New("Compare SRTDataRoot, current is " + currentExtra.SRTDataRoot.String() + ". but verify is " + verifyExtra.SRTDataRoot.String())
	}
	//esbp_s    = "StorageBwPay"
	err = verifyStorageBwPay(currentExtra.StorageBwPay, verifyExtra.StorageBwPay)
	if err != nil {
		return err
	}
	return nil
}

func verifyUint32Config(current uint32, verify uint32, name string) error {
	if current != verify {
		s := strconv.FormatUint(uint64(current), 10)
		s2 := strconv.FormatUint(uint64(verify), 10)
		return errors.New("Compare " + name + ", current is " + s + ". but verify is " + s2)
	}
	return nil
}

func verifyLockReward(current []LockRewardRecord, verify []LockRewardRecord) error {
	arrLen, err := verifyArrayBasic(lr_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareLockReward(current, verify)
	if err != nil {
		return err
	}
	err = compareLockReward(verify, current)
	if err != nil {
		return err
	}
	return nil
}
func compareLockReward(a []LockRewardRecord, b []LockRewardRecord) error {
	b2 := make([]LockRewardRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Amount.Cmp(v.Amount) == 0 && c.FlowValue1 == v.FlowValue1 && c.FlowValue2 == v.FlowValue2 && c.IsReward == v.IsReward && c.Target == v.Target {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(lr_s, c)
		}
	}
	return nil
}

func verifyExchangeNFC(current []ExchangeNFCRecord, verify []ExchangeNFCRecord) error {
	arrLen, err := verifyArrayBasic(en_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareExchangeNFC(current, verify)
	if err != nil {
		return err
	}
	err = compareExchangeNFC(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareExchangeNFC(a []ExchangeNFCRecord, b []ExchangeNFCRecord) error {
	b2 := make([]ExchangeNFCRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Target == v.Target && c.Amount.Cmp(v.Amount) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(en_s, c)
		}
	}
	return nil
}

func verifyDeviceBind(current []DeviceBindRecord, verify []DeviceBindRecord) error {
	arrLen, err := verifyArrayBasic(db_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareDeviceBind(current, verify)
	if err != nil {
		return err
	}
	err = compareDeviceBind(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareDeviceBind(a []DeviceBindRecord, b []DeviceBindRecord) error {
	b2 := make([]DeviceBindRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Device == v.Device && c.Revenue == v.Revenue && c.Contract == v.Contract && c.MultiSign == v.MultiSign && c.Type == v.Type && c.Bind == v.Bind {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(db_s, c)
		}
	}
	return nil
}

func verifyCandidatePledge(current []CandidatePledgeRecord, verify []CandidatePledgeRecord) error {
	arrLen, err := verifyArrayBasic(cpl_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareCandidatePledge(current, verify)
	if err != nil {
		return err
	}
	err = compareCandidatePledge(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareCandidatePledge(a []CandidatePledgeRecord, b []CandidatePledgeRecord) error {
	b2 := make([]CandidatePledgeRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Target == v.Target && c.Amount.Cmp(v.Amount) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(cpl_s, c)
		}
	}
	return nil
}

func verifyCandidatePunish(current []CandidatePunishRecord, verify []CandidatePunishRecord) error {
	arrLen, err := verifyArrayBasic(cp_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareCandidatePunish(current, verify)
	if err != nil {
		return err
	}
	err = compareCandidatePunish(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareCandidatePunish(a []CandidatePunishRecord, b []CandidatePunishRecord) error {
	b2 := make([]CandidatePunishRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Target == v.Target && c.Amount.Cmp(v.Amount) == 0 && c.Credit == v.Credit {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(cp_s, c)
		}
	}
	return nil
}

func verifyMinerStake(current []MinerStakeRecord, verify []MinerStakeRecord) error {
	arrLen, err := verifyArrayBasic(ms_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareMinerStake(current, verify)
	if err != nil {
		return err
	}
	err = compareMinerStake(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareMinerStake(a []MinerStakeRecord, b []MinerStakeRecord) error {
	b2 := make([]MinerStakeRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Target == v.Target && c.Stake.Cmp(v.Stake) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(ms_s, c)
		}
	}
	return nil
}

func verifyExit(current []common.Address, verify []common.Address, name string) error {
	arrLen, err := verifyArrayBasic(name, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareExit(current, verify, name)
	if err != nil {
		return err
	}
	err = compareExit(verify, current, name)
	if err != nil {
		return err
	}
	return nil
}

func compareExit(a []common.Address, b []common.Address, name string) error {
	b2 := make([]common.Address, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c == v {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(name, c)
		}
	}
	return nil
}

func verifyClaimedBandwidth(current []ClaimedBandwidthRecord, verify []ClaimedBandwidthRecord) error {
	arrLen, err := verifyArrayBasic(cb_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareClaimedBandwidth(current, verify)
	if err != nil {
		return err
	}
	err = compareClaimedBandwidth(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareClaimedBandwidth(a []ClaimedBandwidthRecord, b []ClaimedBandwidthRecord) error {
	b2 := make([]ClaimedBandwidthRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Target == v.Target && c.Amount.Cmp(v.Amount) == 0 && c.ISPQosID == v.ISPQosID && c.Bandwidth == v.Bandwidth {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(cb_s, c)
		}
	}
	return nil
}

func verifyBandwidthPunish(current []BandwidthPunishRecord, verify []BandwidthPunishRecord) error {
	arrLen, err := verifyArrayBasic(bp_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareBandwidthPunish(current, verify)
	if err != nil {
		return err
	}
	err = compareBandwidthPunish(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareBandwidthPunish(a []BandwidthPunishRecord, b []BandwidthPunishRecord) error {
	b2 := make([]BandwidthPunishRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Target == v.Target && c.WdthPnsh == v.WdthPnsh {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(bp_s, c)
		}
	}
	return nil
}

func verifyConfigDeposit(current []ConfigDepositRecord, verify []ConfigDepositRecord) error {
	arrLen, err := verifyArrayBasic(cd_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareConfigDeposit(current, verify)
	if err != nil {
		return err
	}
	err = compareConfigDeposit(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareConfigDeposit(a []ConfigDepositRecord, b []ConfigDepositRecord) error {
	b2 := make([]ConfigDepositRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Who == v.Who && c.Amount.Cmp(v.Amount) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(cd_s, c)
		}
	}
	return nil
}

func verifyConfigISPQOS(current []ISPQOSRecord, verify []ISPQOSRecord) error {
	arrLen, err := verifyArrayBasic(ci_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareConfigISPQOS(current, verify)
	if err != nil {
		return err
	}
	err = compareConfigISPQOS(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareConfigISPQOS(a []ISPQOSRecord, b []ISPQOSRecord) error {
	b2 := make([]ISPQOSRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.ISPID == v.ISPID && c.QOS == v.QOS {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(ci_s, c)
		}
	}
	return nil
}

func verifyLockParameters(current []LockParameterRecord, verify []LockParameterRecord) error {
	arrLen, err := verifyArrayBasic(lp_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareLockParameters(current, verify)
	if err != nil {
		return err
	}
	err = compareLockParameters(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareLockParameters(a []LockParameterRecord, b []LockParameterRecord) error {
	b2 := make([]LockParameterRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.LockPeriod == v.LockPeriod && c.RlsPeriod == v.RlsPeriod && c.Interval == v.Interval && c.Who == v.Who {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(lp_s, c)
		}
	}
	return nil
}

func verifyManagerAddress(current []ManagerAddressRecord, verify []ManagerAddressRecord) error {
	arrLen, err := verifyArrayBasic(ma_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareManagerAddress(current, verify)
	if err != nil {
		return err
	}
	err = compareManagerAddress(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareManagerAddress(a []ManagerAddressRecord, b []ManagerAddressRecord) error {
	b2 := make([]ManagerAddressRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Target == v.Target && c.Who == v.Who {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(ma_s, c)
		}
	}
	return nil
}

func verifyFlowHarvest(current *big.Int, verify *big.Int) error {
	fh_s := "FlowHarvest"
	if current == nil && verify == nil {
		return nil
	}
	if current == nil && verify != nil {
		return errorsMsg1(fh_s)
	}
	if current != nil && verify == nil {
		return errorsMsg2(fh_s)
	}
	if current != nil && verify != nil && current.Cmp(verify) != 0 {
		return errors.New("Compare " + fh_s + ", current is " + current.String() + ". but verify is " + verify.String())
	}
	return nil
}

func verifyGrantProfit(current []consensus.GrantProfitRecord, verify []consensus.GrantProfitRecord) error {
	arrLen, err := verifyArrayBasic(gp_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareGrantProfit(current, verify)
	if err != nil {
		return err
	}
	err = compareGrantProfit(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareGrantProfit(a []consensus.GrantProfitRecord, b []consensus.GrantProfitRecord) error {
	b2 := make([]consensus.GrantProfitRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Which == v.Which && c.MinerAddress == v.MinerAddress && c.BlockNumber == v.BlockNumber && c.Amount.Cmp(v.Amount) == 0 && c.RevenueAddress == v.RevenueAddress && c.RevenueContract == v.RevenueContract && c.MultiSignature == v.MultiSignature {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(gp_s, c)
		}
	}
	return nil
}

func verifyFlowReport(current []MinerFlowReportRecord, verify []MinerFlowReportRecord) error {
	arrLen, err := verifyArrayBasic(fr_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareFlowReport(current, verify)
	if err != nil {
		return err
	}
	err = compareFlowReport(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareFlowReport(a []MinerFlowReportRecord, b []MinerFlowReportRecord) error {
	b2 := make([]MinerFlowReportRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.ChainHash == v.ChainHash && c.ReportTime == v.ReportTime {
				if err := verifyMinerFlowReportItem(c.ReportContent, v.ReportContent); err == nil {
					find = true
					b2 = append(b2[:i], b2[i+1:]...)
					break
				}
			}
		}
		if !find {
			return errorsMsg4(fr_s, c)
		}
	}
	return nil
}

func verifyMinerFlowReportItem(current []MinerFlowReportItem, verify []MinerFlowReportItem) error {
	arrLen, err := verifyArrayBasic(mfrt_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareMinerFlowReportItem(current, verify)
	if err != nil {
		return err
	}
	err = compareMinerFlowReportItem(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareMinerFlowReportItem(a []MinerFlowReportItem, b []MinerFlowReportItem) error {
	b2 := make([]MinerFlowReportItem, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Target == v.Target && c.ReportNumber == v.ReportNumber && c.FlowValue1 == v.FlowValue1 && c.FlowValue2 == v.FlowValue2 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(mfrt_s, c)
		}
	}
	return nil
}

func errorsMsg1(name string) error {
	return errors.New("Compare " + name + " , current is nil. but verify is not nil")
}
func errorsMsg2(name string) error {
	return errors.New("Compare " + name + " , current is not nil. but verify is nil")
}
func errorsMsg3(name string, lenc int, lenv int) error {
	return errors.New(fmt.Sprintf("Compare "+name+", The array length is not equals. the current length is %d. the verify length is %d", lenc, lenv))
}
func errorsMsg4(name string, c interface{}) error {
	return errors.New(fmt.Sprintf("Compare "+name+", can't find %v in verify data", c))
}

func isNull(obj interface{}) bool {
	if obj == nil {
		return true
	}
	kind := reflect.TypeOf(obj).Kind()
	if reflect.Array == kind || reflect.Slice == kind {
		vc := reflect.ValueOf(obj)
		return vc.Len() == 0
	}
	return false
}

/**
 * compare current and verify, current and verify must be array
 * return (array length,error)
 */
func verifyArrayBasic(title string, current interface{}, verify interface{}) (int, error) {
	if current == nil {
		if verify == nil {
			return 0, nil
		}
		verifyLen := reflect.ValueOf(verify).Len()
		if verifyLen == 0 {
			return 0, nil
		}
		return 0, errorsMsg1(title)
	}
	currentLen := reflect.ValueOf(current).Len()
	if verify == nil {
		if currentLen == 0 {
			return 0, nil
		} else {
			return 0, errorsMsg2(title)
		}
	}
	verifyLen := reflect.ValueOf(verify).Len()
	if currentLen != verifyLen {
		return 0, errorsMsg3(title, currentLen, verifyLen)
	}
	return currentLen, nil
}

func verifyStoragePledge(current []SPledgeRecord, verify []SPledgeRecord) error {
	arrLen, err := verifyArrayBasic(sp_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareStoragePledge(current, verify)
	if err != nil {
		return err
	}
	err = compareStoragePledge(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareStoragePledge(a []SPledgeRecord, b []SPledgeRecord) error {
	b2 := make([]SPledgeRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.PledgeAddr == v.PledgeAddr && c.Address == v.Address && c.Price.Cmp(v.Price) == 0 && c.SpaceDeposit.Cmp(v.SpaceDeposit) == 0 && c.StorageCapacity.Cmp(v.StorageCapacity) == 0 && c.StorageSize.Cmp(v.StorageSize) == 0 && c.RootHash == v.RootHash && c.PledgeNumber.Cmp(v.PledgeNumber) == 0 && c.Bandwidth.Cmp(v.Bandwidth) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(sp_s, c)
		}
	}
	return nil
}

func verifyStoragePledgeExit(current []SPledgeExitRecord, verify []SPledgeExitRecord) error {
	arrLen, err := verifyArrayBasic(spe_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareStoragePledgeExit(current, verify)
	if err != nil {
		return err
	}
	err = compareStoragePledgeExit(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareStoragePledgeExit(a []SPledgeExitRecord, b []SPledgeExitRecord) error {
	b2 := make([]SPledgeExitRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Address == v.Address && c.PledgeStatus.Cmp(v.PledgeStatus) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(spe_s, c)
		}
	}
	return nil
}

func verifyLeaseRequest(current []LeaseRequestRecord, verify []LeaseRequestRecord) error {
	arrLen, err := verifyArrayBasic(sr_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareLeaseRequest(current, verify)
	if err != nil {
		return err
	}
	err = compareLeaseRequest(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareLeaseRequest(a []LeaseRequestRecord, b []LeaseRequestRecord) error {
	b2 := make([]LeaseRequestRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Tenant == v.Tenant && c.Address == v.Address && c.Capacity.Cmp(v.Capacity) == 0 && c.Duration.Cmp(v.Duration) == 0 && c.Price.Cmp(v.Price) == 0 && c.Hash == v.Hash {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(sr_s, c)
		}
	}
	return nil
}

func verifyExchangeSRT(current []ExchangeSRTRecord, verify []ExchangeSRTRecord) error {
	arrLen, err := verifyArrayBasic(esrt_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareExchangeSRT(current, verify)
	if err != nil {
		return err
	}
	err = compareExchangeSRT(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareExchangeSRT(a []ExchangeSRTRecord, b []ExchangeSRTRecord) error {
	b2 := make([]ExchangeSRTRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Target == v.Target && c.Amount.Cmp(v.Amount) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(esrt_s, c)
		}
	}
	return nil
}

func verifyLeasePledge(current []LeasePledgeRecord, verify []LeasePledgeRecord) error {
	arrLen, err := verifyArrayBasic(esrpg_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareLeasePledge(current, verify)
	if err != nil {
		return err
	}
	err = compareLeasePledge(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareLeasePledge(a []LeasePledgeRecord, b []LeasePledgeRecord) error {
	b2 := make([]LeasePledgeRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Address == v.Address && c.DepositAddress == v.DepositAddress && c.Hash == v.Hash && c.Capacity.Cmp(v.Capacity) == 0 && c.RootHash == v.RootHash && c.BurnSRTAmount.Cmp(v.BurnSRTAmount) == 0 && c.BurnAmount.Cmp(v.BurnAmount) == 0 && c.Duration.Cmp(v.Duration) == 0 && c.BurnSRTAddress == v.BurnSRTAddress && c.PledgeHash == v.PledgeHash && c.LeftCapacity.Cmp(v.LeftCapacity) == 0 && c.LeftRootHash == v.LeftRootHash {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(esrpg_s, c)
		}
	}
	return nil
}

func verifyLeaseRenewal(current []LeaseRenewalRecord, verify []LeaseRenewalRecord) error {
	arrLen, err := verifyArrayBasic(esrrn_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareLeaseRenewal(current, verify)
	if err != nil {
		return err
	}
	err = compareLeaseRenewal(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareLeaseRenewal(a []LeaseRenewalRecord, b []LeaseRenewalRecord) error {
	b2 := make([]LeaseRenewalRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Address == v.Address && c.Duration.Cmp(v.Duration) == 0 && c.Hash == v.Hash && c.Price.Cmp(v.Price) == 0 && c.Tenant == v.Tenant && c.NewHash == v.NewHash && c.Capacity.Cmp(v.Capacity) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(esrrn_s, c)
		}
	}
	return nil
}

func verifyLeaseRenewalPledge(current []LeaseRenewalPledgeRecord, verify []LeaseRenewalPledgeRecord) error {
	arrLen, err := verifyArrayBasic(esrrnpg_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareLeaseRenewalPledge(current, verify)
	if err != nil {
		return err
	}
	err = compareLeaseRenewalPledge(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareLeaseRenewalPledge(a []LeaseRenewalPledgeRecord, b []LeaseRenewalPledgeRecord) error {
	b2 := make([]LeaseRenewalPledgeRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Address == v.Address && c.Hash == v.Hash && c.Capacity.Cmp(v.Capacity) == 0 && c.RootHash == v.RootHash && c.BurnSRTAmount.Cmp(v.BurnSRTAmount) == 0 && c.BurnAmount.Cmp(v.BurnAmount) == 0 && c.Duration.Cmp(v.Duration) == 0 && c.BurnSRTAddress == v.BurnSRTAddress && c.PledgeHash == v.PledgeHash {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(esrrnpg_s, c)
		}
	}
	return nil
}

func verifyLeaseRescind(current []LeaseRescindRecord, verify []LeaseRescindRecord) error {
	arrLen, err := verifyArrayBasic(esrc_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareLeaseRescind(current, verify)
	if err != nil {
		return err
	}
	err = compareLeaseRescind(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareLeaseRescind(a []LeaseRescindRecord, b []LeaseRescindRecord) error {
	b2 := make([]LeaseRescindRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Address == v.Address && c.Hash == v.Hash {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(esrc_s, c)
		}
	}
	return nil
}

func verifyStorageRecoveryData(current []SPledgeRecoveryRecord, verify []SPledgeRecoveryRecord) error {
	arrLen, err := verifyArrayBasic(esrd_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareStorageRecoveryData(current, verify)
	if err != nil {
		return err
	}
	err = compareStorageRecoveryData(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareStorageRecoveryData(a []SPledgeRecoveryRecord, b []SPledgeRecoveryRecord) error {
	b2 := make([]SPledgeRecoveryRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Address == v.Address && compareLeaseHash(c.LeaseHash, v.LeaseHash) && compareLeaseHash(v.LeaseHash, c.LeaseHash) && c.SpaceCapacity.Cmp(v.SpaceCapacity) == 0 && c.RootHash == v.RootHash && c.ValidNumber.Cmp(v.ValidNumber) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(esrd_s, c)
		}
	}
	return nil
}

func compareLeaseHash(a []common.Hash, b []common.Hash) bool {
	b2 := make([]common.Hash, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c == v {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return false
		}
	}
	return true
}

func verifyStorageProofRecord(current []StorageProofRecord, verify []StorageProofRecord) error {
	arrLen, err := verifyArrayBasic(espr_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareStorageProofRecord(current, verify)
	if err != nil {
		return err
	}
	err = compareStorageProofRecord(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareStorageProofRecord(a []StorageProofRecord, b []StorageProofRecord) error {
	b2 := make([]StorageProofRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Address == v.Address && c.LeaseHash == v.LeaseHash && c.RootHash == v.RootHash && c.LastVerificationTime.Cmp(v.LastVerificationTime) == 0 && c.LastVerificationSuccessTime.Cmp(v.LastVerificationSuccessTime) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(espr_s, c)
		}
	}
	return nil
}

func verifyStorageExchangePrice(current []StorageExchangePriceRecord, verify []StorageExchangePriceRecord) error {
	arrLen, err := verifyArrayBasic(esep_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareStorageExchangePrice(current, verify)
	if err != nil {
		return err
	}
	err = compareStorageExchangePrice(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareStorageExchangePrice(a []StorageExchangePriceRecord, b []StorageExchangePriceRecord) error {
	b2 := make([]StorageExchangePriceRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Address == v.Address && c.Price.Cmp(v.Price) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(esep_s, c)
		}
	}
	return nil
}
func verifyStorageBwPay(current []StorageBwPayRecord, verify []StorageBwPayRecord) error {

	arrLen, err := verifyArrayBasic(esbp_s, current, verify)
	if err != nil {
		return err
	}
	if arrLen == 0 {
		return nil
	}
	err = compareStorageBwPay(current, verify)
	if err != nil {
		return err
	}
	err = compareStorageBwPay(verify, current)
	if err != nil {
		return err
	}
	return nil
}

func compareStorageBwPay(a []StorageBwPayRecord, b []StorageBwPayRecord) error {
	b2 := make([]StorageBwPayRecord, len(b))
	copy(b2, b)
	for _, c := range a {
		find := false
		for i, v := range b2 {
			if c.Address == v.Address && c.Amount.Cmp(v.Amount) == 0 {
				find = true
				b2 = append(b2[:i], b2[i+1:]...)
				break
			}
		}
		if !find {
			return errorsMsg4(esbp_s, c)
		}
	}
	return nil
}
