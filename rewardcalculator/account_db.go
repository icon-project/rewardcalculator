package rewardcalculator

import (
	"encoding/json"
	"log"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
)

type DelegateData struct {
	Address     common.Address
	Delegate    common.HexInt
}

type IScoreData struct {
	IScore      common.HexInt
	BlockHeight uint64
	Delegations []*DelegateData
}

type IScoreAccount struct {
	Address common.Address
	IScoreData
}

func (ia *IScoreAccount) ID() []byte {
	return ia.Address.Bytes()
}

func (ia *IScoreAccount) Bytes() []byte {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&ia.IScoreData); err != nil {
		log.Panicf("Failed to marshal I-Score account=%+v. err=%+v", ia, err)
		return nil
	} else {
		bytes = bs
	}
	return bytes
}

func (ia *IScoreAccount) BytesForHash() []byte {
	addr := ia.ID()
	iScore := ia.IScore.Bytes()
	blockHeight := common.Uint64ToBytes(ia.BlockHeight)
	buf := make([]byte, len(addr) + len(iScore) + len(blockHeight))
	copy(buf, addr)
	copy(buf[len(addr):], iScore)
	copy(buf[len(addr)+len(iScore):], blockHeight)
	return buf
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

func NewIScoreAccountFromIISS(iisstx *IISSTX) *IScoreAccount {
	if iisstx.DataType != TXDataTypeDelegate {
		return nil
	}

	ia := new(IScoreAccount)
	ia.Address = iisstx.Address
	ia.BlockHeight = iisstx.BlockHeight

	data, _ := common.DecodeAny(iisstx.Data)
	dList1, ok := data.([]interface{})
	if ok {
		ia.Delegations = make([]*DelegateData, len(dList1))
		for i, v := range dList1 {
			dg := new(DelegateData)
			dList2, ok := v.([]interface{})
			if ok {
				if len(dList2) != 2 {
					continue
				}
				for _, v2 := range dList2 {
					switch v2.(type) {
					case *common.Address:
						dg.Address = *v2.(*common.Address)
					case *common.HexInt:
						dg.Delegate = *v2.(*common.HexInt)
					}
				}
				ia.Delegations[i] = dg
			} else {
				return nil
			}
		}
	} else {
		return nil
	}
	return ia
}
