package alien

import (
	"github.com/UltronGlow/UltronGlow-Origin/common"
	"github.com/UltronGlow/UltronGlow-Origin/log"
	"math/big"
	"strconv"
	"strings"
)

func verifyPocStringV1(block, nonce, blockhash, pocstr, roothash, deviceAddr string) bool {
	poc := strings.Split(pocstr, ",")
	if len(poc) < 12 {
		log.Warn("verifyStoragePoc", "invalide poc string format")
		return false
	}
	if poc[0] != "v1" {
		log.Warn("verifyStoragePocV1", "invalide version tag")
		return false
	}

	sampleNumberpos := 4
	blocknumberpos := 6
	blockRangePos := 7
	b0pos := 8
	b1pos := 9
	bnpos := 10

	if !verifyB0(block, nonce, blockhash,poc[b0pos], deviceAddr) {
		log.Warn("verifyPocString1", "verify b0 failed",block,"nonce",nonce,"blockhash",blockhash,"deviceAddr",deviceAddr,"poc[b0pos]",poc[b0pos])
		return false
	}

	if !verifyBn(poc[sampleNumberpos], poc[b0pos], poc[bnpos], poc[b1pos]) {
		log.Warn("verifyPocString", "verify bn failed")
		return false
	}

	ranges := strings.Split(poc[blockRangePos], "-")
	if len(ranges) != 2 {
		log.Warn("verifyPocStringV1", "invalide poc data")
		return false
	}
	start, err := strconv.ParseUint(ranges[0], 10, 64)
	if err != nil {
		log.Warn("verifyPocStringV1", "invalide poc data")
		return false
	}
	end, err := strconv.ParseUint(ranges[1], 10, 64)
	if err != nil {
		log.Warn("verifyPocStringV1", "invalide poc data")
		return false
	}
	n, err := strconv.ParseUint(poc[sampleNumberpos], 10, 64)
	if err != nil {
		log.Warn("verifyPocStringV1", "invalide poc data")
		return false
	}
	if !verifySamplePosV1(n, poc[1], poc[2], poc[3], poc[blocknumberpos], start, end) {
		log.Warn("verifyPocString", "verify samplenumber failed")
		return false
	}

	if n&1 != 0 {
		h1 := Hash(poc[b1pos], poc[bnpos], "")
		return verifyPocV1(poc[11:], h1, roothash, start, n)
	} else {
		h1 := Hash(poc[bnpos], poc[bnpos+1], "")
		return verifyPocV1(poc[12:], h1, roothash, start, n)
	}
}

func verifyStoragePocV1(pocstr, roothash string, nonce uint64) bool {
	poc := strings.Split(pocstr, ",")
	if len(poc) < 12 {
		log.Warn("verifyStoragePocV1", "invalide v1 poc string format","len(poc)",len(poc))
		return false
	}
	if poc[0] != "v1" {
		log.Warn("verifyStoragePocV1", "invalide version tag")
		return false
	}
	if poc[2] != strconv.FormatUint(nonce, 10) {
		log.Warn("verifyStoragePocV1", "invalide nonce")
		return false
	}

	sampleNumberpos := 4
	blocknumberpos := 6
	blockRangePos := 7
	b0pos := 8
	b1pos := 9
	bnpos := 10

	if !verifyBn(poc[sampleNumberpos], poc[b0pos], poc[bnpos], poc[b1pos]) {
		log.Warn("verifyPocStringV1", "verify bn failed")
		return false
	}

	ranges := strings.Split(poc[blockRangePos], "-")
	if len(ranges) != 2 {
		log.Warn("verifyPocStringV1", "invalide poc data")
		return false
	}
	start, err := strconv.ParseUint(ranges[0], 10, 64)
	if err != nil {
		log.Warn("verifyPocStringV1", "invalide poc data")
		return false
	}
	end, err := strconv.ParseUint(ranges[1], 10, 64)
	if err != nil {
		log.Warn("verifyPocStringV1", "invalide poc data")
		return false
	}
	n, _ := strconv.ParseUint(poc[sampleNumberpos], 10, 64)
	if err != nil {
		log.Warn("verifyPocStringV1", "invalide poc data")
		return false
	}
	if !verifySamplePosV1(n, poc[1], poc[2], poc[3], poc[blocknumberpos], start, end) {
		log.Warn("verifyPocStringV1", "verify samplenumber failed")
		return false
	}

	if n&1 != 0 {
		h1 := Hash(poc[b1pos], poc[bnpos], "")
		return verifyPocV1(poc[11:], h1, roothash, start, n)
	} else {
		h1 := Hash(poc[bnpos], poc[bnpos+1], "")
		return verifyPocV1(poc[12:], h1, roothash, start, n)
	}
}

func verifyPocV1(pocstr []string, h1, roothash string, start, r uint64) bool {
	var (
		hash  string
		round int
		hashpos int
	)

	r = r / 2
	hashpos = int(r & 1)
	hash = h1
	for i := 0; i < len(pocstr); i++ {
		if i+1 >= len(pocstr) {
			break
		}
		if round&1 != 1 {
			if hashpos == 0 {
				hash = Acc(hash, pocstr[i], "")
			} else {
				hash = Acc(pocstr[i], hash, "")
			}
		} else {
			if hashpos == 0 {
				hash = Hash(hash, pocstr[i], "")
			} else {
				hash = Hash(pocstr[i], hash, "")
			}
		}

		if round == 18 {
			rn := start / (1024 * 1024)
			r = r/2 - rn
		} else {
			r = r / 2
		}
		hashpos = int(r & 1)
		round++
	}
	if hash == pocstr[len(pocstr)-1] && common.HexToHash(hash) == common.HexToHash(roothash) {
		return true
	}
	log.Warn("verifyPocV1", "root hash:", hash, "roothash", roothash, "pocstr[len(pocstr)-1]", pocstr[len(pocstr)-1], "common.HexToHash(hash)", common.HexToHash(hash))
	return false
}

func verifySamplePosV1(n uint64, block, nonce, blockhash, fileblocknumber string, start, end uint64) bool {
	calPos := getSamplePosV1(block, nonce, blockhash, fileblocknumber, start, end)
	calPos += start
	if n != calPos {
		log.Warn("verifySamplePos", "verify simble number failed")
		return false
	}
	return true
}

func getSamplePosV1(block, nonce, blockhash, fileblocknumber string, start, end uint64) uint64 {
	b1 := Sha1([]byte(block + nonce + blockhash))
	pos := new(big.Int).SetUint64(end - start + 1)
	n := new(big.Int).Mod(hash2bigint(b1), pos).Uint64()
	if n == 0 {
		n, _ = strconv.ParseUint(fileblocknumber, 10, 64)
	}
	return n
}
