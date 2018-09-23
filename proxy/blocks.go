package proxy

import (
	"log"
	"math/big"
	"strconv"
	//"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/eosclassic/open-eosc-pool/rpc"
	"github.com/eosclassic/open-eosc-pool/util"
)

const maxBacklog = 3

type heightDiffPair struct {
	diff   *big.Int
	height uint64
}

type BlockTemplate struct {
	sync.RWMutex
	Header               string
	Seed                 string
	Target               string
	Difficulty           *big.Int
	Height               uint64
	GetPendingBlockCache *rpc.GetBlockReplyPart
	nonces               map[string]bool
	headers              map[string]heightDiffPair
}

type Block struct {
	difficulty  *big.Int
	hashNoNonce common.Hash
	nonce       uint64
	mixDigest   common.Hash
	number      uint64
}

func (b Block) Difficulty() *big.Int     { return b.difficulty }
func (b Block) HashNoNonce() common.Hash { return b.hashNoNonce }
func (b Block) Nonce() uint64            { return b.nonce }
func (b Block) MixDigest() common.Hash   { return b.mixDigest }
func (b Block) NumberU64() uint64        { return b.number }

func (s *ProxyServer) fetchBlockTemplate() {
	r := s.rpc()
	t := s.currentBlockTemplate()
	reply, err := r.GetWork()
	if err != nil {
		log.Printf("Error while refreshing block template on %s: %s", r.Name, err)
		return
	}
	// No need to update, we have fresh job
	if t != nil && t.Header == reply[0] {
		return
	}
	diff := util.TargetHexToDiff(reply[2])
	height, err := strconv.ParseUint(strings.Replace(reply[3], "0x", "", -1), 16, 64)

	pendingReply := &rpc.GetBlockReplyPart{
		Difficulty: util.ToHex(s.config.Proxy.Difficulty),
		Number:     reply[3],
	}

	newTemplate := BlockTemplate{
		Header:               reply[0],
		Seed:                 reply[1],
		Target:               reply[2],
		Height:               height,
		Difficulty:           diff,
		GetPendingBlockCache: pendingReply,
		headers:              make(map[string]heightDiffPair),
	}
	// Copy job backlog and add current one
	newTemplate.headers[reply[0]] = heightDiffPair{
		diff:   diff,
		height: height,
	}
	if t != nil {
		for k, v := range t.headers {
			if v.height > height-maxBacklog {
				newTemplate.headers[k] = v
			}
		}
	}
	s.blockTemplate.Store(&newTemplate)
	log.Printf("New block to mine on %s at height %d / %s / %d", r.Name, height, reply[0][0:10], diff)

	// Stratum
	for i, setting := range s.config.Proxy.Stratums {
		if setting.Enabled {
			go s.broadcastNewJobs(i)
		}

	//nicehash
	if s.config.Proxy.StratumNiceHash.Enabled {
		go s.broadcastNewJobsNH()
	}
}

func (s *ProxyServer) fetchPendingBlock() (*rpc.GetBlockReplyPart, uint64, int64                                                                             , error) {
        rpc := s.rpc()
        reply, err := rpc.GetPendingBlock()
        if err != nil {
                log.Printf("Error while refreshing pending block on %s: %s", rpc                                                                             .Name, err)
                return nil, 0, 0, err
        }
        blockNumber, err := reply.Number, nil//strconv.ParseUint(reply.Number, 1                                                                             0, 64)
        if err != nil {
                log.Println("Can't parse pending block number")
                return nil, 0, 0, err
        }
        blockDiff, err := strconv.ParseInt(reply.Difficulty, 10, 64)

        if err != nil {
                log.Println("Can't parse pending block difficulty")
                return nil, 0, 0, err
        }
        return reply, blockNumber, blockDiff, nil
}
