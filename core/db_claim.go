package core

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"log"
)

const (
	BlockHeightSize = 8
	BlockHashSize   = 32
	TXHashSize      = 32

	ClaimBackupIDSize = BlockHeightSize + common.AddressBytes
	ClaimBackupPeriod = 43120*2 - 1
)

type ClaimData struct {
	BlockHeight uint64
	IScore      common.HexInt
}

func (cd *ClaimData) String() string {
	return fmt.Sprintf("BlockHeight: %d, IScore: %s", cd.BlockHeight, cd.IScore.String())
}

func (cd *ClaimData) equal(cmpData *ClaimData) bool {
	return cd.IScore.Cmp(&cmpData.IScore.Int) == 0 && cd.BlockHeight == cmpData.BlockHeight
}

type Claim struct {
	Address common.Address
	Data    ClaimData
}

func (c *Claim) ID() []byte {
	return c.Address.Bytes()
}

func (c *Claim) BackupID(blockHeight uint64) []byte {
	id := make([]byte, ClaimBackupIDSize)

	bh := common.Uint64ToBytes(blockHeight)
	copy(id[BlockHeightSize-len(bh):], bh)
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
	if err := claim.SetBytes(bs); err != nil {
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

func ClaimBackupKey(blockHeight uint64, address common.Address) []byte {
	id := make([]byte, ClaimBackupIDSize)

	bh := common.Uint64ToBytes(blockHeight)
	copy(id[BlockHeightSize-len(bh):], bh)
	copy(id[BlockHeightSize:], address.Bytes())

	return id
}

type ClaimBackupInfo struct {
	FirstBlockHeight uint64
	LastBlockHeight  uint64
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
