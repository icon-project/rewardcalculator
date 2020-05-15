package core

import (
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/ipc"
)

type RollBackRequest struct {
	BlockHeight uint64
	BlockHash   []byte
}

func (rb *RollBackRequest) String() string {
	return fmt.Sprintf("BlockHeight: %d, BlockHash: %s", rb.BlockHeight, hex.EncodeToString(rb.BlockHash))
}

type RollBackResponse struct {
	Success bool
	RollBackRequest
}

func (rb *RollBackResponse) String() string {
	return fmt.Sprintf("Success: %s, %s", strconv.FormatBool(rb.Success), rb.RollBackRequest.String())
}

func (mh *msgHandler) rollback(c ipc.Connection, id uint32, data []byte) error {
	success := true
	var req RollBackRequest
	var err error
	mh.mgr.IncreaseMsgTask()
	if _, err = codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}
	log.Printf("\t ROLLBACK request: %s", req.String())

	ctx := mh.mgr.ctx

	err = DoRollBack(ctx, &req)

	if err != nil {
		log.Printf("Failed to rollback %d. %v", req.BlockHeight, err)
		success = false
	}

	// send ROLLBACK response
	var resp RollBackResponse
	resp.Success = success
	resp.BlockHeight = req.BlockHeight
	resp.BlockHash = make([]byte, BlockHashSize)
	copy(resp.BlockHash, req.BlockHash)

	mh.mgr.DecreaseMsgTask()
	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgRollBack), id, resp.String())
	return c.Send(MsgRollBack, id, &resp)
}

func DoRollBack(ctx *Context, req *RollBackRequest) error {
	var err error
	idb := ctx.DB
	blockHeight := req.BlockHeight

	log.Printf("Start Rollback to %d", blockHeight)

	// check Rollback block height
	if err := checkRollback(ctx, blockHeight); err != nil {
		return err
	}

	// notify rollback to other goroutines
	ctx.CancelCalculation.notifyRollback()

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

func checkRollback(ctx *Context, rollback uint64) error {
	idb := ctx.DB
	if idb.getPrevCalcDoneBH() >= rollback {
		return &RollbackLowBlockHeightError{idb.getPrevCalcDoneBH(), rollback}
	}
	return nil
}

func checkAccountDBRollback(ctx *Context, rollback uint64) bool {
	idb := ctx.DB
	if rollback > idb.getCalcDoneBH() {
		log.Printf("No need to Rollback account DB. %d > %d", rollback, idb.getCalcDoneBH())
		return false
	}

	return true
}

const (
	CancelNone     uint64 = 0
	CancelExit            = 1
	CancelRollback        = 2
)

type CancelCalculation struct {
	channel    chan struct{} // Do not close channel in normal case. close when caught SIGTERM/SIGINT or got rollback message.
	mutex      sync.Mutex
	cancelCode uint64
}

func (c *CancelCalculation) newChannel() {
	c.channel = make(chan struct{})
}

func (c *CancelCalculation) GetChannel() chan struct{} {
	return c.channel
}

func (c *CancelCalculation) notifyCancelCalculation(cancelPurpose uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	close(c.channel)

	// make new channel for notification
	c.cancelCode = cancelPurpose
	c.newChannel()
}

func (c *CancelCalculation) notifyRollback() {
	// close channel to notify Rollback to all listening goroutines
	c.notifyCancelCalculation(CancelRollback)
}

func (c *CancelCalculation) notifyExit() {
	// close channel to notify Exiting RC process to all listening goroutines
	c.notifyCancelCalculation(CancelExit)
}

func NewCancel() *CancelCalculation {
	c := new(CancelCalculation)
	c.newChannel()
	return c
}

type RollbackLowBlockHeightError struct {
	Comparison  uint64
	BlockHeight uint64
}

func (e *RollbackLowBlockHeightError) Error() string {
	return fmt.Sprintf("too low block height %d >= %d", e.Comparison, e.BlockHeight)
}
