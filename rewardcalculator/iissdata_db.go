package rewardcalculator

import (
	"encoding/json"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
)

const (
	IISSKeyHeader      = "HD"
	IISSKeyGV          = "gv"
)

type IISSHeader struct {
	Version     uint16
	BlockHeight uint64
}

func (db *IISSHeader) ID() []byte {
	return []byte(IISSKeyHeader)
}

func (db *IISSHeader) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(db); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (db *IISSHeader) String() string {
	b, err := json.Marshal(db)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (db *IISSHeader) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, db)
	if err != nil {
		return err
	}
	return nil
}

type IISSGovernanceVariable struct {
	IcxPrice      uint64
	IncentiveRep  uint64
}

func (db *IISSGovernanceVariable) ID() []byte {
	return []byte(IISSKeyGV)
}

func (db *IISSGovernanceVariable) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(db); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (db *IISSGovernanceVariable) String() string {
	b, err := json.Marshal(db)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (db *IISSGovernanceVariable) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, db)
	if err != nil {
		return err
	}
	return nil
}

type IISSPRepData struct {
	BlockGenerateCount uint32
	BlockValidateCount uint32
}

type IISSPRep struct {
	IISSPRepData
	Address *common.Address
}

func (db *IISSPRep) ID() []byte {
	return db.Address.ID()
}

func (db *IISSPRep) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&db.IISSPRepData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (db *IISSPRep) String() string {
	b, err := json.Marshal(db)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (db *IISSPRep) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &db.IISSPRepData)
	if err != nil {
		return err
	}
	return nil
}

func NewIISSPRepFromBytes(bs []byte) (*IISSPRep, error) {
	data := new(IISSPRep)
	if err:= data.SetBytes(bs); err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

const (
	TXDataTypeStake     = 0
	TXDataTypeDelegate  = 1
	TXDataTypeClaim     = 2
	TXDataTypePrepReg   = 3
	TXDataTypePrepUnReg = 4
)

type IISSTXData struct {
	Address     common.Address
	BlockHeight uint64
	DataType    uint16
	Data        *codec.TypedObj
}

type IISSTX struct {
	IISSTXData
	TXHash []byte
}

func (db *IISSTX) ID() []byte {
	return db.TXHash
}

func (db *IISSTX) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&db.IISSTXData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (db *IISSTX) String() string {
	b, err := json.Marshal(db)
	if err != nil {
		return "Can't covert Message to json"
	}

	return fmt.Sprintf("%s, Data: %+v", string(b), common.MustDecodeAny(db.Data))
}

func (db *IISSTX) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &db.IISSTXData)
	if err != nil {
		return err
	}
	return nil
}

type IISSTXDataStake struct {
	stake common.HexInt
}

type IISSDelegationData struct {
	Address common.Address
	ratio   uint64
}

type IISSTXDataDelegation struct {
	Delegation [NumDelegate]IISSDelegationData
}

func (db *IISSTXDataDelegation) FromTypedObj(to *codec.TypedObj) {
	data, _ := common.DecodeAny(to)
	dgList, _ := data.([]interface{})

	for i := 0; i < len(dgList); i++ {
		dg, _ := dgList[i].([]interface{})
		addr, _ := dg[0].(*common.Address)
		ratio, _ := dg[1].(*common.HexInt)
		db.Delegation[i].Address = *addr
		db.Delegation[i].ratio = ratio.Uint64()
	}
}

func (db *IISSTXDataDelegation) String() string {
	b, err := json.Marshal(db)
	if err != nil {
		return "Can't covert Message to json"
	}

	return string(b)
}
