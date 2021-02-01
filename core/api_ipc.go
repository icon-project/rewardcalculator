package core

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/icon-project/rewardcalculator/common/ipc"
)

type RCIPC struct {
	conn ipc.Connection
	id   uint32
}

func InitRCIPC(net string, address string) (*RCIPC, error) {
	rc := new(RCIPC)

	// Connect to server
	retry := 0
RETRY:
	conn, err := ipc.Dial(net, address)
	if err != nil {
		if retry != 5 {
			time.Sleep(200 * time.Millisecond)
			retry++
			goto RETRY
		}
		fmt.Printf("Failed to dial %s:%s with %d tries. err=%+v\n", net, address, retry, err)
		return nil, err
	}
	rc.conn = conn

	// flush READY message
	for true {
		var m ResponseVersion
		msg, _, _ := conn.Receive(m)
		if msg == MsgReady {
			break
		}
	}

	return rc, nil
}

func FiniRCIPC(ipc *RCIPC) {
	ipc.conn.Close()
}

func (rc *RCIPC) SendVersion() (*ResponseVersion, error) {
	resp := new(ResponseVersion)
	rc.id++
	err := rc.conn.SendAndReceive(MsgVersion, rc.id, nil, resp)
	return resp, err
}

func (rc *RCIPC) SendClaim(address string, blockHeight uint64, blockHash string,
	txIndex uint64, txHash string, noCommitClaim bool, noCommitBlock bool) (*ResponseClaim, error) {
	var req ClaimMessage
	resp := new(ResponseClaim)

	// Send CLAIM and get response
	req.Address.SetString(address)
	req.BlockHeight = blockHeight
	req.BlockHash = make([]byte, BlockHashSize)
	if len(blockHash) == 0 {
		binary.BigEndian.PutUint64(req.BlockHash, blockHeight)
	} else {
		bh, err := hex.DecodeString(blockHash)
		if err != nil {
			log.Printf("Failed to send CLAIM. Invalid block hash. %v\n", err)
			return nil, err
		}
		copy(req.BlockHash, bh)
	}
	req.TXIndex = txIndex
	req.TXHash = make([]byte, TXHashSize)
	if len(txHash) == 0 {
		binary.BigEndian.PutUint64(req.TXHash, blockHeight+txIndex)
	} else {
		th, err := hex.DecodeString(txHash)
		if err != nil {
			log.Printf("Failed to send CLAIM. Invalid TX hash. %v\n", err)
		}
		copy(req.TXHash, th)
	}

	log.Printf("Send CLAIM message: %s\n", req.String())
	rc.id++
	err := rc.conn.SendAndReceive(MsgClaim, rc.id, &req, resp)
	if err != nil {
		return resp, err
	}
	log.Printf("Get CLAIM response: %s\n", resp.String())

	// send COMMIT_CLAIM and get ack
	if noCommitClaim == false && resp.IScore.Sign() != 0 {
		err := rc.SendCommitClaim(true, address, blockHeight, blockHash, txIndex, txHash)
		if err != nil {
			log.Printf("Failed to send COMMIT_CLAIM and get response. %v", err)
			return resp, err
		}
	}

	// send COMMIT_BLOCK and get response
	if noCommitBlock == false {
		_, err := rc.SendCommitBlock(true, blockHeight, blockHash)
		if err != nil {
			log.Printf("Failed to send COMMIT_BLOCK and get response. %v", err)
			return resp, err
		}
	}

	return resp, nil
}

func (rc *RCIPC) SendQuery(address string, txHash string) (*ResponseQuery, error) {
	var req Query
	resp := new(ResponseQuery)

	req.Address.SetString(address)
	req.TXHash = make([]byte, TXHashSize)
	th, err := hex.DecodeString(txHash)
	if err != nil {
		log.Printf("Failed to QUERY. Invalid TX hash. %v\n", err)
		return resp, err
	}
	copy(req.TXHash, th)

	err = rc.conn.SendAndReceive(MsgQuery, rc.id, &req, resp)

	return resp, err
}

func (rc *RCIPC) SendCalculate(iissData string, blockHeight uint64) (*CalculateResponse, error) {
	var req CalculateRequest
	resp := new(CalculateResponse)

	req.Path = iissData
	req.BlockHeight = blockHeight

	// Send CALCULATE and get response
	err := rc.conn.SendAndReceive(MsgCalculate, rc.id, &req, resp)
	if err != nil {
		log.Printf("Failed to get CALCULATE response. %v", err)
		return nil, err
	}
	log.Printf("Get CALCULATE get response: %s\n", resp.String())
	if resp.Status != CalcRespStatusOK {
		return resp, nil
	}

	// Get CALCULATE_DONE
	var respDone CalculateDone
	msg, id, err := rc.conn.Receive(&respDone)
	if err != nil {
		log.Printf("Failed to get CALCULATE_DONE. %v", err)
		return resp, err
	}
	if msg == MsgCalculateDone {
		log.Printf("Get CALCULATE_DONE: %s\n", respDone.String())
	} else {
		log.Printf("Get invalid response : (msg:%d, id:%d)\n", msg, id)
	}

	return resp, nil
}

func (rc *RCIPC) SendStartBlock(success bool, blockHeight uint64, blockHash string) (*StartBlock, error) {
	var req StartBlock
	resp := new(StartBlock)

	req.BlockHash = make([]byte, BlockHashSize)
	if len(blockHash) == 0 {
		binary.BigEndian.PutUint64(req.BlockHash, blockHeight)
	} else {
		bh, err := hex.DecodeString(blockHash)
		if err != nil {
			log.Printf("Failed to START_BLOCK. Invalid block hash. %v\n", err)
			return resp, err
		}
		copy(req.BlockHash, bh)
	}
	req.BlockHeight = blockHeight

	log.Printf("Send START_BLOCK message: %s\n", req.String())
	rc.id++
	err := rc.conn.SendAndReceive(MsgStartBlock, rc.id, &req, &resp)
	log.Printf("Get START_BLOCK response: %s\n", resp.String())

	return resp, err
}

func (rc *RCIPC) SendCommitBlock(success bool, blockHeight uint64, blockHash string) (*CommitBlock, error) {
	var req CommitBlock
	resp := new(CommitBlock)

	req.Success = success
	req.BlockHash = make([]byte, BlockHashSize)
	if len(blockHash) == 0 {
		binary.BigEndian.PutUint64(req.BlockHash, blockHeight)
	} else {
		bh, err := hex.DecodeString(blockHash)
		if err != nil {
			log.Printf("Failed to COMMIT_BLOCK. Invalid block hash. %v\n", err)
			return resp, err
		}
		copy(req.BlockHash, bh)
	}
	req.BlockHeight = blockHeight

	log.Printf("Send COMMIT_BLOCK message: %s\n", req.String())
	rc.id++
	err := rc.conn.SendAndReceive(MsgCommitBlock, rc.id, &req, &resp)
	log.Printf("Get COMMIT_BLOCK response: %s\n", resp.String())

	return resp, err
}

func (rc *RCIPC) SendCommitClaim(success bool, address string, blockHeight uint64, blockHash string,
	txIndex uint64, txHash string) error {
	var req CommitClaim

	req.Success = success
	req.Address.SetString(address)
	req.BlockHeight = blockHeight
	req.BlockHash = make([]byte, BlockHashSize)
	if len(blockHash) == 0 {
		binary.BigEndian.PutUint64(req.BlockHash, blockHeight)
	} else {
		bh, err := hex.DecodeString(blockHash)
		if err != nil {
			log.Printf("Failed to send COMMIT_CLAIM. Invalid block hash. %v\n", err)
			return err
		}
		copy(req.BlockHash, bh)
	}
	req.TXIndex = txIndex
	req.TXHash = make([]byte, TXHashSize)
	if len(txHash) == 0 {
		binary.BigEndian.PutUint64(req.TXHash, blockHeight+txIndex)
	} else {
		th, err := hex.DecodeString(txHash)
		if err != nil {
			log.Printf("Failed to send COMMIT_CLAIM. Invalid TX hash. %v\n", err)
			return err
		}
		copy(req.TXHash, th)
	}

	log.Printf("Send COMMIT_CLAIM message: %s\n", req.String())
	rc.id++
	err := rc.conn.SendAndReceive(MsgCommitClaim, rc.id, &req, nil)
	log.Printf("Get COMMIT_CLAIM ack. %v\n", err)

	return err
}

func (rc *RCIPC) SendQueryCalculateStatus() (*QueryCalculateStatusResponse, error) {
	resp := new(QueryCalculateStatusResponse)

	// Send QUERY_CALCULATE_STATUS and get response
	err := rc.conn.SendAndReceive(MsgQueryCalculateStatus, rc.id, nil, resp)
	if err != nil {
		log.Printf("Failed to get QUERY_CALCULATE_STATUS response. %v", err)
		return nil, err
	}
	log.Printf("Get QUERY_CALCULATE_STATUS response: %s\n", resp.String())
	return resp, nil
}

func (rc *RCIPC) SendQueryCalculateResult(blockHeight uint64) (*QueryCalculateResultResponse, error) {
	resp := new(QueryCalculateResultResponse)

	// Send QUERY_CALCULATE_RESULT and get response
	err := rc.conn.SendAndReceive(MsgQueryCalculateResult, rc.id, &blockHeight, resp)
	if err != nil {
		log.Printf("Failed to get QUERY_CALCULATE_RESULT response. %v", err)
		return nil, err
	}

	log.Printf("Get QUERY_CALCULATE_RESULT response: %s\n", resp.String())
	return resp, nil
}

func (rc *RCIPC) SendRollback(blockHeight uint64, blockHash string) (*RollBackResponse, error) {
	var req RollBackRequest
	resp := new(RollBackResponse)

	hash, err := hex.DecodeString(blockHash)
	if err != nil {
		log.Printf("Failed to decode blockHash. %v\n", err)
		return nil, err
	}

	req.BlockHeight = blockHeight
	req.BlockHash = make([]byte, BlockHashSize)
	copy(req.BlockHash, hash)

	err = rc.conn.SendAndReceive(MsgRollBack, rc.id, &req, &resp)
	if err != nil {
		log.Printf("Failed to ROLLBACK response. %v\n", err)
		return nil, err
	}
	log.Printf("Get ROLLBACK get response: %s\n", resp.String())
	return resp, nil
}

func (rc *RCIPC) SendInit(blockHeight uint64) (*ResponseInit, error) {
	resp := new(ResponseInit)

	err := rc.conn.SendAndReceive(MsgINIT, rc.id, &blockHeight, &resp)
	if err != nil {
		log.Printf("Failed to INIT response. %v\n", err)
		return nil, err
	}
	fmt.Printf("Get INIT response: %s\n", resp.String())
	return resp, nil
}
