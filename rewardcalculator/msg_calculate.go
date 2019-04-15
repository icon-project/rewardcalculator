package rewardcalculator

import (
	"log"
	"os"
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

		//log.Printf("[Delegation reward] Read data: %s\n", ia.String())

		// calculate
		if calculateIScore(ia, gvList, pRepCandidates, blockHeight) == false {
			continue
		}

		//log.Printf("[Delegation reward] Updated data: %s\n", ia.String())

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

func preCalculate(ctx *Context) {
	iScoreDB := ctx.db

	// change calculate DB to query DB
	iScoreDB.toggleAccountDB()

	// close and delete old query DB and open new calculate DB
	iScoreDB.resetCalcDB()
}

func (mh *msgHandler) calculate(c ipc.Connection, id uint32, data []byte) error {
	var req CalculateRequest
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}

	success, blockHeight, stateHash := DoCalculate(mh.mgr.ctx, &req)

	// remove IISS data DB
	if success == true {
		os.RemoveAll(req.Path)
	} else {
		os.Rename(req.Path, req.Path + "_failed")
	}

	// send response
	var resp CalculateResponse
	resp.BlockHeight = blockHeight
	resp.Success = success
	resp.StateHash = stateHash

	return c.Send(msgCalculate, id, &resp)
}

func DoCalculate(ctx *Context, req *CalculateRequest) (bool, uint64, []byte){
	iScoreDB := ctx.db
	blockHeight := req.BlockHeight

	log.Printf("Get calculate message: blockHeight: %d, IISS data path: %s", blockHeight, req.Path)

	startTime := time.Now()

	// Load IISS Data
	header, gvList, bpInfoList, prepList, txList := LoadIISSData(req.Path, false)

	// set block height
	if blockHeight == 0 {
		if header != nil {
			blockHeight = header.BlockHeight
		} else {
			blockHeight = iScoreDB.info.BlockHeight + 1
		}
	}

	if blockHeight != 0 && blockHeight <= iScoreDB.info.BlockHeight {
		log.Printf("Calculate message has too low blockHeight(request: %d, RC blockHeight: %d)\n",
			blockHeight, iScoreDB.info.BlockHeight)
		return false, blockHeight, nil
	}

	preCalculate(ctx)

	// Update GV
	ctx.UpdateGovernanceVariable(gvList)

	// Update Main/Sub P-Rep list
	ctx.UpdatePRep(prepList)

	// Update P-Rep candidate with IISS TX(P-Rep register/unregister)
	ctx.UpdatePRepCandidate(txList)

	//
	// Calculate I-Score @ Account DB
	//

	// calculate deletation reward
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
			count, entry, stateHashList[i] = calculateDB(read, write, ctx.GV, ctx.PRepCandidates, blockHeight, writeBatchCount)

			totalCount += count
			totalEntry += entry
		} (queryDBList[i], cDB)
	}
	wait.Wait()

	// Update calculate DB with delegate TX
	calculateIISSTX(ctx, txList, blockHeight)

	// Update block produce reward
	calculateIISSBlockProduce(ctx, bpInfoList, blockHeight)

	// Update P-Rep reward
	calculatePRepReward(ctx, blockHeight)

	// make stateHash
	stateHash := make([]byte, 64)
	for _, hash := range stateHashList {
		sha3.ShakeSum256(stateHash, hash)
	}

	elapsedTime := time.Since(startTime)
	log.Printf("Finish calculation: Duration: %s, block height: %d -> %d, DB: %d, batch: %d, %d for %d entries",
		elapsedTime, ctx.db.info.BlockHeight, blockHeight, iScoreDB.info.DBCount, writeBatchCount, totalCount, totalEntry)

	// set blockHeight
	ctx.db.setBlockHeight(blockHeight)

	return true, blockHeight, stateHash
}

// Update I-Score of account in TX list
func calculateIISSTX(ctx *Context, txList []*IISSTX, blockHeight uint64) {
	for _, tx := range txList {
		switch tx.DataType {
		case TXDataTypeDelegate:
			// get Calculate DB for account
			cDB := ctx.db.getCalculateDB(tx.Address)
			bucket, _ := cDB.GetBucket(db.PrefixIScore)

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

				// backup original I-Score that calculated to blockHeight
				newIA.IScore.Set(&ia.IScore.Int)

				// calculated I-Score from tx.BlockHeight to blockHeight with old delegation Info
				ia.BlockHeight = tx.BlockHeight
				ia.IScore.SetUint64(0)
				calculateIScore(ia, ctx.GV, ctx.PRepCandidates, blockHeight)

				// reset I-Score to tx.BlockHeight
				newIA.IScore.Sub(&newIA.IScore.Int, &ia.IScore.Int)
			}

			// calculate I-Score from tx.BlockHeight to blockHeight with new delegation Info.
			calculateIScore(newIA, ctx.GV, ctx.PRepCandidates, blockHeight)

			// write to account DB
			bucket.Set(newIA.ID(), newIA.Bytes())
		case TXDataTypePrepReg:
		case TXDataTypePrepUnReg:
		}
	}
}

// Calculate Block produce reward
func calculateIISSBlockProduce(ctx *Context, bpInfoList []*IISSBlockProduceInfo, blockHeight uint64) {
	bpMap := make(map[common.Address]common.HexInt)

	// calculate reward
	for _, bpInfo := range bpInfoList {
		// get Governance variable for block height
		gv := ctx.getGV(bpInfo.BlockHeight)

		// update Generator reward
		generator := bpMap[bpInfo.Generator]
		generator.Add(&generator.Int, &gv.blockProduceReward.Int)
		bpMap[bpInfo.Generator] = generator

		// set block validator reward value
		var valReward common.HexInt
		valCount := common.NewHexInt(int64(len(bpInfo.Validator)))
		valReward.Div(&gv.blockProduceReward.Int, &valCount.Int)

		// update Validator reward
		for _, v := range bpInfo.Validator {
			validator := bpMap[v]
			validator.Add(&validator.Int, &valReward.Int)
			bpMap[v] = validator
		}
	}

	// write to account DB
	for addr, reward := range bpMap {
		// get Account DB for account
		cDB := ctx.db.getCalculateDB(addr)
		bucket, _ := cDB.GetBucket(db.PrefixIScore)

		// update IScoreAccount
		var ia  *IScoreAccount
		var err error
		data, _ := bucket.Get(addr.Bytes())
		if data != nil {
			ia, err = NewIScoreAccountFromBytes(data)
			if err != nil {
				log.Printf("Failed to make Account Info. for Block produce reward(%s). err=%+v", addr.String(), err)
				break
			}

			// update I-Score
			ia.IScore.Add(&ia.IScore.Int, &reward.Int)
			ia.Address = addr
			//log.Printf("Block produce reward: %s, %s", ia.String(), reward.String())

			// do not update block height of IA
		} else {
			// there is no account in DB
			ia = new(IScoreAccount)
			ia.IScore.Set(&reward.Int)
			ia.Address = addr
			ia.BlockHeight = blockHeight	// has no delegation. Set blockHeight to blocHeight of calculation msg
		}

		// write to account DB
		if ia != nil {
			bucket.Set(ia.ID(), ia.Bytes())
		}
	}
}

// Calculate Main/Sub P-Rep reward
func calculatePRepReward(ctx *Context, to uint64) {
	start := ctx.db.info.BlockHeight
	end := to

	// calculate for PRep list
	for i, prep := range ctx.PRep {
		var s, e = start, end

		if s < prep.BlockHeight {
			s = prep.BlockHeight
		}
		if i+1 < len(ctx.PRep) && ctx.PRep[i+1].BlockHeight < to {
			e = ctx.PRep[i+1].BlockHeight
		}

		if e  <= s {
			continue
		}

		// calculate P-Rep reward for Governance variable and write to calculate DB
		setPRepReward(ctx, s, e, prep)
	}
}

func setPRepReward(ctx *Context, start uint64, end uint64, prep *PRep) {
	totalReward := make([]common.HexInt, len(prep.List))

	// calculate P-Rep reward for Governance variable
	for i, gv := range ctx.GV {
		var s, e  = start, end

		if s < gv.BlockHeight {
			s = gv.BlockHeight
		}
		if i+1 < len(ctx.GV) && ctx.GV[i+1].BlockHeight < end {
			e = ctx.GV[i+1].BlockHeight
		}

		if e <= s {
			break
		}
		period := common.NewHexIntFromUint64(e - s)

		// reward = period * GV
		var rewardRate common.HexInt
		rewardRate.Mul(&period.Int, &gv.pRepReward.Int)

		// update totalReward
		for i, dgInfo:= range prep.List {
			var reward common.HexInt
			reward.Mul(&rewardRate.Int, &dgInfo.DelegatedAmount.Int)
			reward.Div(&reward.Int, &prep.TotalDelegation.Int)
			totalReward[i].Add(&totalReward[i].Int, &reward.Int)
			//log.Printf("P-Reps: %s, deletation: %s, reward: %s\n",
			//	dgInfo.Address.String(), dgInfo.DelegatedAmount.String(), totalReward[i].String())
		}
	}

	// write to account DB
	for i, dgInfo := range prep.List {
		// get Account DB for account
		cDB := ctx.db.getCalculateDB(dgInfo.Address)
		bucket, _ := cDB.GetBucket(db.PrefixIScore)

		// update IScoreAccount
		var ia  *IScoreAccount
		var err error
		data, _ := bucket.Get(dgInfo.Address.Bytes())
		if data != nil {
			ia, err = NewIScoreAccountFromBytes(data)
			if err != nil {
				log.Printf("Failed to make Account Info. for P-Rep reward(%s). err=%+v", dgInfo.Address.String(), err)
				break
			}

			// update I-Score
			ia.IScore.Add(&ia.IScore.Int, &totalReward[i].Int)
			ia.Address = dgInfo.Address
			//log.Printf("P-Rep reward: %s, %s", ia.String(), totalReward[i].String())

			// do not update block height of IA
		} else {
			// there is no account in DB
			ia = new(IScoreAccount)
			ia.IScore.Set(&totalReward[i].Int)
			ia.Address = dgInfo.Address
			ia.BlockHeight = end // Set blockHeight to end
		}

		// write to account DB
		if ia != nil {
			bucket.Set(ia.ID(), ia.Bytes())
		}
	}
}
