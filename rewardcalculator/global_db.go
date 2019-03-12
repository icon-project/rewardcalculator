package rewardcalculator

import (
	"encoding/json"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
)


type DBInfo struct {
	BlockHeight  common.HexUint64
	DbCount      int
	AccountCount common.HexUint64
}

func (db *DBInfo) ID() []byte {
	return []byte(KeyDBInfo)
}

func (db *DBInfo) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(db); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (db *DBInfo) String() string {
	b, err := json.Marshal(db)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}


func (db *DBInfo) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, db)
	if err != nil {
		return err
	}
	return nil
}

type GovernanceVariable struct {
	IcxPrice      common.HexUint64
	IncentiveRep  common.HexUint64
	IncentiveDapp common.HexUint64
	IncentiveEEP  common.HexUint64
	RewardRep     common.HexUint64
	RewardDapp    common.HexUint64
	RewardEEP     common.HexUint64
}

type GlobalOptions struct {
	BlockHeight   common.HexUint64
	Validators    [NumPRep]common.Address

	GV GovernanceVariable
}

func (opts *GlobalOptions) ID() []byte {
	return nil
}

func (opts *GlobalOptions) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(opts); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (opts *GlobalOptions) String() string {
	b, err := json.Marshal(opts)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (opts *GlobalOptions) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &opts)
	if err != nil {
		return err
	}
	return nil
}
