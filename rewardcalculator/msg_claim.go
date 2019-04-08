package rewardcalculator

import (
	"bytes"
	"log"
	"sync"

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

func (mh *msgHandler) claim(c ipc.Connection, id uint32, data []byte) error {
	var req ClaimMessage
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		log.Printf("Failed to deserialize CLAIM message. err=%+v", err)
		return err
	}

	blockHeight, IScore := DoClaim(mh.mgr.ctx, &req)

	var resp ResponseClaim
	resp.ClaimMessage = req
	resp.BlockHeight = blockHeight
	if IScore != nil {
		resp.IScore = *IScore
	}

	return c.Send(msgClaim, id, &resp)
}

func DoClaim(ctx *Context, req *ClaimMessage) (uint64, *common.HexInt) {
	claim := ctx.preCommit.queryAndAdd(req.BlockHeight, req.BlockHash, req.Address)
	if claim != nil {
		// already claimed in current block
		return claim.BlockHeight, nil
	}

	var ia *IScoreAccount = nil
	var err error
	isDB := ctx.db

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
			return 0, nil
		}
		ia.Address = req.Address
	} else {
		// No Info. about account
		return 0, nil
	}

	if claim != nil {
		if ia.BlockHeight == claim.BlockHeight {
			// already claimed in current period
			return ia.BlockHeight, nil
		}
		// subtract claimed I-Score
		ia.IScore.Sub(&ia.IScore.Int, &claim.IScore.Int)
	}

	// update preCommit with calculated I-Score
	ctx.preCommit.update(req.BlockHeight, req.BlockHash, ia)

	return ia.BlockHeight, &ia.IScore
}

type CommitBlock struct {
	Success     bool
	BlockHeight uint64
	BlockHash   []byte
}

func (mh *msgHandler) commitBlock(c ipc.Connection, id uint32, data []byte) error {
	var req CommitBlock
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); nil != err {
		return err
	}

	ret := true
	if req.Success == true {
		ret = mh.mgr.ctx.preCommit.writeClaimToDB(mh.mgr.ctx, req.BlockHeight, req.BlockHash)
	} else {
		mh.mgr.ctx.preCommit.delete(req.BlockHeight, req.BlockHash)
	}

	var resp CommitBlock
	resp = req
	resp.Success = ret

	return c.Send(msgCommitBlock, id, &resp)
}


type preCommit struct {
	lock     sync.RWMutex
	dataList []*preCommitData
}

type preCommitData struct {
	BlockHeight uint64
	BlockHash   []byte
	claimMap    map[common.Address]*Claim
}

func (pc *preCommit) queryAndAdd(blockHeight uint64, blockHash []byte, address common.Address) *Claim {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	var claim = new(Claim)
	claim.Address = address

	// find preCommitData and insert claim
	for _, pcData := range pc.dataList {
		if pcData.BlockHeight == blockHeight && bytes.Compare(pcData.BlockHash, blockHash) == 0 {
			claim, ok := pcData.claimMap[address]
			if true == ok {
				// find claim
				return claim
			} else {
				// insert new claim
				pcData.claimMap[address] = claim
				return nil
			}
		}
	}

	// There is no preCommitData.

	if nil == pc.dataList {
		// initialize preCommitData list
		pc.dataList = make([]*preCommitData, 0)
	}

	// there is no preCommitData. make new preCommitData and insert claim
	var pcData = new(preCommitData)
	pcData.BlockHash = blockHash
	pcData.BlockHeight = blockHeight
	pcData.claimMap = make(map[common.Address]*Claim)
	pcData.claimMap[claim.Address] = claim

	// append new preCommitData
	pc.dataList = append(pc.dataList, pcData)

	return nil
}

func (pc *preCommit) update(blockHeight uint64, blockHash []byte, ia *IScoreAccount) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	// find preCommitData and update claim
	for _, data := range pc.dataList {
		if data.BlockHeight == blockHeight && bytes.Compare(data.BlockHash, blockHash) == 0 {
			claim, ok := data.claimMap[ia.Address]
			if false == ok {
				log.Printf("Failed to update preCommit: preCommit is nil\n")
				return
			}
			claim.BlockHeight = ia.BlockHeight
			claim.IScore = ia.IScore
			return
		}
	}
}

func (pc *preCommit) delete(blockHeight uint64, blockHash []byte) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	listLen := len(pc.dataList)
	if listLen == 0 {
		return
	}

	// find preCommitData and delete
	for i, data := range pc.dataList {
		if data.BlockHeight == blockHeight && bytes.Compare(data.BlockHash, blockHash) == 0 {
			if listLen == 1 {
				pc.dataList = nil
			} else {
				pc.dataList[i] = pc.dataList[listLen-1]
				pc.dataList = pc.dataList[:listLen-1]
			}
			break
		}
	}
}

func (pc *preCommit) writeClaimToDB(ctx *Context, blockHeight uint64, blockHash []byte) bool {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	claimDB := ctx.db.GetClaimDB()
	bucket, _ := claimDB.GetBucket(db.PrefixIScore)

	// find preCommit and write preCommit to preCommitData
	for _, data := range pc.dataList {
		if data.BlockHeight == blockHeight && bytes.Compare(data.BlockHash, blockHash) == 0 {
			for _, pc := range data.claimMap {
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

				// write to claim DB
				bucket.Set(pc.ID(), pc.Bytes())
			}

			// delete all preCommitData
			pc.dataList = nil

			return true
		}
	}
	return false
}
