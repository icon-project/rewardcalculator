package core

import (
	"encoding/hex"
	"fmt"
	"log"
	"strconv"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	BlockHeightSize = 8
	BlockHashSize   = 32
	TXHashSize      = 32

	PreCommitIDSize = BlockHeightSize + BlockHashSize + common.AddressBytes

	ClaimBackupIDSize = BlockHeightSize + common.AddressBytes
	ClaimBackupPeriod = 43120 * 2 - 1
)

type ClaimData struct {
	BlockHeight   uint64
	IScore        common.HexInt
}

func (cd *ClaimData) String() string {
	return fmt.Sprintf("BlockHeight: %d, IScore: %s", cd.BlockHeight, cd.IScore.String())
}

func (cd *ClaimData) equal(cmpData *ClaimData) bool {
	return cd.IScore.Cmp(&cmpData.IScore.Int) == 0 && cd.BlockHeight == cmpData.BlockHeight
}

type Claim struct {
	Address  common.Address
	Data     ClaimData
}

func (c *Claim) ID() []byte {
	return c.Address.Bytes()
}

func (c *Claim) BackupID(blockHeight uint64) []byte {
	id := make([]byte, ClaimBackupIDSize)

	bh := common.Uint64ToBytes(blockHeight)
	copy(id[BlockHeightSize- len(bh):], bh)
	copy(id[BlockHeightSize:], c.ID())

	return id
}

func (c *Claim) Bytes() []byte {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&c.Data); err != nil {
		log.Panicf("Failed to marshal claim data=%+v. err=%+v", c, err)
		return nil
	} else {
		bytes = bs
	}
	return bytes
}

func (c *Claim) String() string {
	return fmt.Sprintf("Address: %s, %s", c.Address.String(), c.Data.String())
}

func (c *Claim) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &c.Data)
	if err != nil {
		return err
	}
	return nil
}

func NewClaimFromBytes(bs []byte) (*Claim, error) {
	claim := new(Claim)
	if err:= claim.SetBytes(bs); err != nil {
		return nil, err
	} else {
		return claim, nil
	}
}

func ClaimBackupKeyString(key []byte) string {
	blockHeight := common.BytesToUint64(key[:BlockHeightSize])
	address := common.NewAddress(key[BlockHeightSize:])
	return fmt.Sprintf("BlockHeight: %d, Address: %s", blockHeight, address.String())
}

type ClaimBackupInfo struct {
	FirstBlockHeight uint64
	LastBlockHeight uint64
}

func (cb *ClaimBackupInfo) ID() []byte {
	return []byte("")
}

func (cb *ClaimBackupInfo) Bytes() []byte {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&cb); err != nil {
		log.Panicf("Failed to marshal claim backup management data=%+v. err=%+v", cb, err)
		return nil
	} else {
		bytes = bs
	}
	return bytes
}

func (cb *ClaimBackupInfo) String() string {
	return fmt.Sprintf("BlockHeight: %d - %d", cb.FirstBlockHeight, cb.LastBlockHeight)
}

func (cb *ClaimBackupInfo) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &cb)
	if err != nil {
		return err
	}
	return nil
}

type PreCommitData struct {
	Confirmed bool
	TXIndex uint64
	TXHash []byte
	Claim
}

func (pc *PreCommitData) String() string {
	return fmt.Sprintf("TXIndex: %d, TXHash: %s, Confirmed: %s, %s",
		pc.TXIndex,
		hex.EncodeToString(pc.TXHash),
		strconv.FormatBool(pc.Confirmed),
		pc.Claim.String())
}

type PreCommit struct {
	BlockHeight uint64
	BlockHash []byte
	PreCommitData
}

func (pc *PreCommit) ID() []byte {
	id := make([]byte, PreCommitIDSize)

	bh := common.Uint64ToBytes(pc.BlockHeight)
	copy(id[BlockHeightSize-len(bh):], bh)
	copy(id[BlockHeightSize:], pc.BlockHash)
	copy(id[BlockHeightSize+BlockHashSize:], pc.Address.Bytes())

	return id
}

func (pc *PreCommit) SetID(id []byte) {
	pc.BlockHeight = common.BytesToUint64(id[:BlockHeightSize])
	pc.BlockHash = make([]byte, BlockHashSize)
	copy(pc.BlockHash, id[BlockHeightSize:BlockHeightSize+BlockHashSize])
	pc.Address.SetBytes(id[BlockHeightSize+BlockHashSize:])
}

func (pc *PreCommit) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&pc.PreCommitData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (pc *PreCommit) String() string {
	return fmt.Sprintf("BlockHeight: %d, BlockHash: %s, %s",
		pc.BlockHeight,
		hex.EncodeToString(pc.BlockHash),
		pc.PreCommitData.String())
}

func (pc *PreCommit) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &pc.PreCommitData)
	if err != nil {
		return err
	}
	return nil
}

func newPreCommit(blockHeight uint64, blockHash []byte, txIndex uint64, txHash []byte, address common.Address) *PreCommit {
	pc := new(PreCommit)

	pc.BlockHeight = blockHeight
	pc.BlockHash = make([]byte, BlockHashSize)
	copy(pc.BlockHash, blockHash)
	pc.TXIndex = txIndex
	pc.TXHash = make([]byte, TXHashSize)
	copy(pc.TXHash, txHash)
	pc.Address = address

	return pc
}

func (pc *PreCommit) query(pcDB db.Database) bool {
	bucket, _ := pcDB.GetBucket(db.PrefixClaim)
	bs, _ := bucket.Get(pc.ID())
	if bs != nil {
		pc.SetBytes(bs)
		return true
	}

	return false
}

func (pc *PreCommit) write(pcDB db.Database, iScore *common.HexInt) error {
	bucket, _ := pcDB.GetBucket(db.PrefixClaim)
	if iScore != nil {
		pc.Data.BlockHeight = pc.BlockHeight
		pc.Data.IScore.Set(&iScore.Int)
	}
	bs, err := pc.Bytes()
	if err != nil {
		return err
	}

	return bucket.Set(pc.ID(), bs)
}

func (pc *PreCommit) delete(pcDB db.Database) error {
	bucket, _ := pcDB.GetBucket(db.PrefixClaim)
	return bucket.Delete(pc.ID())
}

func (pc *PreCommit) commit(pcDB db.Database) error {
	txIndex := pc.TXIndex
	if pc.query(pcDB) == false {
		return fmt.Errorf("no data to commit")
	}
	if txIndex == pc.TXIndex {
		if pc.Confirmed == true {
			return nil
		}
		pc.Confirmed = true
		return pc.write(pcDB, nil)
	}
	return nil
}

func (pc *PreCommit) revert(pcDB db.Database) error {
	txIndex := pc.TXIndex
	if pc.query(pcDB) == false {
		return fmt.Errorf("no data to revert")
	}
	if txIndex == pc.TXIndex {
		if pc.Confirmed == true {
			return nil
		} else {
			return pc.delete(pcDB)
		}
	}
	return nil
}

func makeIteratorPrefix(prefix db.BucketID, blockHeight uint64, data []byte, dataSize int) *util.Range {
	bsSize := len(prefix) + BlockHeightSize
	if data != nil {
		bsSize += dataSize
	}

	bh := common.Uint64ToBytes(blockHeight)
	bs := make([]byte, bsSize)

	copy(bs, prefix)
	copy(bs[len(prefix) + BlockHeightSize - len(bh):], bh)
	if data != nil {
		copy(bs[bsSize-dataSize:], data)
	}

	return util.BytesPrefix(bs)
}

// Delete PreCommit data with blockHeight and blockHash
// To delete only with blockHeight, pass blockHash as nil
func flushPreCommit(pcDB db.Database, blockHeight uint64, blockHash []byte) error {
	prefix := makeIteratorPrefix(db.PrefixClaim, blockHeight, blockHash, BlockHashSize)

	return deletePreCommit(pcDB, prefix.Start, prefix.Limit)
}

// Delete PreCommit data with block height greater than blockHeight
func initPreCommit(pcDB db.Database, blockHeight uint64) error {
	prefix := makeIteratorPrefix(db.PrefixClaim, blockHeight + 1, nil, BlockHashSize)

	return deletePreCommit(pcDB, prefix.Start, nil)
}

// Delete PreCommit data with iterator
func deletePreCommit(pcDB db.Database, start []byte, limit []byte) error {
	bucket, err := pcDB.GetBucket(db.PrefixClaim)
	if err != nil {
		return err
	}

	iter, err := pcDB.GetIterator()
	if err != nil {
		return err
	}

	// iterate & get keys to delete
	keys := make([][]byte, 0)
	iter.New(start, limit)
	for iter.Next() {
		key := make([]byte, PreCommitIDSize)
		copy(key, iter.Key()[len(db.PrefixClaim):])
		keys = append(keys, key)
	}
	iter.Release()
	err = iter.Error()
	if err != nil {
		log.Printf("There is error while flush preCommit. %v", err)
		return err
	}

	for _, key := range keys {
		err = bucket.Delete(key)
		if err != nil {
			log.Printf("Failed to delete precommit data. %x", key)
		}
	}

	return nil
}

func writePreCommitToClaimDB(preCommitDB db.Database, claimDB db.Database, claimBackupDB db.Database,
	blockHeight uint64, blockHash []byte) error {
	iter, err := preCommitDB.GetIterator()
	if err != nil {
		return err
	}

	// iterate & get values to write
	var pc PreCommit
	var claim Claim
	bucket, _ := claimDB.GetBucket(db.PrefixIScore)
	cbBucket, _ := claimBackupDB.GetBucket(db.PrefixIScore)

	prefix := makeIteratorPrefix(db.PrefixClaim, blockHeight, blockHash, BlockHashSize)
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		err = pc.SetBytes(iter.Value())
		if err != nil {
			break
		}

		claim = pc.Claim
		if pc.Confirmed == false || claim.Data.IScore.Sign() == 0 {
			log.Printf("Do not write precommit data to claim DB. (precommit: %s)", pc.String())
			continue
		}
		bs, _ := bucket.Get(claim.ID())
		if nil != bs {
			oldClaim, _ := NewClaimFromBytes(bs)
			if claim.Data.BlockHeight <= oldClaim.Data.BlockHeight {
				log.Printf("Do not write precommit data to claim DB. too low block height(%d <= %d)",
					claim.Data.BlockHeight, oldClaim.Data.BlockHeight)
				continue
			}
			// update with old I-Score
			claim.Data.IScore.Add(&claim.Data.IScore.Int, &oldClaim.Data.IScore.Int)
		}

		// read value original from claim DB
		if bs, _ := bucket.Get(claim.ID()); bs != nil {
			// write original value to claim backup DB
			cbBucket.Set(claim.BackupID(blockHeight - 1), bs)
		} else {
			// write empty value to claim backup DB
			var nilClaim Claim
			cbBucket.Set(claim.BackupID(blockHeight - 1), nilClaim.Bytes())
		}

		// write to claim DB
		bucket.Set(claim.ID(), claim.Bytes())
	}
	iter.Release()
	if err != nil {
		log.Printf("There is error while write preCommit to claim. %v", err)
		return err
	}

	err = iter.Error()
	if err != nil {
		return err
	}

	err = writeClaimBackupInfo(claimBackupDB, blockHeight)
	if err != nil {
		return err
	}

	// flush precommit with block height
	return flushPreCommit(preCommitDB, blockHeight, nil)
}

func writeClaimBackupInfo(claimBackupDB db.Database, blockHeight uint64) error {
	var cbInfo ClaimBackupInfo
	cbBucket, _ := claimBackupDB.GetBucket(db.PrefixManagement)
	bs, err := cbBucket.Get(cbInfo.ID())
	if err != nil {
		return err
	}
	if bs != nil {
		err = cbInfo.SetBytes(bs)
		if err != nil {
			return err
		}
	}
	if cbInfo.FirstBlockHeight == 0 {
		cbInfo.FirstBlockHeight = blockHeight
	}
	if cbInfo.LastBlockHeight == 0 {
		cbInfo.LastBlockHeight = blockHeight
	}

	// do garbage collection of claim backup DB
	if blockHeight > ClaimBackupPeriod + cbInfo.FirstBlockHeight {
		garbageBlock := blockHeight - ClaimBackupPeriod - 1

		err = garbageCollectClaimBackupDB(claimBackupDB, cbInfo.FirstBlockHeight, garbageBlock)
		if err != nil {
			return err
		}
		// set first block height
		if cbInfo.FirstBlockHeight < garbageBlock + 1 {
			cbInfo.FirstBlockHeight = garbageBlock + 1
		}
	}

	// write management Info. to claim backup DB
	if cbInfo.LastBlockHeight < blockHeight {
		cbInfo.LastBlockHeight = blockHeight
	}
	return cbBucket.Set(cbInfo.ID(), cbInfo.Bytes())
}

func garbageCollectClaimBackupDB(cbDB db.Database, from uint64, to uint64) error {
	bucket, err := cbDB.GetBucket(db.PrefixClaim)
	if err != nil {
		return err
	}

	iter, err := cbDB.GetIterator()
	if err != nil {
		return err
	}

	keys := make([][]byte, 0)
	for blockHeight := from; blockHeight <= to; blockHeight++ {
		prefix := makeIteratorPrefix(db.PrefixClaim, blockHeight, nil, 0)
		iter.New(prefix.Start, prefix.Limit)
		for iter.Next() {
			key := make([]byte, ClaimBackupIDSize)
			copy(key, iter.Key()[len(db.PrefixClaim):])
			keys = append(keys, key)
		}
		iter.Release()

		err = iter.Error()
		if err != nil {
			return err
		}
	}

	for _, key := range keys {
		err = bucket.Delete(key)
		if err != nil {
			log.Printf("Failed to delete claim backup data. %x", key)
		}
	}

	return nil
}

func checkClaimDBRollback(cbInfo *ClaimBackupInfo, rollback uint64) (bool, error) {
	if cbInfo.FirstBlockHeight > rollback {
		return false, &RollbackLowBlockHeightError{ cbInfo.FirstBlockHeight, rollback}
	}

	if cbInfo.LastBlockHeight <= rollback {
		log.Printf("No need to rollback claim DB to %d. backup %d", rollback, cbInfo.LastBlockHeight)
		return false, nil
	}

	return true, nil
}

func rollbackClaimDB(ctx *Context, to uint64, blockHash []byte) error {
	log.Printf("Start Rollback claim DB to %d", to)
	idb := ctx.DB
	cDB := idb.getClaimDB()
	cbDB := idb.getClaimBackupDB()
	bucket, err := cbDB.GetBucket(db.PrefixManagement)
	if err != nil {
		return err
	}

	var cbInfo ClaimBackupInfo
	bs, _ := bucket.Get(cbInfo.ID())
	cbInfo.SetBytes(bs)

	// check Rollback block height
	if ok, err := checkClaimDBRollback(&cbInfo, to); ok != true {
		return err
	}

	from := cbInfo.LastBlockHeight

	cBucket, err := cDB.GetBucket(db.PrefixClaim)
	if err != nil {
		return err
	}

	for i := from; to < i; i-- {
		err = _rollbackClaimDB(cbDB, cBucket, i)
		if err != nil {
			return err
		}
	}

	// update management Info.
	cbInfo.LastBlockHeight = to
	bucket.Set(cbInfo.ID(), cbInfo.Bytes())

	idb.rollbackCurrentBlockInfo(to, blockHash)

	log.Printf("End Rollback claim DB from %d to %d", from, to)
	return nil
}

func _rollbackClaimDB(cbDB db.Database, cBucket db.Bucket, blockHeight uint64) error {
	iter, err := cbDB.GetIterator()
	if err != nil {
		return err
	}

	prefix := makeIteratorPrefix(db.PrefixClaim, blockHeight, nil, 0)
	log.Printf("blockHeight: %d, iterate %v - %v", blockHeight, prefix.Start, prefix.Limit)
	iter.New(prefix.Start, prefix.Limit)
	var claim Claim
	keys := make([][]byte, 0)
	for iter.Next() {
		claim.SetBytes(iter.Value())
		key := iter.Key()[len(db.PrefixClaim)+BlockHeightSize:]
		log.Printf("Rollback with %s %s", ClaimBackupKeyString(iter.Key()), claim.String())
		if claim.Data.BlockHeight == 0 && claim.Data.IScore.Sign() == 0 {
			cBucket.Delete(key)
		} else {
			cBucket.Set(key, iter.Value())
		}
		// gather IDs for deletion
		key = make([]byte, ClaimBackupIDSize)
		copy(key, iter.Key()[len(db.PrefixClaim):])
		keys = append(keys, key)
	}
	iter.Release()

	if err := iter.Error(); err != nil {
		return err
	}

	// delete Rollback data from claim backup DB
	cbBucket, err := cbDB.GetBucket(db.PrefixClaim)
	if err != nil {
		log.Printf("Failed to delete claim backup data. %+v", err)
		return err
	}
	for _, v := range keys {
		cbBucket.Delete(v)
	}

	return nil
}