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

	var resp ResponseClaim
	resp.ClaimMessage = req

	pc := rc.queryAndAddPreCommit(req.BlockHeight, req.BlockHash, req.Address)
	if pc != nil {
		// already claimed in current block
		resp.BlockHeight = pc.BlockHeight
		return c.Send(msgQuery, &resp)
	}

	var claim *Claim = nil
	var ia *IScoreAccount = nil
	var err error
	opts := rc.mgr.gOpts
	isDB := opts.db

	// read from claim DB
	cDB := isDB.GetClaimDB()
	bucket, _ := cDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(req.Address.Bytes())
	if bs != nil {
		claim, _ = NewClaimFromBytes(bs)
	}

	// read from account query DB
	aDB := isDB.GetQueryDB(req.Address)
	bucket, _ = aDB.GetBucket(db.PrefixIScore)
	bs, _ = bucket.Get(req.Address.Bytes())
	if bs != nil {
		ia, err = NewIScoreAccountFromBytes(bs)
		if nil != err {
			return c.Send(msgQuery, &resp)
		}
		ia.Address = req.Address
		resp.BlockHeight = ia.BlockHeight
	} else {
		// No Info. about account
		return c.Send(msgQuery, &resp)
	}

	if claim != nil {
		if ia.BlockHeight == claim.BlockHeight {
			// already claimed in current period
			return c.Send(msgQuery, &resp)
		}
		// subtract claimed I-Score
		ia.IScore.Sub(&ia.IScore.Int, &claim.IScore.Int)
	}

	// set calculated I-Score to response
	resp.IScore = ia.IScore

	// update preCommit with calculated I-Score
	rc.updatePreCommit(req.BlockHeight, req.BlockHash, ia)

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

func (rc *rewardCalculate) updatePreCommit(blockHeight uint64, blockHash []byte, ia *IScoreAccount) {
	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()

	// find preCommitMap and update ClaimData
	for _, pcMap := range rc.preCommitMapList {
		if pcMap.BlockHeight == blockHeight && bytes.Compare(pcMap.BlockHash, blockHash) == 0 {
			claim, ok := pcMap.claimMap[ia.Address]
			if false == ok {
				log.Printf("Failed to update preCommit: preCommit is nil\n")
			}
			claim.BlockHeight = ia.BlockHeight
			claim.IScore = ia.IScore
			log.Printf("Update claim preCommit %s\n", claim.String())
			return
		}
	}
}

func (rc *rewardCalculate) queryAndAddPreCommit(blockHeight uint64, blockHash []byte, address common.Address) *Claim {
	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()

	var claim = new(Claim)
	claim.Address = address

	// find preCommitMap and insert address
	for _, pcMap := range rc.preCommitMapList {
		if pcMap.BlockHeight == blockHeight && bytes.Compare(pcMap.BlockHash, blockHash) == 0 {
			pc, ok := pcMap.claimMap[address]
			if true == ok {
				// find preCommit
				return pc
			} else {
				// insert preCommit
				pcMap.claimMap[address] = claim
				return nil
			}
		}
	}

	// There is no preCommitMap.

	if nil == rc.preCommitMapList {
		// initialize preCommitMap list
		rc.preCommitMapList = make([]*preCommitMap, 0)
	}

	// there is no preCommitMap. make new preCommitMap and insert preCommit
	var pcMap = new(preCommitMap)
	pcMap.BlockHash = blockHash
	pcMap.BlockHeight = blockHeight
	pcMap.claimMap = make(map[common.Address]*Claim)
	pcMap.claimMap[claim.Address] = claim

	// append new preCommitMap to preCommitMapList
	rc.preCommitMapList = append(rc.preCommitMapList, pcMap)

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
				rc.preCommitMapList = nil
			} else {
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

	cDB := rc.mgr.gOpts.db.GetClaimDB()
	bucket, _ := cDB.GetBucket(db.PrefixIScore)

	// find preCommit and write preCommit to claimMap
	for _, pcMap := range rc.preCommitMapList {
		if pcMap.BlockHeight == blockHeight && bytes.Compare(pcMap.BlockHash, blockHash) == 0 {
			for _, pc := range pcMap.claimMap {
				if pc.IScore.Sign() == 0 {
					continue
				}

				bs, _ := bucket.Get(pc.ID())
				if nil != bs {
					claim, _ := NewClaimFromBytes(bs)
					if pc.BlockHeight <= claim.BlockHeight {
						continue
					}
					// update with old I-Score
					pc.IScore.Add(&pc.IScore.Int, &claim.IScore.Int)
				}

				log.Printf("Insert preCommit(%s) to claim DB\n", pc.String())
				// write to claim DB
				value := pc.Bytes()
				bucket.Set(pc.ID(), value)
			}

			// delete all preCommit
			rc.preCommitMapList = nil

			return true
		}
	}
	return false
}
