package rewardcalculator

import (
	"bytes"
	"log"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
)

type ClaimMessage struct {
	Address     common.Address
	BlockHeight uint64
	BlockHash   []byte
}

type ResponseClaim struct {
	ClaimMessage
	IScore common.HexInt
}

func (rc *rewardCalculate) claim(c ipc.Connection, data []byte) error {
	var req ClaimMessage
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		log.Printf("Failed to deserialize CLAIM message. err=%+v", err)
		return err
	}

	claim := rc.queryPreCommit(req.BlockHeight, req.BlockHash, req.Address)

	var ia = new(IScoreAccount)

	if claim != nil {
		ia.Address = req.Address
		ia.BlockHeight = claim.BlockHeight
	} else {
		// read from claim DB
		cDB := rc.mgr.gOpts.db.GetClaimDB()
		bucket, err := cDB.GetBucket(db.PrefixIScore)
		if err != nil {
			return err
		}
		bs, err := bucket.Get(req.Address.Bytes())
		if err != nil {
			return err
		}
		if bs != nil {
			claim.SetBytes(bs)
		}

		// read from Query DB
		qDB := rc.mgr.gOpts.db.GetQueryDB(req.Address)
		bucket, err = qDB.GetBucket(db.PrefixIScore)
		if err != nil {
			return err
		}
		bs, err = bucket.Get(req.Address.Bytes())
		if err != nil {
			return err
		}
		if bs != nil {
			ia.SetBytes(bs)
			ia.Address = req.Address
		}

		// add to pre commit list
		rc.addPreCommit(req.BlockHeight, req.BlockHash, ia)
	}

	var resp ResponseClaim
	resp.ClaimMessage = req
	resp.BlockHeight = ia.BlockHeight
	resp.IScore = ia.IScore

	return c.Send(msgClaim, &resp)
}

type CommitBlock struct {
	Success     bool
	BlockHeight uint64
	BlockHash   []byte
}

func (rc *rewardCalculate) commitBlock(c ipc.Connection, data []byte) error {
	var req CommitBlock
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); nil != err {
		return err
	}

	ret := true
	if req.Success == true {
		ret = rc.writePreCommit(req.BlockHeight, req.BlockHash)
	} else {
		rc.deletePreCommit(req.BlockHeight, req.BlockHash)
	}

	var resp CommitBlock
	resp = req
	resp.Success = ret

	return c.Send(msgCommitBlock, &resp)
}

func (rc *rewardCalculate) addPreCommit(blockHeight uint64, blockHash []byte, ia *IScoreAccount) {
	var claim = new(Claim)
	claim.BlockHeight = ia.BlockHeight
	claim.IScore = ia.IScore
	claim.Address = ia.Address

	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()
	// initialize preCommitMap list
	if nil == rc.preCommitMapList {
		rc.preCommitMapList = make([]*preCommitMap, 0)
	}

	// find preCommitMap and insert ClaimData
	for _, pcMap := range rc.preCommitMapList {
		if pcMap.BlockHeight == blockHeight && bytes.Compare(pcMap.BlockHash, blockHash) == 0 {
			pcMap.claimMap[claim.Address] = claim
			log.Printf("Insert claim preCommit %s\n", claim.String())
			return
		}
	}

	// there is no preCommitMap. make new preCommitMap and insert address.
	var pcMap = new(preCommitMap)
	pcMap.BlockHash = blockHash
	pcMap.BlockHeight = blockHeight
	pcMap.claimMap = make(map[common.Address]*Claim)
	pcMap.claimMap[claim.Address] = claim
	log.Printf("Insert claim preCommit %s\n", claim.String())

	// append new preCommitMap to preCommitMapList
	rc.preCommitMapList = append(rc.preCommitMapList, pcMap)
}

func (rc *rewardCalculate) queryPreCommit(blockHeight uint64, blockHash []byte, address common.Address) *Claim {
	rc.claimLock.RLock()
	defer rc.claimLock.RUnlock()

	// find preCommitMap and insert address
	for _, pcMap := range rc.preCommitMapList {
		if pcMap.BlockHeight == blockHeight && bytes.Compare(pcMap.BlockHash, blockHash) == 0 {
			return pcMap.claimMap[address]
		}
	}

	// there is no preCommit
	return nil
}

func (rc *rewardCalculate) deletePreCommit(blockHeight uint64, blockHash []byte) {
	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()

	listLen := len(rc.preCommitMapList)
	if listLen == 0 {
		return
	}

	// find preCommit and delete
	for i, pcMap := range rc.preCommitMapList {
		if pcMap.BlockHeight == blockHeight && bytes.Compare(pcMap.BlockHash, blockHash) == 0 {
			if listLen == 1 {
				log.Printf("Clear claim preCommits (%d, %v)\n", blockHeight, blockHash)
				rc.preCommitMapList = nil
			} else {
				log.Printf("Delete claim preCommits (%d, %v)\n", blockHeight, blockHash)
				rc.preCommitMapList[i] = rc.preCommitMapList[listLen-1]
				rc.preCommitMapList = rc.preCommitMapList[:listLen-1]
			}
			break
		}
	}
}

func (rc *rewardCalculate) writePreCommit(blockHeight uint64, blockHash []byte) bool {
	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()

	// find preCommit and write preCommit to claimMap
	for _, pcMap := range rc.preCommitMapList {
		if pcMap.BlockHeight == blockHeight && bytes.Compare(pcMap.BlockHash, blockHash) == 0 {
			for _, claim := range pcMap.claimMap {
				log.Printf("Insert claim(%s) to claim DB\n", claim.String())
				// write to claim DB
				cDB := rc.mgr.gOpts.db.GetClaimDB()
				bucket, _ := cDB.GetBucket(db.PrefixIScore)
				value, _ := claim.Bytes()
				bucket.Set(claim.ID(), value)
			}

			// delete all preCommit
			rc.preCommitMapList = nil

			return true
		}
	}
	return false
}

func (rc *rewardCalculate) setClaimToGC(key []byte) {
	cDB := rc.mgr.gOpts.db.GetClaimDB()

	bucket, _ := cDB.GetBucket(db.PrefixIScore)

	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()

	bs, _ := bucket.Get(key)
	if bs != nil {
		claim, _ := NewClaimFromBytes(bs)
		if claim != nil {
			claim.applyGC = true
			value, _ := claim.Bytes()
			bucket.Set(key, value)
		}
	}
}

func (rc *rewardCalculate) garbageCollectClaim() {
	cDB := rc.mgr.gOpts.db.GetClaimDB()

	bucket, _ := cDB.GetBucket(db.PrefixIScore)
	iter, _ := cDB.GetIterator()

	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()

	iter.New(nil, nil)
	for iter.Next() {
		// read
		key := iter.Key()[len(db.PrefixIScore):]
		claim, err := NewClaimFromBytes(iter.Value())
		if err != nil {
			log.Printf("Can't read data with iterator\n")
			continue
		}

		if claim.applyGC {
			bucket.Delete(key)
		}
	}
	iter.Release()
}