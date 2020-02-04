package core

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/crypto/sha3"
)

const (
	writeBatchCount = 10

	minRewardRep = 200

	blocksPerYear    = 15552000
	gvDivider        = 10000
	iScoreMultiplier = 1000
	rewardDivider    = blocksPerYear * gvDivider / iScoreMultiplier

	MinDelegation = blocksPerYear / iScoreMultiplier * (gvDivider / minRewardRep)
)

var BigIntRewardDivider = big.NewInt(rewardDivider)

type CalculateRequest struct {
	Path        string
	BlockHeight uint64
	BlockHash   []byte
}

func (cr *CalculateRequest) String() string {
	return fmt.Sprintf("Path: %s, BlockHeight: %d", cr.Path, cr.BlockHeight)
}

type CalculateResponse struct {
	Status      uint16
	BlockHeight uint64
}

const (
	CalcRespStatusOK          uint16 = 0
	CalcRespStatusInvalidData uint16 = 1
	CalcRespStatusDoing       uint16 = 2
	CalcRespStatusInvalidBH   uint16 = 3
	CalcRespStatusDuplicateBH uint16 = 4
)

func CalcRespStatusToString(status uint16) string {
	switch status {
	case CalcRespStatusOK:
		return "OK"
	case CalcRespStatusInvalidData:
		return "Invalid IISS data"
	case CalcRespStatusDoing:
		return "Calculating"
	case CalcRespStatusInvalidBH:
		return "Invalid block height"
	case CalcRespStatusDuplicateBH:
		return "Duplicate block height"
	default:
		return "Unknown status"
	}
}

func (cr *CalculateResponse) String() string {
	return fmt.Sprintf("status: %s, BlockHeight: %d", CalcRespStatusToString(cr.Status), cr.BlockHeight)
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

func calculateDelegationReward(ctx *Context, delegationInfo *DelegateData, start uint64, end uint64,
	pRep *PRepCandidate, rewardAddress common.Address) *common.HexInt {
	// adjust start and end with P-Rep candidate
	if start < pRep.Start {
		start = pRep.Start
	}

	if pRep.End != 0 && pRep.End < end {
		end = pRep.End
	}

	total := new(common.HexInt)

	// period in gv
	for i, gv := range ctx.GV {
		var s, e = start, end

		if start < gv.BlockHeight {
			s = gv.BlockHeight
		}
		if i+1 < len(ctx.GV) && ctx.GV[i+1].BlockHeight < end {
			e = ctx.GV[i+1].BlockHeight
		}

		if e <= s {
			continue
		}
		period := common.NewHexIntFromUint64(e - s)

		// reward = delegation amount * period * GV / rewardDivider
		var reward common.HexInt
		reward.Mul(&delegationInfo.Delegate.Int, &period.Int)
		reward.Mul(&reward.Int, &gv.RewardRep.Int)
		reward.Div(&reward.Int, BigIntRewardDivider)

		// update total
		total.Add(&total.Int, &reward.Int)
		WriteBeta3Info(ctx, rewardAddress, gv.RewardRep.Uint64(), delegationInfo, period.Uint64(), e)
	}

	return total
}

func calculateIScore(ctx *Context, ia *IScoreAccount, blockHeight uint64) (bool, *common.HexInt) {
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

		_, ok := ctx.PRepCandidates[dg.Address]
		if ok == false {
			// there is no P-Rep
			continue
		}

		reward := calculateDelegationReward(ctx, dg, ia.BlockHeight,
			blockHeight, ctx.PRepCandidates[dg.Address], ia.Address)

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

func calculateDB(quit <-chan struct{}, index int, readDB db.Database, writeDB db.Database, ctx *Context, blockHeight uint64, batchCount uint64) (uint64, *Statistics, []byte) {

	iter, _ := readDB.GetIterator()
	bucket, _ := writeDB.GetBucket(db.PrefixIScore)
	batch, _ := writeDB.GetBatch()
	var entries, count uint64 = 0, 0
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)
	stats := new(Statistics)
	checkInterrupt := false

	batch.New()
	iter.New(nil, nil)
	for entries = 0; iter.Next(); entries++ {
		// check quit message
		select {
		case <-quit:
			checkInterrupt = true
		default:
		}

		if checkInterrupt {
			break
		}

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

		if needToUpdateCalcDebugResult(ctx) {
			if len(ctx.calcDebugConf.Addresses) > len(ctx.calcDebugResult.Results) {
				initCalcDebugIScores(ctx, *ia, blockHeight)
			}
		}

		// calculate
		ok, reward := calculateIScore(ctx, ia, blockHeight)
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
		h.Write(ia.BytesForHash())

		// update Statistics
		stats.Increase("Beta3", *reward)

		count++
	}
	// finalize iterator
	iter.Release()
	err := iter.Error()
	if err != nil {
		log.Printf("There is error while calculate iteration. %+v", err)
	}

	if checkInterrupt != true {
		// write batch to DB
		if batchCount > 0 {
			err := batch.Write()
			if err != nil {
				log.Printf("Failed to write batch\n")
			}
			batch.Reset()
		}

		// get stateHash if there is update
		if count > 0 {
			h.Read(stateHash)
		}

		log.Printf("Calculate %d: %s, stateHash: %v", index, stats.Beta3.String(), hex.EncodeToString(stateHash))

		return count, stats, stateHash
	} else {
		batch.Reset()
		h.Reset()
		log.Printf("Quit calculate %d with signal", index)
		return 0, stats, nil
	}
}

func sendCalculateACK(c ipc.Connection, id uint32, status uint16, blockHeight uint64) error {
	if c != nil {
		response := CalculateResponse{Status: status, BlockHeight: blockHeight}
		log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgCalculate), id, response.String())
		if err := c.Send(MsgCalculate, id, response); err != nil {
			return err
		}
	}
	return nil
}

func (mh *msgHandler) calculate(c ipc.Connection, id uint32, data []byte) error {
	success := true
	var req CalculateRequest
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}
	log.Printf("\t CALCULATE request: %s", req.String())

	ctx := mh.mgr.ctx
	rollback := ctx.Rollback.GetChannel()

	// do calculation
	err, blockHeight, stats, stateHash := DoCalculate(rollback, ctx, &req, c, id)

	// manage IISS data DB
	if err == nil {
		cleanupIISSData(req.Path)
	} else {
		log.Printf("Failed to calculate. %v", err)
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

func DoCalculate(quit <-chan struct{}, ctx *Context, req *CalculateRequest, c ipc.Connection, id uint32) (error, uint64, *Statistics, []byte) {
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)
	stats := new(Statistics)
	reload := isReloadRequest(req.BlockHeight, id)
	var newAccount uint64

	iScoreDB := ctx.DB
	blockHeight := req.BlockHeight

	log.Printf("Get calculate message: blockHeight: %d, IISS data path: %s", blockHeight, req.Path)
	if !reload && ctx.DB.isCalculating() {
		// send response of CALCULATE
		sendCalculateACK(c, id, CalcRespStatusDoing, blockHeight)
		err := fmt.Errorf("calculating now. drop calculate message. blockHeight: %d, IISS data path: %s",
			blockHeight, req.Path)
		return err, blockHeight, nil, nil
	}
	ctx.DB.setCalculatingBH(req.BlockHeight)

	startTime := time.Now()

	// open IISS Data
	iissDB := OpenIISSData(req.Path)
	defer iissDB.Close()

	// Load IISS data - Header, Governance variable, P-Rep list
	header, gvList, prepList := LoadIISSData(iissDB)
	if header == nil {
		sendCalculateACK(c, id, CalcRespStatusInvalidData, blockHeight)
		err := fmt.Errorf("Failed to load IISS data (path: %s)\n", req.Path)
		ctx.DB.resetCalculatingBH()
		return err, blockHeight, nil, nil
	}

	// set block height
	blockHeight = header.BlockHeight
	if blockHeight == 0 {
		blockHeight = iScoreDB.getCalcDoneBH() + 1
	}

	// check blockHeight and blockHash
	calcDoneBH := iScoreDB.getCalcDoneBH()
	if blockHeight == calcDoneBH {
		sendCalculateACK(c, id, CalcRespStatusDuplicateBH, blockHeight)
		err := fmt.Errorf("duplicated block(height: %d, hash: %s)\n",
			blockHeight, hex.EncodeToString(req.BlockHash))
		ctx.DB.resetCalculatingBH()
		return err, blockHeight, nil, nil
	}
	if blockHeight < calcDoneBH {
		sendCalculateACK(c, id, CalcRespStatusInvalidBH, blockHeight)
		err := fmt.Errorf("too low blockHeight(request: %d, RC blockHeight: %d)\n",
			blockHeight, calcDoneBH)
		ctx.DB.resetCalculatingBH()
		return err, blockHeight, nil, nil
	}

	// set toggle block height with Term start block height
	ctx.DB.toggleAccountDB(blockHeight + 1)

	// send response of CALCULATE after toggle DB
	sendCalculateACK(c, id, CalcRespStatusOK, blockHeight)

	// close and backup old query DB and open new calculate DB
	ctx.DB.resetAccountDB(blockHeight, ctx.DB.getCalcDoneBH())

	// Update header Info.
	if header != nil {
		ctx.Revision = header.Revision
	}

	// Update GV
	ctx.UpdateGovernanceVariable(gvList)

	// Update Main/Sub P-Rep list
	ctx.UpdatePRep(prepList)

	// Update P-Rep candidate with IISS TX(P-Rep register/unregister)
	ctx.UpdatePRepCandidate(iissDB)

	ctx.Print()

	initDebugResult(ctx, blockHeight, req.BlockHash)

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
		go func(ch <-chan struct{}, index int, read db.Database, write db.Database) {
			defer wait.Done()

			var count uint64

			// Update all Accounts in the calculate DB
			count, statsList[index], stateHashList[index] =
				calculateDB(ch, index, read, write, ctx, blockHeight, writeBatchCount)

			totalCount += count
		}(quit, i, queryDBList[i], cDB)
	}
	wait.Wait()

	if quit != ctx.Rollback.GetChannel() {
		return &CalcCancelByRollbackError{blockHeight}, blockHeight, nil, nil
	}

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
	newAccount, reward, hashValue = calculateIISSTX(ctx, iissDB, blockHeight, false)
	stats.Increase("Accounts", newAccount)
	stats.Increase("Beta3", *reward)
	stats.Increase("TotalReward", *reward)
	h.Write(hashValue)

	// Update block produce reward
	newAccount, reward, hashValue = calculateIISSBlockProduce(ctx, iissDB, blockHeight, false)
	stats.Increase("Accounts", newAccount)
	stats.Increase("Beta1", *reward)
	stats.Increase("TotalReward", *reward)
	h.Write(hashValue)

	// Update P-Rep delegated reward
	newAccount, reward, hashValue = calculatePRepReward(ctx, blockHeight)
	stats.Increase("Accounts", newAccount)
	stats.Increase("Beta2", *reward)
	stats.Increase("TotalReward", *reward)
	h.Write(hashValue)

	ctx.stats = stats

	// make stateHash
	for _, hash := range stateHashList {
		h.Write(hash)
	}
	h.Read(stateHash)

	elapsedTime := time.Since(startTime)
	log.Printf("Finish calculation: Duration: %s, block height: %d -> %d, DB: %d, batch: %d, %d entries",
		elapsedTime, ctx.DB.getCalcDoneBH(), blockHeight, iScoreDB.info.DBCount, writeBatchCount, totalCount)
	log.Printf("%s", stats.String())
	log.Printf("stateHash : %s", hex.EncodeToString(stateHash))

	if needToUpdateCalcDebugResult(ctx) {
		log.Printf("CalculationResult : %s", ctx.calcDebugResult.String())
		writeResultToFile(ctx)
		writeCalcDebugOutput(ctx)
		resetCalcResults(ctx)
	}

	// set blockHeight
	ctx.DB.setCalcDoneBH(blockHeight)

	// write calculation result
	WriteCalculationResult(ctx.DB.getCalculateResultDB(), blockHeight, stats, stateHash)

	return nil, blockHeight, ctx.stats, stateHash
}

// Update I-Score of account in TX list
func calculateIISSTX(ctx *Context, iissDB db.Database, blockHeight uint64, verbose bool) (
	uint64, *common.HexInt, []byte) {
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)
	stats := new(common.HexInt)
	var tx IISSTX
	var entries, newAccount uint64 = 0, 0

	iter, _ := iissDB.GetIterator()
	prefix := util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)
	for entries = 0; iter.Next(); entries++ {
		err := tx.SetBytes(iter.Value())
		if err != nil {
			log.Printf("Failed to load IISS TX data")
			continue
		}
		if verbose {
			tx.Index = common.BytesToUint64(iter.Key()[len(db.PrefixIISSTX):])
			log.Printf("[IISSTX] TX %d : %s", entries, tx.String())
		}
		switch tx.DataType {
		case TXDataTypeDelegate:
			// get Calculate DB for account
			cDB := ctx.DB.getCalculateDB(tx.Address)
			bucket, _ := cDB.GetBucket(db.PrefixIScore)

			// update I-Score
			newIA := NewIScoreAccountFromIISS(&tx)

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
				calculateIScore(ctx, ia, blockHeight)

				// reset I-Score to tx.BlockHeight
				newIA.IScore.Sub(&newIA.IScore.Int, &ia.IScore.Int)

				// Statistics
				stats.Sub(&stats.Int, &ia.IScore.Int)
			} else {
				newAccount++
			}

			// calculate I-Score from tx.BlockHeight to blockHeight with new delegation Info.
			ok, reward := calculateIScore(ctx, newIA, blockHeight)
			// Statistics
			if ok == true {
				stats.Add(&stats.Int, &reward.Int)
			}

			if verbose {
				log.Printf("[IISSTX] %s", newIA.String())
			}

			// write to account DB
			bucket.Set(newIA.ID(), newIA.Bytes())

			// update stateHash
			h.Write(newIA.BytesForHash())
		case TXDataTypePrepReg:
		case TXDataTypePrepUnReg:
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		log.Printf("There is error while calculate IISS TX iteration. %+v", err)
	}

	// get stateHash
	h.Read(stateHash)

	log.Printf("IISS TX: TX count: %d, new account: %d, I-Score: %s, stateHash: %s",
		entries, newAccount, stats.String(), hex.EncodeToString(stateHash))

	return newAccount, stats, stateHash
}

// Calculate Block produce reward
func calculateIISSBlockProduce(ctx *Context, iissDB db.Database, blockHeight uint64, verbose bool) (
	uint64, *common.HexInt, []byte) {
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)
	bpMap := make(map[common.Address]common.HexInt)

	// calculate reward
	var bp IISSBlockProduceInfo
	var entries, newAccount uint64
	iter, _ := iissDB.GetIterator()
	prefix := util.BytesPrefix([]byte(db.PrefixIISSBPInfo))
	iter.New(prefix.Start, prefix.Limit)
	for entries = 0; iter.Next(); entries++ {
		err := bp.SetBytes(iter.Value())
		if err != nil {
			log.Printf("Failed to load IISS Block Produce information.")
			continue
		}
		bp.BlockHeight = common.BytesToUint64(iter.Key()[len(db.PrefixIISSBPInfo):])
		if verbose {
			log.Printf("[IISS BP] %d: %s", entries, bp.String())
		}

		// get Governance variable
		gv := ctx.getGVByBlockHeight(bp.BlockHeight)
		if gv == nil {
			continue
		}

		// update Generator reward
		generator := bpMap[bp.Generator]
		generator.Add(&generator.Int, &gv.BlockProduceReward.Int)
		bpMap[bp.Generator] = generator

		// set block validator reward value
		if len(bp.Validator) == 0 {
			continue
		}

		var valReward common.HexInt
		valCount := common.NewHexInt(int64(len(bp.Validator)))
		valReward.Div(&gv.BlockProduceReward.Int, &valCount.Int)

		// update Validator reward
		for _, v := range bp.Validator {
			validator := bpMap[v]
			validator.Add(&validator.Int, &valReward.Int)
			bpMap[v] = validator
		}
		WriteBeta1Info(ctx, gv.BlockProduceReward.Uint64(), bp)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		log.Printf("There is error while calculate IISS BP iteration. %+v", err)
	}

	totalReward := new(common.HexInt)
	iaSlice := make([]*IScoreAccount, 0)

	// write to account DB
	for addr, reward := range bpMap {
		// get Account DB for account
		cDB := ctx.DB.getCalculateDB(addr)
		bucket, _ := cDB.GetBucket(db.PrefixIScore)

		// update IScoreAccount
		var ia *IScoreAccount
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
			ia.BlockHeight = blockHeight // has no delegation. Set blockHeight to blocHeight of calculation msg

			newAccount++
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
		h.Write(ia.BytesForHash())
	}
	h.Read(stateHash)

	log.Printf("IISS Block produce: BP count: %d, new account: %d, I-Score: %s, stateHash: %s",
		entries, newAccount, totalReward.String(), hex.EncodeToString(stateHash))

	return newAccount, totalReward, stateHash
}

// Calculate Main/Sub P-Rep reward
func calculatePRepReward(ctx *Context, to uint64) (uint64, *common.HexInt, []byte) {
	h := sha3.NewShake256()
	stateHash := make([]byte, 64)
	start := ctx.DB.getCalcDoneBH()
	end := to

	totalReward := new(common.HexInt)
	var newAccount uint64

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
		if i+1 < len(ctx.PRep) && ctx.PRep[i+1].BlockHeight < end {
			e = ctx.PRep[i+1].BlockHeight
		}
		//log.Printf("[P-Rep reward] : s, e : %d - %d", s, e)

		if e <= s {
			continue
		}

		// calculate P-Rep reward for Governance variable and write to calculate DB
		account, reward, hash := setPRepReward(ctx, s, e, prep, to)
		h.Write(hash)
		totalReward.Add(&totalReward.Int, &reward.Int)
		newAccount += account
	}

	// get stateHash
	h.Read(stateHash)

	return newAccount, totalReward, stateHash
}

func setPRepReward(ctx *Context, start uint64, end uint64, prep *PRep, blockHeight uint64) (uint64, *common.HexInt, []byte) {
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
		var s, e = start, end

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
		for i, dgInfo := range prep.List {
			var iScore common.HexInt
			iScore.Mul(&rewardRate.Int, &dgInfo.DelegatedAmount.Int)
			iScore.Div(&iScore.Int, &prep.TotalDelegation.Int)
			rewards[i].iScore.Add(&rewards[i].iScore.Int, &iScore.Int)
			rewards[i].blockHeight = e
			//log.Printf("[P-Rep reward] delegation: %s, reward: %s,%d\n",
			//	dgInfo.String(), rewards[i].IScore.String(), rewards[i].blockHeight)
			WriteBeta2Info(ctx, dgInfo, *prep, s, e, gv.PRepReward.Uint64())
		}
	}

	totalReward := new(common.HexInt)
	var newAccount uint64

	// write to account DB
	for i, dgInfo := range prep.List {
		// get Account DB for account
		cDB := ctx.DB.getCalculateDB(dgInfo.Address)
		bucket, _ := cDB.GetBucket(db.PrefixIScore)

		// update IScoreAccount
		var ia *IScoreAccount
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
			// do not update block height of IA
			if ctx.Revision < Revision8 {
				ia.BlockHeight = rewards[i].blockHeight
			}
		} else {
			// there is no account in DB
			ia = new(IScoreAccount)
			ia.IScore.Set(&rewards[i].iScore.Int)
			if ctx.Revision >= Revision8 {
				ia.BlockHeight = blockHeight
			} else {
				ia.BlockHeight = end // Set blockHeight to end
			}

			newAccount++
		}

		// write to account DB
		if ia != nil {
			ia.Address = dgInfo.Address
			//log.Printf("[P-Rep reward] Write to DB %s, increased reward: %s", ia.String(), rewards[i].iScore.String())
			bucket.Set(ia.ID(), ia.Bytes())
			h.Write(ia.BytesForHash())
			totalReward.Add(&totalReward.Int, &rewards[i].iScore.Int)
		}
	}

	// get stateHash
	h.Read(stateHash)

	return newAccount, totalReward, stateHash
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
	if ctx.DB.isCalculating() {
		resp.Status = CalculationDoing
		resp.BlockHeight = ctx.DB.getCalculatingBH()
	} else {
		resp.Status = CalculationDone
		resp.BlockHeight = ctx.DB.getCalcDoneBH()
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
	if ctx.DB.isCalculating() {
		if blockHeight == ctx.DB.getCalculatingBH() {
			resp.Status = calcDoing
			return
		}
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

type CalcCancelByRollbackError struct {
	BlockHeight uint64
}

func (e *CalcCancelByRollbackError) Error() string {
	return fmt.Sprintf("CALCULATE(%d) was canceled by ROLLBACK", e.BlockHeight)
}

func isCalcCancelByRollback(err error) bool {
	_, ok := err.(*CalcCancelByRollbackError)
	return ok
}
