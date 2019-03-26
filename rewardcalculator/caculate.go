package rewardcalculator

import (
	"log"
	"sync"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
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

		if e - s <= 0 {
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

	if (blockHeight - ia.BlockHeight) == 0 {
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
	ia.IScore.Add(&ia.IScore.Int, &totalReward.Int)

	// update BlockHeight
	ia.BlockHeight = blockHeight

	return true
}

func calculateDB(dbi db.Database, gvList []*GovernanceVariable,
	pRepCandidates map[common.Address]*PRepCandidate,
	blockHeight uint64, batchCount uint64) (count uint64, entries uint64) {
	bucket, _ := dbi.GetBucket(db.PrefixIScore)
	iter, _ := dbi.GetIterator()
	batch, _ := dbi.GetBatch()
	entries = 0; count = 0

	batch.New()
	iter.New(nil, nil)
	for entries = 0; iter.Next(); entries++ {
		// read
		key := iter.Key()[len(db.PrefixIScore):]
		ia, err := NewIScoreAccountFromBytes(iter.Value())
		if err != nil {
			log.Printf("Can't read data with iterator\n")
			return 0, 0
		}
		ia.Address = *common.NewAddress(key)

		//log.Printf("Read data: %s\n", ia.String())

		// calculate
		if calculateIScore(ia, gvList, pRepCandidates, blockHeight) == false {
			continue
		}

		log.Printf("Updated data: %s\n", ia.String())

		value, _ := ia.Bytes()

		if batchCount > 0 {
			batch.Set(iter.Key(), value)

			// write batch to DB
			if entries != 0 && (entries % batchCount) == 0 {
				err = batch.Write()
				if err != nil {
					log.Printf("Failed to write batch\n")
				}
				batch.Reset()
			}
		} else {
			bucket.Set(key, value)
		}

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

	//log.Printf("Calculate %d entries for prefix %v-%v %d entries\n", count, start, limit, entries)

	return count, entries
}

func (rc *rewardCalculate) preCalculate() error {
	opts := rc.mgr.gOpts

	// change Account DB snapshot
	err := opts.SetAccountDBSnapshot()
	if err != nil {
		log.Printf("Can't set snapshot of account DB. err=%+v\n", err)
		return err
	}

	// reset preCommit and claimMap for claim
	rc.modifyClaimMap()

	return nil
}

func (rc *rewardCalculate) calculate(c ipc.Connection, data []byte) error {
	var req CalculateRequest
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}

	opts := rc.mgr.gOpts
	mdb := opts.db
	blockHeight := req.BlockHeight

	log.Printf("Get calculate message: blockHeight: %d, IISS data path: %s", blockHeight, req.Path)

	error := rc.preCalculate()
	if error != nil {
		log.Printf("Failed to do precalculate process\n")
	}

	// process IISS data TX, GV
	rc.processIISSData(req.Path)

	// get governance variable
	gvList := opts.GetGVList(opts.BlockHeight, blockHeight)
	if len(gvList) == 0 {
		log.Printf("Can't get Governance variables for block (%d, %d)\n", opts.BlockHeight, blockHeight)
		return nil
	}

	// calculate I-Score @ Account DB
	var totalCount, totalEntry uint64
	var wait sync.WaitGroup
	wait.Add(mdb.info.DBCount)

	for _, adb := range mdb.Account {
		go func(accountDB db.Database) {
			defer wait.Done()

			c, e := calculateDB(accountDB, gvList, opts.PRepCandidates, blockHeight, writeBatchCount)

			totalCount += c
			totalEntry += e
		} (adb)
	}
	wait.Wait()

	log.Printf("Total>block height: %d -> %d, worker: %d, batch: %d, calculation %d for %d entries\n",
		opts.BlockHeight, blockHeight, mdb.info.DBCount, writeBatchCount, totalCount, totalEntry)

	opts.BlockHeight = blockHeight

	// write blockInfo to DB
	var bInfo BlockInfo
	bInfo.BlockHeight = blockHeight
	bucket, err := mdb.Global.GetBucket(db.PrefixBlockInfo)
	if err != nil {
		log.Printf("Can't get BlockInfo bucket. err %+v\n", err)
		return nil
	}
	value, _ := bInfo.Bytes()
	bucket.Set(bInfo.ID(), value)


	// TODO make stateHash and send response
	return rc.sendCalcResponse(blockHeight, true, nil)
}

func (rc *rewardCalculate) processIISSData(dbPath string) {
	opts := rc.mgr.gOpts

	// Load IISS Data
	header, gvList, prepStatList, txList := LoadIISSData(dbPath, false)
	if header == nil {
		return
	}

	// Update GV of Global options
	for _, gvIISS := range gvList {
		// there is new GV
		if  len(opts.GV) == 0 || opts.GV[len(opts.GV)-1].BlockHeight < gvIISS.BlockHeight {
			gv :=  NewGVFromIISS(gvIISS)

			// write to memory
			opts.GV = append(opts.GV, gv)

			// write to global DB
			bucket, _ := rc.mgr.gOpts.db.Global.GetBucket(db.PrefixGovernanceVariable)
			value, _ := gv.Bytes()
			bucket.Set(gv.ID(), value)
		}
	}

	// Update I-Score of P-Rep
	rc.updatePRepReward(prepStatList)

	// Update I-Score of account in TX list and update P-Rep Info.
	for _, tx := range txList {
		switch tx.DataType {
		case TXDataTypeDelegate:
			// get Account DB for account
			aDB := opts.GetAccountDB(tx.Address)
			bucket, _ := aDB.GetBucket(db.PrefixIScore)

			// update IScoreAccount
			var ia *IScoreAccount
			var err error
			newIA := NewIScoreAccountFromIISS(tx)
			data, _ := bucket.Get(tx.Address.Bytes())
			if data != nil {
				ia, err = NewIScoreAccountFromBytes(data)
				if err != nil {
					log.Printf("Failed to make Account Info. from IISS TX(%s). err=%+v", tx.String(), err)
					break
				}
				// calculate I-Score to newIA.BlockHeight
				calculateIScore(ia, opts.GV, opts.PRepCandidates, newIA.BlockHeight)

				// update delegation Info.
				ia.Delegations = newIA.Delegations
			} else {
				// there is no account in DB
				ia = newIA
			}
			// write to account DB
			value, _ := ia.Bytes()
			bucket.Set(ia.ID(), value)

		case TXDataTypeClaim:
			// get Account DB for account
			aDB := opts.GetAccountDB(tx.Address)
			bucket, _ := aDB.GetBucket(db.PrefixIScore)

			// update IScoreAccount
			data, _ := bucket.Get(tx.Address.Bytes())
			if data != nil {
				ia, _ := NewIScoreAccountFromBytes(data)
				// set I-Score to 0 and update block height
				ia.IScore.SetUint64(0)
				ia.BlockHeight = tx.BlockHeight

				// write to account DB
				value, _ := ia.Bytes()
				bucket.Set(ia.ID(), value)
			} else {
				log.Printf("Failed to get Account Info for claim. IISS TX=%s\n", tx.String())
			}

		case TXDataTypePrepReg:
			pRep := opts.PRepCandidates[tx.Address]
			if pRep == nil {
				p := new(PRepCandidate)
				p.Address = tx.Address
				p.Start = tx.BlockHeight
				p.End = 0

				// write to memory
				opts.PRepCandidates[tx.Address] = p

				// write to global DB
				bucket, _ := opts.db.Global.GetBucket(db.PrefixPrepCandidate)
				data, _ := p.Bytes()
				bucket.Set(p.ID(), data)
			} else {
				log.Printf("Failed to register PRepCandidate. %s was registered already\n", tx.Address.String())
			}
		case TXDataTypePrepUnReg:
			pRep := opts.PRepCandidates[tx.Address]
			if pRep != nil {
				// write to memory
				pRep.End = tx.BlockHeight

				// write to global DB
				bucket, _ := opts.db.Global.GetBucket(db.PrefixPrepCandidate)
				data, _ := pRep.Bytes()
				bucket.Set(pRep.ID(), data)
			} else {
				log.Printf("Failed to unregister PRepCandidate. %s was not registered\n", tx.Address.String())
			}
		}
	}
}

func (rc *rewardCalculate) sendCalcResponse(blockHeight uint64, success bool, stateHash []byte) error {
	var resp CalculateResponse
	resp.BlockHeight = blockHeight
	resp.Success = success
	resp.StateHash = stateHash

	return rc.conn.Send(msgCalculate, &resp)
}

func (rc *rewardCalculate) updatePRepReward(prepStatList []*IISSPRepStat) {
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
		opts := rc.mgr.gOpts

		// get Account DB for account
		aDB := opts.GetAccountDB(addr)
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
		value, _ := ia.Bytes()
		bucket.Set(ia.ID(), value)
	}

}
