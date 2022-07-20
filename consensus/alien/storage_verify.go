package alien

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"github.com/UltronGlow/UltronGlow-Origin/common"
	"github.com/UltronGlow/UltronGlow-Origin/log"
	"math/big"
	"strconv"
	"strings"
)

/*

	block
nonce, blockhash
	pocstr
	roothash
*/
func verifyPocString(block, nonce, blockhash, pocstr, roothash string, deviceAddr string) bool {
	poc := strings.Split(pocstr, ",")
	if len(poc) < 10 {
		log.Warn("verifyPocString", "invalide poc string format")
		return false
	}

	sampleNumberpos := 3
	blocknumberpos := 5
	b0pos := 6
	b1pos := 7
	bnpos := 8

	//
	if !verifyB0(block, nonce, blockhash, poc[b0pos], deviceAddr) {
		log.Warn("verifyPocString", "verify b0 failed")
		return false
	}

	if !verifyBn(poc[sampleNumberpos], poc[b0pos], poc[bnpos], poc[b1pos]) {
		log.Warn("verifyPocString", "verify bn failed")
		return false
	}

	n, _ := strconv.ParseUint(poc[sampleNumberpos], 10, 64)
	if !verifySamplePos(n, poc[0], poc[1], poc[2], poc[blocknumberpos]) {
		log.Warn("verifyPocString", "verify samplenumber failed")
		return false
	}

	if n&1 != 0 {
		return verifyPoc(poc[9:], roothash, n)
	} else {
		return verifyPoc(poc[10:], roothash, n)
	}
}

func verifyStoragePoc(pocstr, roothash string, nonce uint64) bool {
	poc := strings.Split(pocstr, ",")
	if len(poc) < 10 {
		log.Warn("verifyStoragePoc", "invalide poc string format")
		return false
	}
	if poc[1] != strconv.FormatUint(nonce, 10) {
		log.Warn("verifyStoragePoc", "invalide nonce")
		return false
	}

	sampleNumberpos := 3
	blocknumberpos := 5
	b0pos := 6
	b1pos := 7
	bnpos := 8

	if !verifyBn(poc[sampleNumberpos], poc[b0pos], poc[bnpos], poc[b1pos]) {
		log.Warn("verifyPocString", "verify bn failed")
		return false
	}

	n, _ := strconv.ParseUint(poc[sampleNumberpos], 10, 64)
	if !verifySamplePos(n, poc[0], poc[1], poc[2], poc[blocknumberpos]) {
		log.Warn("verifyPocString", "verify samplenumber failed")
		return false
	}

	if n&1 != 0 {

		return verifyPoc(poc[9:], roothash, n)
	} else {

		return verifyPoc(poc[10:], roothash, n)
	}
}
func verifyPoc(pocstr []string, roothash string, r uint64) bool {
	var (
		hash  string
		round int
	)

	if len(pocstr)&1 != 1 {
		log.Warn("verifyPoc", "invalide poc data, len:", len(pocstr))
		return false
	}

	var hashpos int
	for i := 0; i < len(pocstr); i += 2 {
		if i+1 >= len(pocstr) {
			break
		}
		if round > 0 && pocstr[i+hashpos] != hash {
			return false
		}

		if round&1 != 1 {
			hash = Hash(pocstr[i], pocstr[i+1], "")
			log.Debug("verifyPoc", "round", round+1, "hash:", pocstr[i], pocstr[i+1], "=>", hash)
		} else {
			hash = Acc(pocstr[i], pocstr[i+1], "")
			log.Debug("verifyPoc", "round", round+1, "acc: ", pocstr[i], pocstr[i+1], "=>", hash)
		}
		r = r / 2
		hashpos = int(r & 1)
		//fmt.Println("cal hash:", hash, "pos", hashpos)
		round++
	}
	log.Warn("verifyPoc", "root hash:", hash, "roothash", roothash, "pocstr[len(pocstr)-1]", pocstr[len(pocstr)-1], "common.HexToHash(hash)", common.HexToHash(hash))

	if hash == pocstr[len(pocstr)-1] && common.HexToHash(hash) == common.HexToHash(roothash) {
		return true
	}
	return false
}

func verifyB0(block, nonce, blockhash, b0hash string, deviceAddr string) bool {
	b0 := Sha1([]byte(block + nonce + blockhash + deviceAddr))
	return b0 == b0hash
}

func verifyBn(blockpos, b0, bn, bn1 string) bool {
	n, err := strconv.ParseUint(blockpos, 10, 64)
	if err != nil {
		log.Warn("verifyPoc", "parse blockpos failed:", err)
		return false
	}
	if n&1 == 0 {
		return bn == Hash(b0, bn1, blockpos)
	}
	return bn == Acc(b0, bn1, blockpos)
}

func verifySamplePos(n uint64, block, nonce, blockhash, fileblocknumber string) bool {
	if n != getSamplePos(block, nonce, blockhash, fileblocknumber) {
		log.Warn("verifySamplePos", "verify simble number failed")
		return false
	}
	return true
}

func getSamplePos(block, nonce, blockhash, fileblocknumber string) uint64 {
	b1 := Sha1([]byte(block + nonce + blockhash))
	pos, _ := new(big.Int).SetString(fileblocknumber, 10)
	n := new(big.Int).Mod(hash2bigint(b1), pos).Uint64()
	return n
}

////////////////////////////////////////////////////////////////////////////////////////////////
func Hash(b1, b2, pos string) string {
	bb1, _ := hex.DecodeString(b1)
	bb2, _ := hex.DecodeString(b2)

	bb1 = append(bb1, bb2...)
	if pos != "" {
		n, err := strconv.ParseUint(pos, 10, 64)
		if err != nil {
			return ""
		}
		nbytes := UInt64ToByteLittleEndian(n)
		bb1 = append(bb1, nbytes...)
	}
	return Sha1(bb1)
}

func Sha1(orgin []byte) string {
	m := sha1.New()
	m.Write(orgin)
	return hex.EncodeToString(m.Sum(nil))
}

func Acc(b1, b2, n string) string {
	block1, err := hex.DecodeString(b1)
	if err != nil {
		return ""
	}

	block2, err := hex.DecodeString(b2)
	if err != nil {
		return ""
	}

	var N uint64
	if n != "" {
		if N, err = strconv.ParseUint(n, 10, 64); err != nil {
			return ""
		}
	} else {
		N = uint64(0)
	}

	accblock := make([]byte, 20)
	copy(accblock[:8], UInt64ToByteLittleEndian(ByteToUint64LittleEndian(block1[:8])+ByteToUint64LittleEndian(block2[:8])+N))
	copy(accblock[8:16], UInt64ToByteLittleEndian(ByteToUint64LittleEndian(block1[8:16])+ByteToUint64LittleEndian(block2[8:16])+N))
	accblock[16] = block1[16] + block2[16]
	accblock[17] = block1[17] + block2[17]
	accblock[18] = block1[18] + block2[18]
	accblock[19] = block1[19] + block2[19]
	return hex.EncodeToString(accblock)
}

func hash2bigint(hexstr string) *big.Int {
	h, _ := hex.DecodeString(hexstr)
	bi := new(big.Int).SetBytes(h)
	return bi
}

func ByteToUint64LittleEndian(b []byte) uint64 {
	buf := bytes.NewBuffer(b)
	var i uint64
	binary.Read(buf, binary.LittleEndian, &i)
	return i
}

func UInt64ToByteLittleEndian(i uint64) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.LittleEndian, i)
	return buf.Bytes()
}
