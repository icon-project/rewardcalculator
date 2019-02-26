package rewardcalculator

import (
	"encoding/json"
	"strconv"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
)

const (
	PrefixIScore             = ""
	PrefixGovernanceVariable = "G"
	PrefixPrepCandidate      = "P"
	PrefixLastCalculation    = "L"
)

const (
	NumDelegate              = 10
	NumPRep                  = 22
)

type IconDB interface {
	ID() []byte
	Bytes() []byte
	String() string
	SetBytes([]byte) error
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

type prepList struct {
	Address   common.Address
	start     common.HexUint64
	end       common.HexUint64
}

type iscoreDB struct {
	db db.Database
}

func writeGovernanceVariable(lvlDB db.Database, gv []byte, blockHeight uint) {
	bucket, _ := lvlDB.GetBucket(PrefixGovernanceVariable)

	bucket.Set([]byte(strconv.FormatUint(uint64(blockHeight), 10)), gv)
}

func writeIscoreDBBytes(data []byte) {

}

func InitDB(dbPath string, dbType string, dbName string) db.Database {
	return db.Open(dbPath, dbType, dbName)
}
