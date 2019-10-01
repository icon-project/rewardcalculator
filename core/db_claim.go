package core

import (
	"encoding/json"
	"fmt"
	"log"
	"unsafe"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type ClaimData struct {
	BlockHeight   uint64
	IScore        common.HexInt
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
	b, err := json.Marshal(c)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
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

type PreCommitData struct {
	Confirmed bool
	Claim
}

const blockHeightSize = 8
const blockHashSize = 32
const PreCommitIDSize = blockHeightSize + blockHashSize + common.AddressBytes

type PreCommit struct {
	BlockHeight uint64
	BlockHash []byte
	PreCommitData
}

func (pc *PreCommit) ID() []byte {
	id := make([]byte, PreCommitIDSize)

	bh := common.Uint64ToBytes(pc.BlockHeight)
	copy(id[blockHeightSize - len(bh):], bh)
	copy(id[blockHeightSize:], pc.BlockHash)
	copy(id[blockHeightSize + blockHashSize:], pc.Address.Bytes())

	return id
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
	b, err := json.Marshal(pc)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (pc *PreCommit) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &pc.PreCommitData)
	if err != nil {
		return err
	}
	return nil
}

func newPreCommit(blockHeight uint64, blockHash []byte, address common.Address) *PreCommit {
	pc := new(PreCommit)

	pc.BlockHeight = blockHeight
	pc.BlockHash = make([]byte, blockHashSize)
	copy(pc.BlockHash, blockHash)
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
	if pc.query(pcDB) == false {
		return fmt.Errorf("no data to commit")
	}
	if pc.Confirmed == true {
		return nil
	}
	pc.Confirmed = true
	return pc.write(pcDB, nil)
}

func (pc *PreCommit) revert(pcDB db.Database) error {
	if pc.query(pcDB) == false {
		return fmt.Errorf("no data to commit")
	}
	if pc.Confirmed == true {
		return nil
	} else {
		return pc.delete(pcDB)
	}
}

func makeIteratorPrefix(blockHeight uint64, blockHash []byte) *util.Range {
	blockHeightSize := int(unsafe.Sizeof(blockHeight))
	bsSize := len(db.PrefixClaim) + blockHeightSize
	if blockHash != nil {
		bsSize += blockHashSize
	}

	bh := common.Uint64ToBytes(blockHeight)
	bs := make([]byte, bsSize)

	copy(bs, db.PrefixClaim)
	copy(bs[len(db.PrefixClaim) + blockHeightSize - len(bh):], bh)
	if blockHash != nil {
		copy(bs[bsSize-blockHashSize:], blockHash)
	}

	return util.BytesPrefix(bs)
}

func flushPreCommit(pcDB db.Database, blockHeight uint64, blockHash []byte) error {
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
	prefix := makeIteratorPrefix(blockHeight, blockHash)
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		key := make([]byte, PreCommitIDSize)
		copy(key, iter.Key()[len(db.PrefixClaim):])
		keys = append(keys, key)
	}
	iter.Release()

	err = iter.Error()
	if err != nil {
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

func writePreCommitToClaimDB(pcDB db.Database, cDB db.Database, blockHeight uint64, blockHash []byte) error {
	iter, err := pcDB.GetIterator()
	if err != nil {
		return err
	}

	// iterate & get values to write
	var pc PreCommit
	var claim Claim
	bucket, _ := cDB.GetBucket(db.PrefixIScore)

	prefix := makeIteratorPrefix(blockHeight, blockHash)
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		err = pc.SetBytes(iter.Value())
		if err != nil {
			break
		}

		claim = pc.Claim
		if pc.Confirmed == false || claim.Data.IScore.Sign() == 0 {
			continue
		}
		bs, _ := bucket.Get(claim.ID())
		if nil != bs {
			oldClaim, _ := NewClaimFromBytes(bs)
			if claim.Data.BlockHeight <= oldClaim.Data.BlockHeight {
				continue
			}
			// update with old I-Score
			claim.Data.IScore.Add(&claim.Data.IScore.Int, &oldClaim.Data.IScore.Int)
		}

		// write to claim DB
		bucket.Set(claim.ID(), claim.Bytes())
	}
	iter.Release()
	if err != nil {
		return err
	}

	err = iter.Error()
	if err != nil {
		return err
	}

	// flush precommit with block height
	return flushPreCommit(pcDB, blockHeight, nil)
}
