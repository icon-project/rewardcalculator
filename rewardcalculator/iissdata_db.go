package rewardcalculator

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type IISSHeader struct {
	Version     uint16
	BlockHeight uint64
}

func (ih *IISSHeader) ID() []byte {
	return []byte(db.PrefixIISSHeader)
}

func (ih *IISSHeader) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(ih); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (ih *IISSHeader) String() string {
	b, err := json.Marshal(ih)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (ih *IISSHeader) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, ih)
	if err != nil {
		return err
	}
	return nil
}

type IISSGVData struct {
	IcxPrice      uint64
	IncentiveRep  uint64
}

type IISSGovernanceVariable struct {
	BlockHeight   uint64
	IISSGVData
}

func (gv *IISSGovernanceVariable) ID() []byte {
	bs := make([]byte, 8)
	id := common.Uint64ToBytes(gv.BlockHeight)
	copy(bs[len(bs)-len(id):], id)
	return bs
}

func (gv *IISSGovernanceVariable) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&gv.IISSGVData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (gv *IISSGovernanceVariable) String() string {
	b, err := json.Marshal(gv)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (gv *IISSGovernanceVariable) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &gv.IISSGVData)
	if err != nil {
		return err
	}
	return nil
}

const numPRep	= 22

type IISSPRepStatData struct {
	Generator common.Address
	Validator []common.Address
}

type IISSPRepStat struct {
	BlockHeight uint64
	IISSPRepStatData
}

func (prep  *IISSPRepStat) ID() []byte {
	bs := make([]byte, 8)
	id := common.Uint64ToBytes(prep.BlockHeight)
	copy(bs[len(bs)-len(id):], id)
	return bs
}

func (prep *IISSPRepStat) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&prep.IISSPRepStatData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (prep *IISSPRepStat) String() string {
	b, err := json.Marshal(prep)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (prep *IISSPRepStat) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &prep.IISSPRepStatData)
	if err != nil {
		return err
	}
	return nil
}

const (
	TXDataTypeDelegate  = 0
	TXDataTypeClaim     = 1
	TXDataTypePrepReg   = 2
	TXDataTypePrepUnReg = 3
)

type IISSTXData struct {
	Address     common.Address
	BlockHeight uint64
	DataType    uint64
	Data        *codec.TypedObj
}

type IISSTX struct {
	Index uint64
	IISSTXData
}

func (tx *IISSTX) ID() []byte {
	bs := make([]byte, 8)
	id := common.Uint64ToBytes(tx.Index)
	copy(bs[len(bs)-len(id):], id)
	return bs
}

func (tx *IISSTX) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&tx.IISSTXData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (tx *IISSTX) String() string {
	b, err := json.Marshal(tx)
	if err != nil {
		return "Can't covert Message to json"
	}

	return fmt.Sprintf("%s\n\t Data: %+v", string(b), common.MustDecodeAny(tx.Data))
}

func (tx *IISSTX) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &tx.IISSTXData)
	if err != nil {
		return err
	}
	return nil
}

func LoadIISSData(dbPath string, verbose bool) (*IISSHeader, []*IISSGovernanceVariable, []*IISSPRepStat, []*IISSTX) {
	log.Printf("Start read IISS data DB. name: %s\n", dbPath)

	dbPath = filepath.Clean(dbPath)
	dbDir, dbName := filepath.Split(dbPath)

	iissDB := db.Open(dbDir, string(db.GoLevelDBBackend), dbName)
	defer iissDB.Close()

	// Header
	bucket, _ := iissDB.GetBucket(db.PrefixIISSHeader)
	data, _ := bucket.Get([]byte(""))
	if data == nil {
		log.Printf("There is no header data\n")
		return nil, nil, nil, nil
	}
	header := new(IISSHeader)
	err := header.SetBytes(data)
	if err != nil {
		log.Printf("Failed to read header from IISS Data. err=%+v\n", err)
		return nil, nil, nil, nil
	}
	if verbose {
		log.Printf("Header: %s\n", header.String())
	}

	// Governance Variable
	gvList := make([]*IISSGovernanceVariable, 0)
	iter, _ := iissDB.GetIterator()
	prefix := util.BytesPrefix([]byte(db.PrefixIISSGV))
	iter.New(prefix.Start, prefix.Limit)
	for entries := 0; iter.Next(); entries++ {
		gv := new(IISSGovernanceVariable)
		err = gv.SetBytes(iter.Value())
		if err != nil {
			log.Printf("Failed to read governance variable from IISS Data(%+v). err=%+v\n", iter.Value(), err)
			return nil, nil, nil, nil
		}
		gv.BlockHeight = common.BytesToUint64(iter.Key()[len(db.PrefixIISSGV):])
		gvList = append(gvList, gv)
	}
	iter.Release()

	if verbose {
		if len(gvList) > 0 {
			log.Printf("Governance variable:\n")
			for i, gv := range gvList {
				log.Printf("\t%d: %s", i, gv.String())
			}
		}
	}

	// P-Rep statistics list
	prepStatList := make([]*IISSPRepStat, 0, numPRep)
	iter, _ = iissDB.GetIterator()
	prefix = util.BytesPrefix([]byte(db.PrefixIISSPRep))
	iter.New(prefix.Start, prefix.Limit)
	for entries := 0; iter.Next(); entries++ {
		prepStat := new(IISSPRepStat)
		err = prepStat.SetBytes(iter.Value())
		if err != nil {
			log.Printf("Failed to read P-Rep list from IISS Data. err=%+v\n", err)
			return nil, nil, nil, nil
		}
		prepStat.BlockHeight = common.BytesToUint64(iter.Key()[len(db.PrefixIISSPRep):])
		prepStatList = append(prepStatList, prepStat)
	}
	iter.Release()
	if verbose {
		if len(prepStatList) > 0 {
			log.Printf("P-Rep Stat:\n")
			for i, prepStat := range prepStatList {
				log.Printf("\t%d: %s\n", i, prepStat.String())
			}
		}
	}

	// TX list
	txList := make([]*IISSTX, 0)
	iter, _ = iissDB.GetIterator()
	prefix = util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)
	for entries := 0; iter.Next(); entries++ {
		tx := new(IISSTX)
		err = tx.SetBytes(iter.Value())
		if err != nil {
			log.Printf("Failed to read TX list from IISS Data. err=%+v\n", err)
			return nil, nil, nil, nil
		}
		tx.Index = common.BytesToUint64(iter.Key()[len(db.PrefixIISSTX):])
		txList = append(txList, tx)
	}
	iter.Release()
	if verbose {
		if len(txList) > 0 {
			log.Printf("TX:\n")
			for i, tx := range txList {
				log.Printf("\t%d: %s\n", i, tx.String())
			}
		}
	}

	return header, gvList, prepStatList, txList
}
