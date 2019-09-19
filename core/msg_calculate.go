package core

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"golang.org/x/crypto/sha3"
)

const (
	writeBatchCount = 10

	minRewardRep  = 200

	blocksPerYear    = 15552000
	gvDivider        = 10000
	iScoreMultiplier = 1000
	rewardDivider    = blocksPerYear * gvDivider / iScoreMultiplier

	MinDelegation = blocksPerYear / iScoreMultiplier * (gvDivider / minRewardRep)
)

var BigIntRewardDivider = big.NewInt(rewardDivider)

type CalculateStatus struct {
	Doing       bool
	BlockHeight uint64
}

func (cs *CalculateStatus) set(start bool, blockHeight uint64) {
	cs.Doing = start
	cs.BlockHeight = blockHeight
}

func (cs *CalculateStatus) reset() {
	cs.set(false, 0)
}

type CalculateRequest struct {
	Path        string
	BlockHeight uint64
}
func (cr *CalculateRequest) String() string {
	return fmt.Sprintf("Path: %s, BlockHeight: %d", cr.Path, cr.BlockHeight)
}

type CalculateDone struct {
	Success     bool
	BlockHeight uint64
	IScore      common.HexInt
	StateHash   []byte
}

func (cd *CalculateDone) String() string {
	return fmt.Sprintf("Success: %s, BlockHeight: %d, IScore: %s, StateHash: %s",
		strconv.FormatBool(cd.Success),
		cd.BlockHeight,
		cd.IScore.String(),
		hex.EncodeToString(cd.StateHash))
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

		// reward = delegation amount * period * GV / rewardDivider
		var reward common.HexInt
		reward.Mul(&delegation.Int, &period.Int)
		reward.Mul(&reward.Int, &gv.RewardRep.Int)
		reward.Div(&reward.Int, BigIntRewardDivider)

		// update total
		total.Add(&total.Int, &reward.Int)
	}

	return total
}

func calculateIScore(ia *IScoreAccount,  gvList []*GovernanceVariable,
	pRepCandidates map[common.Address]*PRepCandidate, blockHeight uint64) (bool, *common.HexInt) {
	//log.Printf("[Delegation reward] Read data: %s\n", ia.String())

	totalReward := common.NewHexIntFromUint64(0)

	// IScore = old + period * G.V * sum(valid dgAmount)
	if blockHeight == 0 {
		blockHeight = ia.BlockHeight + 1
	}

	if blockHeight <= ia.BlockHeight {
		return false, nil
	}

	for _, dg := range ia.Delegations {
		if MinDelegation > dg.Delegate.Uint64() {
			// not enough delegation
			continue
		}

		_, ok := pRepCandidates[dg.Address]
		if ok == false {
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

	//log.Printf("[Delegation reward] Updated data: %s\n", ia.String())
	return true, totalReward
}

func calculateDB(index int, readDB db.Database, writeDB db.Database, revision uint64, gvList []*GovernanceVariable,
	pRepCandidates map[common.Address]*PRepCandidate, blockHeight uint64, batchCount uint64) (uint64, *Statistics, []byte) {

	iter, _ := readDB.GetIterator()
	bucket, _ := writeDB.GetBucket(db.PrefixIScore)
	batch, _ := writeDB.GetBatch()
	var entries, count uint64 = 0, 0
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)
	stats := new(Statistics)

	batch.New()
	iter.New(nil, nil)
	for entries = 0; iter.Next(); entries++ {
		// read
		key := iter.Key()[len(db.PrefixIScore):]
		ia, err := NewIScoreAccountFromBytes(iter.Value())
		if err != nil {
			log.Printf("Can't read data with iterator\n")
			return 0, stats, nil
		}
		ia.Address = *common.NewAddress(key)

		// update Statistics account
		stats.Increase("Accounts", uint64(1))

		// calculate
		ok, reward := calculateIScore(ia, gvList, pRepCandidates, blockHeight)
		if ok == false {
			continue
		}

		if batchCount > 0 {
			batch.Set(iter.Key(), ia.Bytes())

			// write batch to DB
			if entries == batchCount {
				err = batch.Write()
				if err != nil {
					log.Printf("Failed to write batch\n")
				}
				batch.Reset()
				entries = 0
			}
		} else {
			bucket.Set(key, ia.Bytes())
		}

		// update stateHash
		makeHash(revision, h, ia.BytesForHash())

		// update Statistics
		stats.Increase("Beta3", *reward)

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

	// get stateHash if there is update
	if count > 0 {
		h.Read(stateHash)
	}

	log.Printf("Calculate %d: %s, stateHash: %v", index, stats.Beta3.String(), hex.EncodeToString(stateHash))

	return count, stats, stateHash
}

func checkToggle(ctx *Context, blockHeight uint64) bool {
	idb := ctx.DB

	// compare block height of first account of Query/Calc DB
	qDB := idb.getQueryDBList()
	var addr common.Address
	var qBH uint64
	findQueryEntry := false

	// pick first account from Query DB
	for _, q := range qDB {
		iter, _ := q.GetIterator()
		iter.New(nil, nil)
		for ; iter.Next(); {
			key := iter.Key()[len(db.PrefixIScore):]
			ia, err := NewIScoreAccountFromBytes(iter.Value())
			if ia == nil || err != nil {
				continue
			}
			qBH = ia.BlockHeight
			addr.SetBytes(key)
			findQueryEntry = true
			break
		}
	}

	if findQueryEntry == false {
		// There is no data. toggle!!
		return true
	}

	// read account Info. from Calc DB
	cDB := idb.getCalculateDB(addr)
	bucket, _ := cDB.GetBucket(db.PrefixIScore)
	data, _ := bucket.Get(addr.Bytes())
	if data == nil {
		// There is no account in Calc DB. No need to toggle.
		return false
	} else {
		ia, _ := NewIScoreAccountFromBytes(data)
		if ia != nil {
			cBH := ia.BlockHeight
			if qBH == cBH {
				// is it possible?
				return false
			} else if qBH < cBH {
				// Calc DB has updated data.
				if cBH == blockHeight {
					// Calculated for this blockHeight. keep going
					return false
				} else {
					// old data. need toggle
					return true
				}
			} else {
				// is it possible?
				return false
			}
		}
		// calc DB was corrupted, overwrite with calculation
		return false
	}
}

func toggleAccountDB(ctx *Context, blockHeight uint64) {
	if checkToggle(ctx, blockHeight) == false {
		return
	}

	// change calculate DB to query DB
	idb := ctx.DB
	idb.toggleAccountDB()
}

func sendCalculateACK(c ipc.Connection, id uint32) error {
	if c != nil {
		log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgCalculate), id, "ack")
		if err := c.Send(MsgCalculate, id, nil); err != nil {
			return err
		}
	}
	return nil
}

func (mh *msgHandler) calculate(c ipc.Connection, id uint32, data []byte) error {
	var req CalculateRequest
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}
	log.Printf("\t CALCULATE request: %s", req.String())

	// do calculation
	err, blockHeight, stats, stateHash := DoCalculate(mh.mgr.ctx, &req, c, id)

	success := true

	// remove IISS data DB
	if err == nil {
		os.RemoveAll(req.Path)
	} else {
		os.Rename(req.Path, req.Path + "_failed")
		success = false
	}

	// send CALCULATE_DONE
	var resp CalculateDone
	resp.BlockHeight = blockHeight
	resp.Success = success
	if stats != nil {
		resp.IScore.Set(&stats.TotalReward.Int)
	} else {
		resp.IScore.SetUint64(0)
	}
	resp.StateHash = stateHash

	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgCalculateDone), 0, resp.String())
	return c.Send(MsgCalculateDone, 0, &resp)
}

func DoCalculate(ctx *Context, req *CalculateRequest, c ipc.Connection, id uint32) (error, uint64, *Statistics, []byte) {
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)
	stats := new(Statistics)

	iScoreDB := ctx.DB
	blockHeight := req.BlockHeight

	log.Printf("Get calculate message: blockHeight: %d, IISS data path: %s", blockHeight, req.Path)
	if ctx.calculateStatus.Doing {
		// send acknowledge of CALCULATE
		sendCalculateACK(c, id)


		errMsg := fmt.Sprintf("Calculating now. Drop this calculate message. blockHeight: %d, IISS data path: %s",
			blockHeight, req.Path)
		log.Printf(errMsg)
		err := fmt.Errorf(errMsg)

		return err, blockHeight, nil, nil
	}

	ctx.calculateStatus.set(true, req.BlockHeight)
	defer ctx.calculateStatus.reset()
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
		// send acknowledge of CALCULATE
		sendCalculateACK(c, id)

		errMsg := fmt.Sprintf("Calculate message has too low blockHeight(request: %d, RC blockHeight: %d)\n",
			blockHeight, iScoreDB.info.BlockHeight)
		log.Printf(errMsg)
		err := fmt.Errorf(errMsg)
		return err, blockHeight, nil, nil
	}

	toggleAccountDB(ctx, blockHeight)

	// send acknowledge of CALCULATE after toggle DB
	sendCalculateACK(c, id)

	// close and delete old account DB and open new calculate DB
	ctx.DB.resetCalcDB()

	// Update header Info.
	if header != nil {
		ctx.Revision = header.Revision
	}

	// Update GV
	ctx.UpdateGovernanceVariable(gvList)

	// Update Main/Sub P-Rep list
	ctx.UpdatePRep(prepList)

	// Update P-Rep candidate with IISS TX(P-Rep register/unregister)
	ctx.UpdatePRepCandidate(txList)

	ctx.Print()

	//
	// Calculate I-Score @ Account DB
	//

	// calculate delegation reward
	var totalCount uint64
	var wait sync.WaitGroup
	wait.Add(iScoreDB.info.DBCount)

	queryDBList := iScoreDB.getQueryDBList()
	calcDBList := iScoreDB.GetCalcDBList()
	stateHashList := make([][]byte, iScoreDB.info.DBCount)
	statsList := make([]*Statistics, iScoreDB.info.DBCount)
	for i, cDB := range calcDBList {
		go func(index int, read db.Database, write db.Database) {
			defer wait.Done()

			var count uint64

			// Update all Accounts in the calculate DB
			count, statsList[index], stateHashList[index] =
				calculateDB(index, read, write, ctx.Revision, ctx.GV, ctx.PRepCandidates, blockHeight, writeBatchCount)

			totalCount += count
		} (i, queryDBList[i], cDB)
	}
	wait.Wait()

	// update Statistics
	for _, s := range statsList {
		if s == nil {
			continue
		}
		stats.Increase("Accounts", s.Accounts)
		stats.Increase("Beta3", s.Beta3)
		stats.Increase("TotalReward", s.Beta3)
	}

	reward := new(common.HexInt)
	var hashValue []byte

	// Update calculate DB with delegate TX
	reward, hashValue = calculateIISSTX(ctx, txList, blockHeight)
	stats.Increase("Beta3", *reward)
	stats.Increase("TotalReward", *reward)
	makeHash(ctx.Revision, h, hashValue)

	// Update block produce reward
	reward, hashValue = calculateIISSBlockProduce(ctx, bpInfoList, blockHeight)
	stats.Increase("Beta1", *reward)
	stats.Increase("TotalReward", *reward)
	makeHash(ctx.Revision, h, hashValue)

	// Update P-Rep reward
	reward, hashValue = calculatePRepReward(ctx, blockHeight)
	stats.Increase("Beta2", *reward)
	stats.Increase("TotalReward", *reward)
	makeHash(ctx.Revision, h, hashValue)

	ctx.stats = stats

	// make stateHash
	for _, hash := range stateHashList {
		makeHash(ctx.Revision, h, hash)
	}
	h.Read(stateHash)

	elapsedTime := time.Since(startTime)
	log.Printf("Finish calculation: Duration: %s, block height: %d -> %d, DB: %d, batch: %d, %d entries",
		elapsedTime, ctx.DB.info.BlockHeight, blockHeight, iScoreDB.info.DBCount, writeBatchCount, totalCount)
	log.Printf("%s", stats.String())
	log.Printf("stateHash : %s", hex.EncodeToString(stateHash))

	// set blockHeight
	ctx.DB.setBlockHeight(blockHeight)

	// write calculation result
	WriteCalculationResult(ctx.DB.getCalculateResultDB(), blockHeight, stats, stateHash)

	return nil, blockHeight, ctx.stats, stateHash
}

// Update I-Score of account in TX list
func calculateIISSTX(ctx *Context, txList []*IISSTX, blockHeight uint64) (*common.HexInt, []byte) {
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)
	stats := new(common.HexInt)

	for _, tx := range txList {
		//log.Printf("[IISSTX] TX : %s", tx.String())
		switch tx.DataType {
		case TXDataTypeDelegate:
			// get Calculate DB for account
			cDB := ctx.DB.getCalculateDB(tx.Address)
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

				// Statistics
				stats.Sub(&stats.Int, &ia.IScore.Int)
			}

			// calculate I-Score from tx.BlockHeight to blockHeight with new delegation Info.
			ok, reward := calculateIScore(newIA, ctx.GV, ctx.PRepCandidates, blockHeight)
			// Statistics
			if ok == true {
				stats.Add(&stats.Int, &reward.Int)
			}

			//log.Printf("[IISSTX] %s", newIA.String())

			// write to account DB
			bucket.Set(newIA.ID(), newIA.Bytes())

			// update stateHash
			makeHash(ctx.Revision, h, newIA.BytesForHash())
		case TXDataTypePrepReg:
		case TXDataTypePrepUnReg:
		}
	}

	// get stateHash
	h.Read(stateHash)

	return stats, stateHash
}

// Calculate Block produce reward
func calculateIISSBlockProduce(ctx *Context, bpInfoList []*IISSBlockProduceInfo, blockHeight uint64) (*common.HexInt, []byte) {
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)
	bpMap := make(map[common.Address]common.HexInt)

	// calculate reward
	for _, bpInfo := range bpInfoList {
		// get Governance variable
		gv := ctx.getGVByBlockHeight(bpInfo.BlockHeight)
		if gv == nil {
			continue
		}

		// update Generator reward
		generator := bpMap[bpInfo.Generator]
		generator.Add(&generator.Int, &gv.BlockProduceReward.Int)
		bpMap[bpInfo.Generator] = generator

		// set block validator reward value
		if len(bpInfo.Validator) == 0 {
			continue
		}

		var valReward common.HexInt
		valCount := common.NewHexInt(int64(len(bpInfo.Validator)))
		valReward.Div(&gv.BlockProduceReward.Int, &valCount.Int)

		// update Validator reward
		for _, v := range bpInfo.Validator {
			validator := bpMap[v]
			validator.Add(&validator.Int, &valReward.Int)
			bpMap[v] = validator
		}
	}

	totalReward := new(common.HexInt)
	iaSlice := make([]*IScoreAccount, 0)

	// write to account DB
	for addr, reward := range bpMap {
		// get Account DB for account
		cDB := ctx.DB.getCalculateDB(addr)
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

			totalReward.Add(&totalReward.Int, &reward.Int)

			// for state root hash
			iaSlice = append(iaSlice, ia)
		}
	}

	// sort data and make state root hash
	sort.Slice(iaSlice, func(i, j int) bool {
		return iaSlice[i].Compare(iaSlice[j]) < 0
	})
	for _, ia := range iaSlice {
		makeHash(ctx.Revision, h, ia.BytesForHash())
	}
	h.Read(stateHash)

	return totalReward, stateHash
}

// Calculate Main/Sub P-Rep reward
func calculatePRepReward(ctx *Context, to uint64) (*common.HexInt, []byte) {
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)
	start := ctx.DB.info.BlockHeight
	end := to

	totalReward := new(common.HexInt)

	// calculate for PRep list
	for i, prep := range ctx.PRep {
		//log.Printf("[P-Rep reward] P-Rep : %s", prep.String())
		if prep.TotalDelegation.Sign() == 0 {
			// there is no delegations, check next
			continue
		}
		var s, e = start, end

		if s < prep.BlockHeight {
			s = prep.BlockHeight
		}
		if i+1 < len(ctx.PRep) && ctx.PRep[i+1].BlockHeight < to {
			e = ctx.PRep[i+1].BlockHeight
		}
		//log.Printf("[P-Rep reward] : s, e : %d - %d", s, e)

		if e  <= s {
			continue
		}

		// calculate P-Rep reward for Governance variable and write to calculate DB
		reward, hash := setPRepReward(ctx, s, e, prep)
		makeHash(ctx.Revision, h, hash)
		totalReward.Add(&totalReward.Int, &reward.Int)
	}

	// get stateHash
	h.Read(stateHash)

	return totalReward, stateHash
}

func setPRepReward(ctx *Context, start uint64, end uint64, prep *PRep) (*common.HexInt, []byte) {
	type reward struct {
		iScore      common.HexInt
		blockHeight uint64
	}
	rewards := make([]reward, len(prep.List))
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)

	// calculate P-Rep reward for Governance variable
	for i, gv := range ctx.GV {
		//log.Printf("setPRepReward: gv: %s", gv.String())
		var s, e  = start, end

		if s <= gv.BlockHeight {
			s = gv.BlockHeight
		}

		if i+1 < len(ctx.GV) && ctx.GV[i+1].BlockHeight < end {
			e = ctx.GV[i+1].BlockHeight
		}

		//log.Printf("[P-Rep reward]setPRepReward: s, e : %d - %d", s, e)
		if e <= s {
			continue
		}
		period := common.NewHexIntFromUint64(e - s)

		// reward = period * GV
		var rewardRate common.HexInt
		rewardRate.Mul(&period.Int, &gv.PRepReward.Int)

		// update rewards
		for i, dgInfo:= range prep.List {
			var reward common.HexInt
			reward.Mul(&rewardRate.Int, &dgInfo.DelegatedAmount.Int)
			reward.Div(&reward.Int, &prep.TotalDelegation.Int)
			rewards[i].iScore.Add(&rewards[i].iScore.Int, &reward.Int)
			rewards[i].blockHeight = e
			//log.Printf("[P-Rep reward] deletation: %s, reward: %s,%d\n",
			//	dgInfo.String(), rewards[i].IScore.String(), rewards[i].blockHeight)
		}
	}

	totalReward := new(common.HexInt)

	// write to account DB
	for i, dgInfo := range prep.List {
		// get Account DB for account
		cDB := ctx.DB.getCalculateDB(dgInfo.Address)
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
			ia.IScore.Add(&ia.IScore.Int, &rewards[i].iScore.Int)
			ia.BlockHeight = rewards[i].blockHeight

			// do not update block height of IA
		} else {
			// there is no account in DB
			ia = new(IScoreAccount)
			ia.IScore.Set(&rewards[i].iScore.Int)
			ia.BlockHeight = end // Set blockHeight to end
		}

		// write to account DB
		if ia != nil {
			ia.Address = dgInfo.Address
			//log.Printf("[P-Rep reward] Write to DB %s, increased reward: %s", ia.String(), rewards[i].IScore.String())
			bucket.Set(ia.ID(), ia.Bytes())
			makeHash(ctx.Revision, h, ia.BytesForHash())
			totalReward.Add(&totalReward.Int, &rewards[i].iScore.Int)
		}
	}

	// get stateHash
	h.Read(stateHash)

	return totalReward, stateHash
}

func makeHash(revision uint64, h sha3.ShakeHash, data []byte) {
	if revision == IISSDataRevisionDefault {
		h.Reset()	// Reset hash for backward compatibility
		h.Write(data)
	} else {
		h.Write(data)
	}
}

const (
	CalculationDone  uint64 = 0
	CalculationDoing uint64 = 1
)

type QueryCalculateStatusResponse struct {
	Status      uint64
	BlockHeight uint64
}

func (cs *QueryCalculateStatusResponse) StatusString() string {
	switch cs.Status {
	case CalculationDone:
		return "Calculation Done"
	case CalculationDoing:
		return "Calculating"
	default:
		return "Unknown status"
	}
}

func (cs *QueryCalculateStatusResponse) String() string {
	return fmt.Sprintf("Status: %s, BlockHeight: %d", cs.StatusString(), cs.BlockHeight)
}

func (mh *msgHandler) queryCalculateStatus(c ipc.Connection, id uint32, data []byte) error {
	ctx := mh.mgr.ctx

	// send QUERY_CALCULATE_STATUS response
	var resp QueryCalculateStatusResponse

	DoQueryCalculateStatus(ctx, &resp)

	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgQueryCalculateStatus), id, resp.String())
	return c.Send(MsgQueryCalculateStatus, id, &resp)
}

func DoQueryCalculateStatus(ctx *Context, resp *QueryCalculateStatusResponse) {
	if ctx.calculateStatus.Doing {
		resp.Status = CalculationDoing
		resp.BlockHeight = ctx.calculateStatus.BlockHeight
	} else {
		resp.Status = CalculationDone
		resp.BlockHeight = ctx.DB.info.BlockHeight
	}
}

const (
	calcSucceeded uint16 = 0
	calcFailed    uint16 = 1
	calcDoing     uint16 = 2
	InvalidBH     uint16 = 3
)

type QueryCalculateResultResponse struct {
	Status      uint16
	BlockHeight uint64
	IScore      common.HexInt
	StateHash   []byte
}

func (cr *QueryCalculateResultResponse) StatusString() string {
	switch cr.Status {
	case calcSucceeded:
		return "Succeeded"
	case calcFailed:
		return "Failed"
	case calcDoing:
		return "Calculating"
	case InvalidBH:
		return "Invalid block height"
	default:
		return "Unknown status"
	}
}

func (cr *QueryCalculateResultResponse) String() string {
	return fmt.Sprintf("Status: %s, BlockHeight: %d, IScore: %s, StateHash: %s",
		cr.StatusString(),
		cr.BlockHeight,
		cr.IScore.String(),
		hex.EncodeToString(cr.StateHash))
}

func (mh *msgHandler) queryCalculateResult(c ipc.Connection, id uint32, data []byte) error {
	var blockHeight uint64
	if _, err := codec.MP.UnmarshalFromBytes(data, &blockHeight); err != nil {
		log.Printf("Failed to unmarshal data. err=%+v", err)
		return err
	}
	log.Printf("\t Query calculate result : block height : %d", blockHeight)

	ctx := mh.mgr.ctx

	// send QUERY_CALCULATE_RESULT response
	var resp QueryCalculateResultResponse

	DoQueryCalculateResult(ctx, blockHeight, &resp)

	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgQueryCalculateResult), id, resp.String())
	return c.Send(MsgQueryCalculateResult, id, &resp)
}

func DoQueryCalculateResult(ctx *Context, blockHeight uint64, resp *QueryCalculateResultResponse) {
	resp.BlockHeight = blockHeight

	// check doing calculation
	if blockHeight == ctx.calculateStatus.BlockHeight {
		if blockHeight == 0 {
			resp.Status = InvalidBH
		} else {
			resp.Status = calcDoing
		}
		return
	}

	// read from calculate result DB
	crDB := ctx.DB.getCalculateResultDB()
	bucket, _ := crDB.GetBucket(db.PrefixCalcResult)
	key := common.Uint64ToBytes(blockHeight)
	bs, _ := bucket.Get(key)
	if bs != nil {
		cr, _ := NewCalculationResultFromBytes(bs)
		resp.BlockHeight = blockHeight
		if cr.Success {
			resp.Status = calcSucceeded
			resp.IScore.Set(&cr.IScore.Int)
			resp.StateHash = cr.StateHash
		} else {
			resp.Status = calcFailed
		}
	} else {
		// No calculation result
		resp.Status = InvalidBH
	}
}
