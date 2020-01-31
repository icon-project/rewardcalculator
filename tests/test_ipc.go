package tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/core"
)

const (
	ipcTypeVersion = "version"
	ipcTypeClaim = "claim"
	ipcTypeQuery = "query"
	ipcTypeCalculate = "calculate"
	ipcTypeCommitBlock = "commit_block"
	ipcTypeCommitClaim = "commit_claim"
	ipcTypeQueryCalcStatus = "query_calc_status"
	ipcTypeQueryCalcResult = "query_calc_result"
	ipcTypeRollback = "rollback"
	ipcTypeINIT = "init"
)

type TestIPC interface {
	json() string
	run(opt *testOption) (interface{}, error)
	convertExpectToResponse() interface{}
}

type ipc struct {
	Name     string          `json:"name"`
	DataType string          `json:"type"`
	Data     json.RawMessage `json:"data"`
}

func (id *ipc) UnMarshal() (TestIPC, error) {
	var dst TestIPC
	switch id.DataType {
	case ipcTypeVersion:
		dst = new(ipcVersion)
	case ipcTypeClaim:
		dst = new(ipcClaim)
	case ipcTypeQuery:
		dst = new(ipcQuery)
	case ipcTypeCalculate:
		dst = new(ipcCalculate)
	case ipcTypeCommitBlock:
		dst = new(ipcCommitBlock)
	case ipcTypeCommitClaim:
		dst = new(ipcCommitClaim)
	case ipcTypeQueryCalcStatus:
		dst = new(ipcQueryCalcStatus)
	case ipcTypeQueryCalcResult:
		dst = new(ipcQueryCalcResult)
	case ipcTypeRollback:
		dst = new(ipcRollback)
	case ipcTypeINIT:
		dst = new(ipcINIT)
	default:
		return nil, fmt.Errorf("unknown datatype : %s", id.DataType)
	}

	// unmarshal data
	err := json.Unmarshal(id.Data, dst)
	if err != nil {
		return nil, err
	}

	return dst, nil
}

func (id *ipc) json() string {
	data, err := id.UnMarshal()
	if err != nil {
		return fmt.Sprintf("Failed to unmarshal IPC data. %v", err)
	}
	return fmt.Sprintf("{\"name\":%s,\"type\":\"%s\",\"data\":%s}", id.Name, id.DataType, data.json())
}

func (id *ipc) run(opt *testOption) error {
	data, err := id.UnMarshal()
	if err != nil {
		return fmt.Errorf("failed to unmarshal IPC data. %v", err)
	}

	resp, err := data.run(opt)
	if err != nil {
		err = fmt.Errorf("'%s' failed. %v", id.Name, err)
		return err
	}

	if resp != nil {
		exp := data.convertExpectToResponse()
		if exp != nil {
			return equalsUp(exp, resp, id.Name)
		}
	}

	return nil
}

type versionExpect struct {
	Version     uint64 `json:"version"`
	BlockHeight uint64 `json:"block_height"`
	BlockHash   string `json:"block_hash,omitempty"`
}

type ipcVersion struct {
	Request struct{
	}	`json:"request"`
	Expect *versionExpect	`json:"expect,omitempty"`
}

func (v *ipcVersion) json() string {
	b, err := json.Marshal(v)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (v *ipcVersion) run(opt *testOption) (interface{}, error) {
	resp, err := opt.ipc.SendVersion()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (v *ipcVersion) convertExpectToResponse() interface{} {
	if v.Expect == nil { return nil }
	resp := new(core.ResponseVersion)
	resp.Version = v.Expect.Version
	resp.BlockHeight = v.Expect.BlockHeight
	blockHash, _ := hex.DecodeString(v.Expect.BlockHash)
	copy(resp.BlockHash[:], blockHash)

	return resp
}

type claimExpect struct {
	Address string		`json:"address"`
	BlockHeight uint64	`json:"block_height"`
	BlockHash string	`json:"block_hash"`
	TXIndex uint64		`json:"tx_index"`
	TXHash string		`json:"tx_hash"`
	IScore uint64		`json:"iscore"`
}

type ipcClaim struct {
	Request struct {
		Address string		`json:"address"`
		BlockHeight uint64	`json:"block_height"`
		BlockHash string	`json:"block_hash"`
		TXIndex uint64		`json:"tx_index"`
		TXHash string		`json:"tx_hash"`
	}	`json:"request"`
	Expect *claimExpect		`json:"expect,omitempty"`
}

func (c *ipcClaim) json() string {
	b, err := json.Marshal(c)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (c *ipcClaim) run(opt *testOption) (interface{}, error) {
	req := c.Request
	resp, err := opt.ipc.SendClaim(req.Address, req.BlockHeight, req.BlockHash, req.TXIndex, req.TXHash, false, false)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *ipcClaim) convertExpectToResponse() interface{} {
	if c.Expect == nil { return nil }
	exp := c.Expect
	resp := new(core.ResponseClaim)
	resp.Address = *common.NewAddressFromString(exp.Address)
	resp.BlockHeight = exp.BlockHeight
	hash, _ := hex.DecodeString(exp.BlockHash)
	resp.BlockHash = make([]byte, core.BlockHashSize)
	copy(resp.BlockHash[:], hash)
	resp.TXIndex = exp.TXIndex
	hash, _ = hex.DecodeString(exp.TXHash)
	resp.TXHash = make([]byte, core.TXHashSize)
	copy(resp.TXHash[:], hash)
	resp.IScore = *common.NewHexIntFromUint64(exp.IScore)

	return resp
}

type queryExpect struct {
	Address string		`json:"address"`
	IScore uint64		`json:"iscore"`
	BlockHeight uint64	`json:"block_height"`
}

type ipcQuery struct {
	Request struct{
		Address string		`json:"address"`
	}	`json:"request"`
	Expect *queryExpect		`json:"expect,omitempty"`
}

func (q *ipcQuery) json() string {
	b, err := json.Marshal(q)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (q *ipcQuery) run(opt *testOption) (interface{}, error) {
	resp, err := opt.ipc.SendQuery(q.Request.Address)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (q *ipcQuery) convertExpectToResponse() interface{} {
	if q.Expect == nil { return nil }
	exp := q.Expect
	resp := new(core.ResponseQuery)
	resp.Address = *common.NewAddressFromString(exp.Address)
	resp.BlockHeight = exp.BlockHeight
	resp.IScore =  *common.NewHexIntFromUint64(exp.IScore)
	return resp
}

type calculateExpect struct {
	Status uint16		`json:"status"`
	BlockHeight uint64	`json:"block_height"`
}

type ipcCalculate struct {
	Request struct{
		BlockHeight uint64	`json:"block_height"`
	}	`json:"request"`
	Expect *calculateExpect	`json:"expect,omitempty"`
}

func (c *ipcCalculate) json() string {
	b, err := json.Marshal(c)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (c *ipcCalculate) run(opt *testOption) (interface{}, error) {
	req := c.Request
	path := filepath.Join(opt.rootPath, fmt.Sprintf(IISSDBPathFormat, req.BlockHeight))
	resp, err := opt.ipc.SendCalculate(path, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *ipcCalculate) convertExpectToResponse() interface{} {
	if c.Expect == nil { return nil }
	exp := c.Expect
	resp := new(core.CalculateResponse)
	resp.Status = exp.Status
	resp.BlockHeight = exp.BlockHeight
	return resp
}

type commitBlockExpect struct {
	Success bool		`json:"success"`
	BlockHeight uint64	`json:"block_height"`
	BlockHash string	`json:"block_hash"`
}

type ipcCommitBlock struct {
	Request struct{
		Success bool		`json:"success"`
		BlockHeight uint64	`json:"block_height"`
		BlockHash string	`json:"block_hash"`
	}	`json:"request"`
	Expect *commitBlockExpect	`json:"expect,omitempty"`
}

func (c *ipcCommitBlock) json() string {
	b, err := json.Marshal(c)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (c *ipcCommitBlock) run(opt *testOption) (interface{}, error) {
	req := c.Request
	resp, err := opt.ipc.SendCommitBlock(req.Success, req.BlockHeight, req.BlockHash)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *ipcCommitBlock) convertExpectToResponse() interface{} {
	if c.Expect == nil { return nil }
	exp := c.Expect
	resp := new(core.CommitBlock)
	resp.Success = exp.Success
	resp.BlockHeight = exp.BlockHeight
	hash, _ := hex.DecodeString(exp.BlockHash)
	resp.BlockHash = make([]byte, core.BlockHashSize)
	copy(resp.BlockHash[:], hash)
	return resp
}

type ipcCommitClaim struct {
	Request struct{
		Success bool		`json:"success"`
		Address string		`json:"address"`
		BlockHeight uint64	`json:"block_height"`
		BlockHash string	`json:"block_hash"`
		TXIndex uint64		`json:"tx_index"`
		TXHash string		`json:"tx_hash"`
	}	`json:"request"`
	Expect struct {
	}	`json:"expect,omitempty"`
}

func (c *ipcCommitClaim) json() string {
	b, err := json.Marshal(c)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (c *ipcCommitClaim) run(opt *testOption) (interface{}, error) {
	req := c.Request
	err := opt.ipc.SendCommitClaim(req.Success, req.Address, req.BlockHeight, req.BlockHash, req.TXIndex, req.TXHash)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (c *ipcCommitClaim) convertExpectToResponse() interface{} {
	return nil
}

type queryCalcStatusExpect struct {
	Status uint64		`json:"status"`
	BlockHeight uint64	`json:"block_height"`
}

type ipcQueryCalcStatus struct {
	Request struct{
	}	`json:"request,omitempty"`
	Expect *queryCalcStatusExpect	`json:"expect,omitempty"`
}

func (q *ipcQueryCalcStatus) json() string {
	b, err := json.Marshal(q)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (q *ipcQueryCalcStatus) run(opt *testOption) (interface{}, error) {
	resp, err := opt.ipc.SendQueryCalculateStatus()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (q *ipcQueryCalcStatus) convertExpectToResponse() interface{} {
	if q.Expect == nil { return nil }
	exp := q.Expect
	resp := new(core.QueryCalculateStatusResponse)
	resp.Status = exp.Status
	resp.BlockHeight = exp.BlockHeight
	return resp
}

type queryCalcResultExpect struct {
	Status uint16		`json:"status"`
	BlockHeight uint64	`json:"block_height"`
	IScore uint64		`json:"iscore"`
	StateHash string	`json:"state_hash"`
}

type ipcQueryCalcResult struct {
	Request struct{
		BlockHeight uint64	`json:"block_height"`
	}	`json:"request"`
	Expect *queryCalcResultExpect `json:"expect,omitempty"`
}

func (q *ipcQueryCalcResult) json() string {
	b, err := json.Marshal(q)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (q *ipcQueryCalcResult) run(opt *testOption) (interface{}, error) {
	req := q.Request
	resp, err := opt.ipc.SendQueryCalculateResult(req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (q *ipcQueryCalcResult) convertExpectToResponse() interface{} {
	if q.Expect == nil { return nil }
	exp := q.Expect
	resp := new(core.QueryCalculateResultResponse)
	resp.Status = exp.Status
	resp.BlockHeight = exp.BlockHeight
	resp.IScore = *common.NewHexIntFromUint64(exp.IScore)
	hash, _ := hex.DecodeString(exp.StateHash)
	copy(resp.StateHash[:], hash)
	return resp
}

type rollbackExpect struct {
	Success bool		`json:"success"`
	BlockHeight uint64	`json:"block_height"`
	BlockHash string	`json:"block_hash"`
}

type ipcRollback struct {
	Request struct{
		BlockHeight uint64	`json:"block_height"`
		BlockHash string	`json:"block_hash"`
	}	`json:"request"`
	Expect *rollbackExpect	`json:"expect,omitempty"`
}

func (r *ipcRollback) json() string {
	b, err := json.Marshal(r)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (r *ipcRollback) run(opt *testOption) (interface{}, error) {
	req := r.Request
	resp, err := opt.ipc.SendRollback(req.BlockHeight, req.BlockHash)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (r *ipcRollback) convertExpectToResponse() interface{} {
	if r.Expect == nil { return nil }
	exp := r.Expect
	resp := new(core.RollBackResponse)
	resp.Success = exp.Success
	resp.BlockHeight = exp.BlockHeight
	hash, _ := hex.DecodeString(exp.BlockHash)
	resp.BlockHash = make([]byte, core.BlockHashSize)
	copy(resp.BlockHash[:], hash)
	return resp
}

type initExpecct struct {
	Success bool		`json:"success"`
	BlockHeight uint64	`json:"block_height"`
}

type ipcINIT struct {
	Request struct{
		BlockHeight uint64	`json:"block_height"`
	}	`json:"request"`
	Expect *initExpecct		`json:"expect,omitempty"`
}

func (i *ipcINIT) json() string {
	b, err := json.Marshal(i)
	if err != nil {
		return "Can't convert Message to json"
	}
	return string(b)
}

func (i *ipcINIT) run(opt *testOption) (interface{}, error) {
	req := i.Request
	resp, err := opt.ipc.SendInit(req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (i *ipcINIT) convertExpectToResponse() interface{} {
	if i.Expect == nil { return nil }
	exp := i.Expect
	resp := new(core.ResponseInit)
	resp.Success = exp.Success
	resp.BlockHeight = exp.BlockHeight
	return resp
}

func equalsUp(exp, act interface{}, name string) error {
	if !reflect.DeepEqual(exp, act) {
		return fmt.Errorf("%s: \n\texppectd: %v (%T)\n\t     got: %v (%T)",
			name, exp, exp, act, act)
	}
	return nil
}
