package core

import (
	"encoding/json"
	"log"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
)

type ClaimData struct {
	BlockHeight   uint64
	IScore        common.HexInt
}

type Claim struct {
	Address  common.Address
	PrevData ClaimData
	Confirmed bool
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
