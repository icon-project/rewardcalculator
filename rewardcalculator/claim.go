package rewardcalculator

import (
	"bytes"
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
		return err
	}

	opts := rc.mgr.gOpts
	isDB := opts.db

	ia := new(IScoreAccount)
	rc.claimLock.RLock()
	blockHeight := rc.claimMap[req.Address]
	rc.claimLock.RUnlock()
	if blockHeight != 0 {
		ia.Address = req.Address
		ia.BlockHeight = blockHeight
	} else {
		// read from account DB snapshot
		isDB.snapshotLock.RLock()
		snapshot := opts.GetAccountDBSnapshot(req.Address)
		bs, _ := snapshot.Get(req.Address.Bytes())
		isDB.snapshotLock.RUnlock()
		if bs != nil {
			ia.SetBytes(bs)

			// add to pre-commit list
			rc.addPreCommit(req.BlockHeight, req.BlockHash, ia.Address)
		}
	}

	var resp ResponseClaim
	resp.ClaimMessage = req
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
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
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

func (rc *rewardCalculate) addPreCommit(blockHeight uint64, blockHash []byte, address common.Address) {
	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()
	// initialize preCommitMap list
	if rc.preCommitMapList == nil {
		rc.preCommitMapList = make([]*preCommitMap, 0)
	}

	// find preCommitMap and insert address
	for _, pcMap := range rc.preCommitMapList {
		if pcMap.BlockHeight == blockHeight && bytes.Compare(pcMap.BlockHash, blockHash) == 0 {
			pcMap.claimMap[address] = blockHeight
			return
		}
	}

	// there is no preCommitMap. make new preCommitMap and insert address.
	pcMap := new(preCommitMap)
	pcMap.BlockHash = blockHash
	pcMap.BlockHeight = blockHeight
	pcMap.claimMap = make(map[common.Address]uint64)
	pcMap.claimMap[address] = blockHeight

	// append new preCommitMap to preCommitMapList
	rc.preCommitMapList = append(rc.preCommitMapList, pcMap)
}

func (rc *rewardCalculate) deletePreCommit(blockHeight uint64, blockHash []byte) {
	newList := make([]*preCommitMap, 0, len(rc.preCommitMapList))
	index := 0

	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()

	// find preCommit and delete
	for _, pcMap := range rc.preCommitMapList {
		if pcMap.BlockHeight == blockHeight && bytes.Compare(pcMap.BlockHash, blockHash) == 0 {
		} else {
			newList[index] = pcMap
			index++
		}
	}

	if len(newList) < cap(newList) {
		rc.preCommitMapList = newList
	}
}

func (rc *rewardCalculate) writePreCommit(blockHeight uint64, blockHash []byte) bool {
	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()

	// find preCommit and write preCommit to claimMap
	for _, pcMap := range rc.preCommitMapList {
		if pcMap.BlockHeight == blockHeight && bytes.Compare(pcMap.BlockHash, blockHash) == 0 {
			for k, v := range pcMap.claimMap {
				rc.claimMap[k] = v
			}

			// delete all preCommit
			rc.preCommitMapList = nil

			return true
		}
	}
	return false
}

func (rc *rewardCalculate) modifyClaimMap() {
	opts := rc.mgr.gOpts

	rc.claimLock.Lock()
	defer rc.claimLock.Unlock()

	for addr, blockHeight := range rc.claimMap {
		adb := opts.GetAccountDB(addr)
		bucket, _ := adb.GetBucket(db.PrefixIScore)
		data, _ := bucket.Get(addr.Bytes())
		ia := new(IScoreAccount)
		if data != nil {
			ia.SetBytes(data)

			if blockHeight <= ia.BlockHeight {
				// remove from claimMap
				delete(rc.claimMap, addr)
			}
		}
	}
}
