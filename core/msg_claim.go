package core

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strconv"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
)

const claimMinIScore = 1000

var BigIntClaimMinIScore = big.NewInt(claimMinIScore)

type ClaimMessage struct {
	Address     common.Address
	BlockHeight uint64
	BlockHash   []byte
	TXIndex     uint64
	TXHash      []byte
}

func (cm *ClaimMessage) String() string {
	return fmt.Sprintf("Address: %s, BlockHeight: %d, BlockHash: %s, TXIndex: %d, TXHash: %s",
		cm.Address.String(),
		cm.BlockHeight,
		hex.EncodeToString(cm.BlockHash),
		cm.TXIndex,
		hex.EncodeToString(cm.TXHash))
}

type ResponseClaim struct {
	ClaimMessage
	IScore common.HexInt
}

func (rc *ResponseClaim) String() string {
	return fmt.Sprintf("%s, IScore: %s", rc.ClaimMessage.String(), rc.IScore.String())
}

func (mh *msgHandler) claim(c ipc.Connection, id uint32, data []byte) error {
	var req ClaimMessage
	mh.mgr.AddMsgTask()
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		log.Printf("Failed to deserialize CLAIM message. err=%+v", err)
		return err
	}
	log.Printf("\t CLAIM request: %s", req.String())

	blockHeight, IScore := DoClaim(mh.mgr.ctx, &req)

	var resp ResponseClaim
	resp.ClaimMessage = req
	resp.BlockHeight = blockHeight
	if IScore != nil {
		resp.IScore.Set(&IScore.Int)
	}

	mh.mgr.DoneMsgTask()
	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgClaim), id, resp.String())
	return c.Send(MsgClaim, id, &resp)
}

// DoClaim calculates the I-Score that the ICONist in ClaimMessage can get.
// Writes calculated I-Score and block height to claim DB.
// It returns the I-Score block height and I-Score. nil I-Score means zero I-Score.
// In error case, block height is zero and I-Score is nil.
func DoClaim(ctx *Context, req *ClaimMessage) (uint64, *common.HexInt) {
	pcDB := ctx.DB.getPreCommitDB()
	preCommit := newPreCommit(req.BlockHeight, req.BlockHash, req.TXIndex, req.TXHash, req.Address)
	if preCommit.query(pcDB) == true {
		if preCommit.Confirmed && req.TXIndex != preCommit.TXIndex {
			// already claimed in current block
			return preCommit.BlockHeight, nil
		} else {
			// re-invoke same TX in same Block
			return preCommit.BlockHeight, &preCommit.Data.IScore
		}
	}

	var claim *Claim = nil
	var ia *IScoreAccount = nil
	var err error
	isDB := ctx.DB

	var cDB, qDB db.Database
	var bucket db.Bucket
	var bs []byte

	// read from claim DB
	cDB = isDB.getClaimDB()
	bucket, _ = cDB.GetBucket(db.PrefixIScore)
	bs, _ = bucket.Get(req.Address.Bytes())
	if bs != nil {
		claim, _ = NewClaimFromBytes(bs)
	}

	// read from query DB
	qDB = isDB.getQueryDB(req.Address)
	bucket, _ = qDB.GetBucket(db.PrefixIScore)
	bs, _ = bucket.Get(req.Address.Bytes())
	if bs != nil {
		ia, err = NewIScoreAccountFromBytes(bs)
		if nil != err {
			log.Printf("Failed to get IScoreAccount. err=%+v", err)
			goto NoReward
		}
		ia.Address = req.Address
	} else {
		// No Info. about account
		goto NoReward
	}

	if claim != nil {
		if ia.BlockHeight == claim.Data.BlockHeight {
			// already claimed in current period
			return ia.BlockHeight, nil
		}
		// subtract claimed I-Score
		ia.IScore.Sub(&ia.IScore.Int, &claim.Data.IScore.Int)
	}

	// Can't claim an I-Score less than 1000
	if ia.IScore.Cmp(BigIntClaimMinIScore) == -1 {
		goto NoReward
	} else {
		var remain common.HexInt
		remain.Mod(&ia.IScore.Int, BigIntClaimMinIScore)
		ia.IScore.Sub(&ia.IScore.Int, &remain.Int)
	}

	// write preCommit with calculated I-Score
	err = preCommit.write(pcDB, &ia.IScore)
	if err != nil {
		log.Printf("Failed to write PreCommit. err=%+v", err)
		goto NoReward
	}

	return ia.BlockHeight, &ia.IScore

NoReward:
	return 0, nil
}

type CommitClaim struct {
	Success     bool
	Address     common.Address
	BlockHeight uint64
	BlockHash   []byte
	TXIndex     uint64
	TXHash      []byte
}

func (cc *CommitClaim) String() string {
	return fmt.Sprintf("Success: %s, Address: %s, BlockHeight: %d, BlockHash: %s, TXIndex: %d, TXHash: %s",
		strconv.FormatBool(cc.Success),
		cc.Address.String(),
		cc.BlockHeight,
		hex.EncodeToString(cc.BlockHash),
		cc.TXIndex,
		hex.EncodeToString(cc.TXHash))
}

func (mh *msgHandler) commitClaim(c ipc.Connection, id uint32, data []byte) error {
	var req CommitClaim
	var err error
	mh.mgr.AddMsgTask()

	if _, err = codec.MP.UnmarshalFromBytes(data, &req); nil != err {
		return err
	}
	log.Printf("\t COMMIT_CLAIM request: %s", req.String())

	err = DoCommitClaim(mh.mgr.ctx, &req)
	if err != nil {
		log.Printf("Failed to commit claim. %+v", err)
		return nil
	}

	mh.mgr.DoneMsgTask()
	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgCommitClaim), id, "ack")
	return c.Send(MsgCommitClaim, id, nil)
}

func DoCommitClaim(ctx *Context, req *CommitClaim) error {
	var err error
	preCommit := newPreCommit(req.BlockHeight, req.BlockHash, req.TXIndex, req.TXHash, req.Address)
	pcDB := ctx.DB.getPreCommitDB()

	if req.Success == true {
		err = preCommit.commit(pcDB)
	} else {
		err = preCommit.revert(pcDB)
	}

	if err != nil {
		log.Printf("Failed to commit claim. err=%+v", err)
	}

	// do not return error
	return nil
}

type CommitBlock struct {
	Success     bool
	BlockHeight uint64
	BlockHash   []byte
}

func (cb *CommitBlock) String() string {
	return fmt.Sprintf("Success: %s, BlockHeight: %d, BlockHash: %s",
		strconv.FormatBool(cb.Success),
		cb.BlockHeight,
		hex.EncodeToString(cb.BlockHash))
}

func (mh *msgHandler) commitBlock(c ipc.Connection, id uint32, data []byte) error {
	var req CommitBlock
	var err error
	mh.mgr.AddMsgTask()
	if _, err = codec.MP.UnmarshalFromBytes(data, &req); nil != err {
		return err
	}
	log.Printf("\t COMMIT_BLOCK request: %s", req.String())

	ret := true
	iDB := mh.mgr.ctx.DB
	if req.Success == true {
		err = writePreCommitToClaimDB(iDB.getPreCommitDB(), iDB.getClaimDB(), iDB.getClaimBackupDB(),
			req.BlockHeight, req.BlockHash)
		if err == nil {
			mh.mgr.ctx.DB.setCurrentBlockInfo(req.BlockHeight, req.BlockHash)
		}
	} else {
		err = flushPreCommit(iDB.getPreCommitDB(), req.BlockHeight, req.BlockHash)
	}

	if err != nil {
		log.Printf("Failed to commit block. %+v", err)
		ret = false
	}

	var resp CommitBlock
	resp = req
	resp.Success = ret

	mh.mgr.DoneMsgTask()
	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgCommitBlock), id, resp.String())
	return c.Send(MsgCommitBlock, id, &resp)
}
