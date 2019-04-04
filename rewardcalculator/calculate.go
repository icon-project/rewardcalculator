package rewardcalculator

import (
	"log"
	"sync"
	"time"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"golang.org/x/crypto/sha3"
)

const writeBatchCount = 10

type CalculateRequest struct {
	Path        string
	BlockHeight uint64
}

type CalculateResponse struct {
	Success     bool
	BlockHeight uint64
	StateHash   []byte
}

func calculateDelegationReward(delegation *common.HexInt, start uint64, end uint64,
	gvList []*GovernanceVariable, pRep *PRepCandidate) *common.HexInt {
	// adjust start and end with P-Rep candidate
	if start < pRep.Start {
		start = pRep.Start
	}

	if pRep.End != 0 && pRep.End < end {
		end = pRep.End
	}

	total := new(common.HexInt)

	// period in gv
	for i, gv := range gvList {
		var s, e = start, end

		if start < gv.BlockHeight {
			s = gv.BlockHeight
		}
		if i+1 < len(gvList) && gvList[i+1].BlockHeight < end {
			e = gvList[i+1].BlockHeight
		}

		if e  <= s {
			continue
		}
		period := common.NewHexIntFromUint64(e - s)

		// reward = delegation amount * period * GV
		var reward common.HexInt
		reward.Mul(&delegation.Int, &period.Int)
		reward.Mul(&reward.Int, &gv.RewardRep.Int)

		//log.Printf("dg: %s, period: %s, Rep: %s. reward: %s\n",
		//	delegation.String(), period.String(), gv.RewardRep.String(), reward.String())

		// update total
		total.Add(&total.Int, &reward.Int)
	}

	return total
}

func calculateIScore(ia *IScoreAccount,  gvList []*GovernanceVariable,
	pRepCandidates map[common.Address]*PRepCandidate, blockHeight uint64) bool {
	// IScore = old + period * G.V * sum(valid dgAmount)
	if blockHeight == 0 {
		blockHeight = ia.BlockHeight + 1
	}

	if blockHeight == ia.BlockHeight {
		return false
	}

	var totalReward common.HexInt
	for _, dg := range ia.Delegations {
		if dg.Delegate.Int.Sign() == 0 {
			// there is no delegation
			continue
		}

		if pRepCandidates[dg.Address] == nil {
			// there is no P-Rep
			continue
		}
		reward := calculateDelegationReward(&dg.Delegate, ia.BlockHeight, blockHeight, gvList, pRepCandidates[dg.Address])

		// update totalReward
		totalReward.Add(&totalReward.Int, &reward.Int)
	}

	// increase value
	if totalReward.Sign() != 0 {
		ia.IScore.Add(&ia.IScore.Int, &totalReward.Int)
	}

	// update BlockHeight
	ia.BlockHeight = blockHeight

	return true
}

func calculateDB(readDB db.Database, writeDB db.Database, gvList []*GovernanceVariable,
	pRepCandidates map[common.Address]*PRepCandidate, blockHeight uint64, batchCount uint64) (uint64, uint64, []byte) {

	iter, _ := readDB.GetIterator()
	bucket, _ := writeDB.GetBucket(db.PrefixIScore)
	batch, _ := writeDB.GetBatch()
	var entries, count uint64 = 0, 0
	stateHash := make([]byte, 64)

	batch.New()
	iter.New(nil, nil)
	for entries = 0; iter.Next(); entries++ {
		// read
		key := iter.Key()[len(db.PrefixIScore):]
		ia, err := NewIScoreAccountFromBytes(iter.Value())
		if err != nil {
			log.Printf("Can't read data with iterator\n")
			return 0, 0, nil
		}
		ia.Address = *common.NewAddress(key)

		// calculate
		if calculateIScore(ia, gvList, pRepCandidates, blockHeight) == false {
			continue
		}

		//log.Printf("Updated data: %s. I-Score: %s\n", ia.String(), ia.IScore.String())

		if batchCount > 0 {
			batch.Set(iter.Key(), ia.Bytes())

			// write batch to DB
			if entries != 0 && (entries % batchCount) == 0 {
				err = batch.Write()
				if err != nil {
					log.Printf("Failed to write batch\n")
				}
				batch.Reset()
			}
		} else {
			bucket.Set(key, ia.Bytes())
		}

		// update stateHash
		sha3.ShakeSum256(stateHash, ia.BytesForHash())

		count++
	}

	// write batch to DB
	if batchCount > 0 {
		err := batch.Write()
		if err != nil {
			log.Printf("Failed to write batch\n")
		}
		batch.Reset()
	}

	// finalize iterator
	iter.Release()
	err := iter.Error()
	if err != nil {
		log.Printf("There is error while iteration. %+v", err)
	}

	//log.Printf("Calculate %d/%d. stateHash:%v\n", count, entries, stateHash)

	return count, entries, stateHash
}

func (rc *rewardCalculate) preCalculate() {
	iScoreDB := rc.mgr.gOpts.db

	// change calculate DB to query DB
	iScoreDB.toggleAccountDB()

	// close and delete old query DB and open new calculate DB
	iScoreDB.resetCalcDB()
}

func (rc *rewardCalculate) calculate(c ipc.Connection, data []byte) error {
	var req CalculateRequest
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}

	opts := rc.mgr.gOpts
	iScoreDB := opts.db
	blockHeight := req.BlockHeight

	log.Printf("Get calculate message: blockHeight: %d, IISS data path: %s", blockHeight, req.Path)

	if req.BlockHeight <= iScoreDB.info.BlockHeight {
		log.Printf("Calculate message has too low blockHeight(request: %d, RC blockHeight: %d)\n",
			blockHeight, iScoreDB.info.BlockHeight)
		return rc.sendCalcResponse(blockHeight, false, nil)
	}

	startTime := time.Now()

	// Load IISS Data
	header, gvList, prepStatList, txList := LoadIISSData(req.Path, false)
	if header == nil {
		log.Printf("Calculate: Failed to load IISS data\n")
		return rc.sendCalcResponse(blockHeight, false, nil)
	}

	if header.BlockHeight != blockHeight {
		log.Printf("Calculate message hash wrong block height. (request: %d, IISS data: %d)\n",
			blockHeight, header.BlockHeight)
		return rc.sendCalcResponse(blockHeight, false, nil)
	}

	rc.preCalculate()

	// Update GV
	opts.UpdateGovernanceVariable(gvList)

	// Update P-Rep candidate with IISS TX(P-Rep register/unregister)
	opts.UpdatePRepCandidate(txList)

	//
	// Calculate I-Score @ Account DB
	//

	// Update calculate DB with delegate TX
	rc.calculateIISSTX(txList, blockHeight)

	// Update P-Rep reward
	rc.calculateIISSPRepStat(prepStatList)

	var totalCount, totalEntry uint64
	var wait sync.WaitGroup
	wait.Add(iScoreDB.info.DBCount)

	queryDBList := iScoreDB.getQueryDBList()
	calcDBList := iScoreDB.getCalcDBList()
	stateHashList := make([][]byte, iScoreDB.info.DBCount)
	for i, cDB := range calcDBList {
		go func(read db.Database, write db.Database) {
			defer wait.Done()

			var count, entry uint64

			// Update all accounts in the calculate DB
			count, entry, stateHashList[i] = calculateDB(read, write, opts.GV, opts.PRepCandidates, blockHeight, writeBatchCount)

			totalCount += count
			totalEntry += entry
		} (queryDBList[i], cDB)
	}
	wait.Wait()

	// make stateHash
	stateHash := make([]byte, 64)
	for _, hash := range stateHashList {
		sha3.ShakeSum256(stateHash, hash)
	}

	elapsedTime := time.Since(startTime)
	log.Printf("Finish calculation: Duration: %s, block height: %d -> %d, DB: %d, batch: %d, %d for %d entries",
		elapsedTime, opts.db.info.BlockHeight, blockHeight, iScoreDB.info.DBCount, writeBatchCount, totalCount, totalEntry)

	// set blockHeight
	opts.db.SetBlockHeight(blockHeight)

	// send response
	return rc.sendCalcResponse(blockHeight, true, stateHash)
}

func (rc *rewardCalculate) sendCalcResponse(blockHeight uint64, success bool, stateHash []byte) error {
	var resp CalculateResponse
	resp.BlockHeight = blockHeight
	resp.Success = success
	resp.StateHash = stateHash

	return rc.conn.Send(msgCalculate, &resp)
}

// Update I-Score of account in TX list
func (rc *rewardCalculate) calculateIISSTX(txList []*IISSTX, blockHeight uint64) {
	opts := rc.mgr.gOpts
	iDB := opts.db

	for _, tx := range txList {
		switch tx.DataType {
		case TXDataTypeDelegate:
			// get Calculate DB for account
			aDB := iDB.GetCalculateDB(tx.Address)
			bucket, _ := aDB.GetBucket(db.PrefixIScore)

			// update I-Score
			newIA := NewIScoreAccountFromIISS(tx)

			data, _ := bucket.Get(tx.Address.Bytes())
			if data != nil {
				ia, err := NewIScoreAccountFromBytes(data)
				if err != nil {
					log.Printf("Failed to make Account Info. from IISS TX(%s). err=%+v", tx.String(), err)
					break
				}
				if ia.BlockHeight != blockHeight {
					log.Printf("Invalid account Info. from calculate DB(%s)", ia.String())
					break
				}

				// get I-Score to tx.BlockHeight
				calculateIScore(ia, opts.GV, opts.PRepCandidates, tx.BlockHeight)

				newIA.IScore = ia.IScore
			}

			// calculate I-Score to blockHeight with updated delegation Info.
			calculateIScore(newIA, opts.GV, opts.PRepCandidates, blockHeight)

			// write to account DB
			bucket.Set(newIA.ID(), newIA.Bytes())
		case TXDataTypePrepReg:
		case TXDataTypePrepUnReg:
		}
	}
}

func (rc *rewardCalculate) calculateIISSPRepStat(prepStatList []*IISSPRepStat) {
	prepMap := make(map[common.Address]common.HexInt)
	var genReward common.HexInt
	for _, pRepStat := range prepStatList {
		// update Generator reward
		generator := prepMap[pRepStat.Generator]
		generator.Add(&generator.Int, &genReward.Int)

		// update Validator reward
		var valReward common.HexInt
		for _, v := range pRepStat.Validator {
			validator := prepMap[v]
			validator.Add(&validator.Int, &valReward.Int)
		}
	}

	for addr, reward := range prepMap {
		iDB := rc.mgr.gOpts.db

		// get Account DB for account
		aDB := iDB.GetCalculateDB(addr)
		bucket, _ := aDB.GetBucket(db.PrefixIScore)

		// update IScoreAccount
		var ia  *IScoreAccount
		data, _ := bucket.Get(addr.Bytes())
		if data != nil {
			ia, err := NewIScoreAccountFromBytes(data)
			if err != nil {
				log.Printf("Failed to make Account Info. for P-Rep reward(%s). err=%+v", addr.String(), err)
				break
			}

			// update I-Score
			ia.IScore.Add(&ia.IScore.Int, &reward.Int)

			// do not update block height
		} else {
			// there is no account in DB
			ia = new(IScoreAccount)
			ia.IScore = reward
			ia.Address = addr
			ia.BlockHeight = 0
		}

		// write to account DB
		bucket.Set(ia.ID(), ia.Bytes())
	}

}
