package core

import (
	"encoding/hex"
	"fmt"
	"log"
	"strconv"

	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/ipc"
)

type RollBackRequest struct {
	BlockHeight uint64
	BlockHash []byte
}

func (rb *RollBackRequest) String() string {
	return fmt.Sprintf("BlockHeight: %d, BlockHash: %s", rb.BlockHeight, hex.EncodeToString(rb.BlockHash))
}

type RollBackResponse struct {
	Status bool
	RollBackRequest
}

func (rb *RollBackResponse) String() string {
	return fmt.Sprintf("Status: %s, %s", strconv.FormatBool(rb.Status), rb.RollBackRequest.String())
}

func (mh *msgHandler) rollback(c ipc.Connection, id uint32, data []byte) error {
	var req RollBackRequest
	var err error
	if _, err = codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}
	log.Printf("\t ROLLBACK request: %s", req.String())

	err = DoRollBack(mh.mgr.ctx, &req)

	// send ROLLBACK response
	var resp RollBackResponse
	if err == nil {
		resp.Status = true
	} else {
		log.Printf("Failed to rollback. %v", err)
		resp.Status = false
	}
	resp.BlockHeight = req.BlockHeight
	resp.BlockHash = make([]byte, BlockHashSize)
	copy(resp.BlockHash, req.BlockHash)

	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgRollBack), id, resp.String())
	return c.Send(MsgRollBack, id, &resp)
}

func DoRollBack(ctx *Context, req *RollBackRequest) error {
	var err error
	idb := ctx.DB
	blockHeight := req.BlockHeight

	log.Printf("Start Rollback to %d", blockHeight)

	// check Rollback block height
	if ok, err := checkRollback(ctx, blockHeight); ok != true {
		return err
	}

	// notify rollback to other goroutines
	ctx.Rollback.notifyRollback()

	// must Rollback claim DB first
	err = rollbackClaimDB(ctx, blockHeight, req.BlockHash)
	if err != nil {
		log.Printf("Failed to Rollback claim DB. %+v", err)
		return err
	}

	if checkAccountDBRollback(ctx, blockHeight) {
		err = idb.rollbackAccountDB(blockHeight)
		if err != nil {
			log.Printf("Failed to Rollback account DB. %+v", err)
			return err
		}
	}

	// rollback GV and Main/Sub P-Rep list
	ctx.RollbackManagementDB(blockHeight)

	return nil
}

func checkRollback(ctx *Context, rollback uint64) (bool, error) {
	idb := ctx.DB
	if idb.getPrevCalcDoneBH() >= rollback {
		return false, &RollbackLowBlockHeightError{idb.getPrevCalcDoneBH(), rollback}
	}
	return true, nil
}

func checkAccountDBRollback(ctx *Context, rollback uint64) bool {
	idb := ctx.DB
	if rollback >= idb.getCalcDoneBH() {
		log.Printf("No need to Rollback account DB. %d >= %d", rollback, idb.getCalcDoneBH())
		return false
	}

	return true
}

type Rollback struct {
	channel chan struct{}	// Do not close channel in normal case
}

func (rb *Rollback) newChannel() {
	rb.channel = make(chan struct{})
}

func (rb *Rollback) GetChannel() chan struct{} {
	return rb.channel
}

func (rb *Rollback) notifyRollback() {
	// close channel to notify Rollback to all listening goroutines
	close(rb.channel)

	// make new channel for notification
	rb.newChannel()
}

func NewRollback() *Rollback {
	rb := new(Rollback)
	rb.newChannel()
	return rb
}

type RollbackLowBlockHeightError struct {
	Comparison  uint64
	BlockHeight uint64
}

func (e *RollbackLowBlockHeightError) Error() string {
	return fmt.Sprintf("too low block height %d >= %d", e.Comparison, e.BlockHeight)
}
