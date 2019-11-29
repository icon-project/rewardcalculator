package tests

import (
	"encoding/json"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
)

const (
	IISSDBPathFormat = "iiss_%d"
)

type TestIISS interface {
	json() string
	run(opt *testOption) error
}

type iiss struct {
	BlockHeight uint64     `json:"block_height"`
	Data        []iissData `json:"data,omitempty"`
}

func (is *iiss) json() string {
	result := fmt.Sprintf("{\"block_hegith\":%d,\"data\":[", is.BlockHeight)
	for i, data :=range is.Data {
		if i != 0 {
			result = result + ","
		}
		result = result + data.json()
	}
	result = result + "]}"

	return result
}

func (is *iiss) run(opts *testOption) error {
	opts.db = db.Open(opts.rootPath, string(db.GoLevelDBBackend), fmt.Sprintf(IISSDBPathFormat, is.BlockHeight))
	defer opts.db.Close()
	//fmt.Printf("\tIISS : blockHeight: %d, len:%d\n", is.BlockHeight, len(is.Data))
	for _, data := range is.Data {
		//fmt.Printf("\t\t%d : ", i)
		//fmt.Printf("%s\n", data.json())
		err := data.run(opts)
		if err != nil {
			return fmt.Errorf("failed to write IISS data DB. %s. %v", data.json(), err)
		}
	}

	return nil
}

const (
	iissDataTypeHeader	= "Header"
	iissDataTypeGV		= "GV"
	iissDataTypeBP		= "BP"
	iissDataTypePRep	= "PRep"
	iissDataTypeTX		= "TX"

	TXTypeDelegation    = "delegation"
	TXTypeRegisterPRep  = "registerPRep"
	TXTypeUnregisterPRep  = "unregisterPRep"
)

type iissData struct {
	DataType string          `json:"type"`
	Data     json.RawMessage `json:"data"`
}

func (id *iissData) UnMarshal() (TestIISS, error) {
	var dst TestIISS
	switch id.DataType {
	case iissDataTypeHeader:
		dst = new(iissHeader)
	case iissDataTypeGV:
		dst = new(iissGV)
	case iissDataTypeBP:
		dst = new(iissBlockProduce)
	case iissDataTypePRep:
		dst = new(iissPRep)
	case iissDataTypeTX:
		dst = new(iissTX)
	default:
		return dst, fmt.Errorf("unknown datatype : %s", id.DataType)
	}

	// unmarshal data
	err := json.Unmarshal(id.Data, dst)
	if err != nil {
		return dst, err
	}

	return dst, nil
}

func (id *iissData) json() string {
	data, err := id.UnMarshal()
	if err != nil {
		return fmt.Sprintf("Failed to unmarshal IISS data. %v", err)
	}
	return fmt.Sprintf("{\"type\":\"%s\",\"data\":%s}", id.DataType, data.json())
}

func (id *iissData) run(opt *testOption) error {
	data, err := id.UnMarshal()
	if err != nil {
		return fmt.Errorf("failed to unmarshal IISS data. %v", err)
	}

	return data.run(opt)
}

type iissHeader struct {
	Version     uint64 `json:"version"`
	BlockHeight uint64 `json:"block_height"`
	Revision    uint64 `json:"revision,omitempty"`
}

func (hd *iissHeader) json() string {
	b, err := json.Marshal(hd)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (hd *iissHeader) run(opt *testOption) error  {
	return core.WriteIISSHeader(opt.db, hd.Version, hd.BlockHeight, hd.Revision)
}

type iissGV struct {
	BlockHeight uint64		`json:"block_height"`
	Incentive uint64		`json:"i_rep"`
	Reward uint64			`json:"r_rep"`
	MainPRepCount uint64	`json:"main_p_rep_count"`
	SubPRepCount uint64		`json:"sub_p_rep_count"`
}

func (gv *iissGV) json() string {
	b, err := json.Marshal(gv)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (gv *iissGV) run(opt *testOption) error  {
	return core.WriteIISSGV(opt.db, gv.BlockHeight, gv.Incentive, gv.Reward, gv.MainPRepCount, gv.SubPRepCount)
}

type iissBlockProduce struct {
	BlockHeight uint64		`json:"block_height"`
	Generator string		`json:"generator"`
	Validator []string		`json:"validator"`
}

func (bp *iissBlockProduce) json() string {
	b, err := json.Marshal(bp)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (bp *iissBlockProduce) run(opt *testOption) error  {
	return core.WriteIISSBP(opt.db, bp.BlockHeight, bp.Generator, bp.Validator)
}

type iissPRep struct {
	BlockHeight uint64		`json:"block_height"`
	TotalDelegation uint64	`json:"total_delegation"`
	Preps []delegation		`json:"preps"`
}

func (prep *iissPRep) json() string {
	b, err := json.Marshal(prep)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (prep *iissPRep) run(opt *testOption) error  {
	preps := make([]*core.PRepDelegationInfo, 0)
	for _, d := range prep.Preps {
		preps = append(preps, d.translateToPRep())
	}
	return core.WriteIISSPRep(opt.db, prep.BlockHeight, prep.TotalDelegation, preps)
}

type iissTX struct {
	Index uint64				`json:"index"`
	BlockHeight uint64			`json:"block_height"`
	Address string				`json:"address"`
	DataType string				`json:"type"`
	Delegations []delegation	`json:"delegations,omitempty"`
}

func (tx *iissTX) json() string {
	b, err := json.Marshal(tx)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (tx *iissTX) run(opt *testOption) error  {
	delegations := make([]*core.PRepDelegationInfo, 0)
	for _, d := range tx.Delegations {
		delegations = append(delegations, d.translateToPRep())
	}
	return core.WriteIISSTX(opt.db, tx.Index, tx.Address, tx.BlockHeight, tx.typeToCode(), delegations)
}

func (tx *iissTX) typeToCode() uint64 {
	switch tx.DataType {
	case TXTypeDelegation:
		return core.TXDataTypeDelegate
	case TXTypeRegisterPRep:
		return core.TXDataTypePrepReg
	case TXTypeUnregisterPRep:
		return core.TXDataTypePrepUnReg
	default:
		return core.TXDataTypeDelegate
	}
}

type delegation struct {
	Address string		`json:"address"`
	Delegation uint64	`json:"delegation"`
}

func (d *delegation) translateToPRep() *core.PRepDelegationInfo {
	pRep := new(core.PRepDelegationInfo)
	pRep.Address = *common.NewAddressFromString(d.Address)
	pRep.DelegatedAmount.SetUint64(d.Delegation)
	return pRep
}