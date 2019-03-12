package rewardcalculator

import (
	"encoding/json"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
)

type DelegateData struct {
	Address     common.Address
	Delegate    common.HexInt
}

type IScoreData struct {
	IScore      common.HexInt
	BlockHeight common.HexUint64
	Stake       common.HexInt
	Delegations [NumDelegate]DelegateData
}

type IScoreAccount struct {
	IScoreData
	Address common.Address
}

func (ia *IScoreAccount) ID() []byte {
	return ia.Address.ID()
}

func (ia *IScoreAccount) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&ia.IScoreData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (ia *IScoreAccount) String() string {
	b, err := json.Marshal(ia)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (ia *IScoreAccount) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &ia.IScoreData)
	if err != nil {
		return err
	}
	return nil
}

func NewIScoreAccountFromBytes(bs []byte) (*IScoreAccount, error) {
	ia := new(IScoreAccount)
	if err:= ia.SetBytes(bs); err != nil {
		return nil, err
	} else {
		return ia, nil
	}
}
