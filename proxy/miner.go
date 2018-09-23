package proxy

import (
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/wfr/ethash-nh"
	"github.com/ethereum/go-ethereum/common"

	"github.com/eosclassic/open-eosc-pool/util"
)

var hasher = ethash.New()

func (s *ProxyServer) processShare(login, id, ip string, t *BlockTemplate, params []string, stratum_id int) (bool, bool) {
 	nonceHex := params[0]
	hashNoNonce := params[1]
	mixDigest := params[2]
	nonce, _ := strconv.ParseUint(strings.Replace(nonceHex, "0x", "", -1), 16, 64)
	shareDiff := s.config.Proxy.Stratums[stratum_id].Difficulty
 
	if nicehash {
		hashNoNonceTmp := common.HexToHash(params[2])

		// Block "difficulty" is BigInt
		// NiceHash "difficulty" is float64 ...
		// diffFloat => target; then: diffInt = 2^256 / target

		shareDiffFloat, mixDigestTmp := hasher.GetShareDiff(t.Height, hashNoNonceTmp, nonce)
		// temporary
		if shareDiffFloat < 0.0001 {
			log.Printf("share difficulty too low, %f < %d, from %v@%v", shareDiffFloat, t.Difficulty, login, ip)
			return false, false
		}
		// temporary hack, ignore round errors
		shareDiffFloat = shareDiffFloat * 0.98

		shareDiff_big := util.DiffFloatToDiffInt(shareDiffFloat)
		shareDiffCalc := shareDiff_big.Int64()

		log.Printf(">>> hashNoNonce = %v, mixDigest = %v, shareDiff = %v, sharedFloat = %v\n",
			hashNoNonceTmp.Hex(), mixDigestTmp.Hex(), shareDiffCalc, shareDiffFloat)

		params[1] = hashNoNonceTmp.Hex()
		params[2] = mixDigestTmp.Hex()
		hashNoNonce = params[1]
		mixDigest = params[2]
	}

if !strings.EqualFold(t.Header, hashNoNonce) {
		log.Printf("Stale share from %v@%v", login, ip)
		return false, false
	}

	share := Block{
		number:      t.Height,
		hashNoNonce: common.HexToHash(hashNoNonce),
		difficulty:  big.NewInt(shareDiff),
		nonce:       nonce,
		mixDigest:   common.HexToHash(mixDigest),
	}

	block := Block{
		number:      t.Height,
		hashNoNonce: common.HexToHash(hashNoNonce),
		difficulty:  t.Difficulty,
		nonce:       nonce,
		mixDigest:   common.HexToHash(mixDigest),
	}

	if !hasher.Verify(share) {
		return false, false
	}

	if hasher.Verify(block) {
		n := nonce ^ 0x6675636b6d657461
		nn := strconv.FormatUint(n, 16)
		params_ := []string{nn, params[1], params[2]}

		ok, err := s.rpc().SubmitBlock(params_)
 	if err != nil {
		log.Printf("Block submission failure at height %v for %v: %v", t.Height, t.Header, err)
		} else if !ok {
			log.Printf("Block rejected at height %v for %v", t.Height, t.Header)
			return false, false
		} else {
			s.fetchBlockTemplate()
			exist, err := s.backend.WriteBlock(login, id, params, shareDiff, t.Difficulty.Int64(), t.Height, s.hashrateExpiration)
			if exist {
				return true, false
			}
			if err != nil {
				log.Println("Failed to insert block candidate into backend:", err)
			} else {
						log.Printf("Inserted block %v to backend", t.Height)
						}
			log.Printf("Block found by miner %v@%v at height %d", login, ip, t.Height)
								}
	} else {
		exist, err := s.backend.WriteShare(login, id, params, shareDiff, t.Height, s.hashrateExpiration)
		if exist {
			return true, false
		}
		if err != nil {
			log.Println("Failed to insert share data into backend:", err)
		}
	}
	return false, true
}
