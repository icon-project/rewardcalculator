package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"io/ioutil"
	"log"
	"os"
)

type CalcDebug struct {
	conf   *CalcDebugConfig
	result *CalcDebugResult
}

type CalcDebugConfig struct {
	Flag      bool              `json:"enable"`
	Addresses []*common.Address `json:"addresses"`
}

func NewCalcDebugConfig() *CalcDebugConfig {
	return &CalcDebugConfig{}
}

type CalcDebugResult struct {
	BlockHeight uint64 `json:"CalculationBlockHeight"`
	BlockHash   string `json:"CalculationBlockHash"`
	CalcDebugData
}

func (cb CalcDebugResult) String() string {
	b, err := json.Marshal(cb)
	if err != nil {
		return "Failed to marshal CalcDebugResult"
	}
	return string(b)
}

func (cb *CalcDebugResult) ID() []byte {
	id := make([]byte, BlockHeightSize+BlockHashSize)
	bh := common.Uint64ToBytes(cb.BlockHeight)
	blockHash, _ := hex.DecodeString(cb.BlockHash[2:])
	copy(id[BlockHeightSize-len(bh):], bh)
	copy(id[BlockHeightSize:], blockHash)

	return id
}

func (cb *CalcDebugResult) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&cb.CalcDebugData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (cb *CalcDebugResult) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &cb.CalcDebugData)
	if err != nil {
		return err
	}
	return nil
}

type CalcDebugData struct {
	Preps   []*PRepCandidate      `json:"PReps"`
	GV      []*GovernanceVariable `json:"GV"`
	Results []*CalcResult         `json:"calculation"`
}

type CalcResult struct {
	Address       *common.Address `json:"address"`
	InitialIScore common.HexInt   `json:"InitialIScore"`
	TotalIScore   common.HexInt   `json:"TotalIScore"`
	Rewards       []*Reward       `json:"rewards"`
}

func NewCalcResult(address *common.Address) *CalcResult {
	return &CalcResult{Address: address}
}

type Reward struct {
	BlockHeight uint64 `json:"blockHeight"`
	Beta1       *Beta1 `json:"beta1"`
	Beta2       *Beta2 `json:"beta2"`
	Beta3       *Beta3 `json:"beta3"`
}

func NewReward(blockHeight uint64) *Reward {
	return &Reward{blockHeight, NewBeta1(), NewBeta2(), NewBeta3()}
}

type Beta1 struct {
	Beta1IScore common.HexInt `json:"Beta1IScore"`
	*Generate   `json:"generate"`
	Validate    []*ValidateInfo `json:"validate"`
}

func NewBeta1() *Beta1 {
	return &Beta1{*common.NewHexInt(0), NewGenerate(), make([]*ValidateInfo, 0)}
}

type Generate struct {
	BlockCount uint64 `json:"blockCount"`
	Formula    string `json:"formula"`
	IScore     uint64 `json:"IScore"`
}

func NewGenerate() *Generate {
	return &Generate{0, "", 0}
}

type ValidateInfo struct {
	ValidatorCount uint64 `json:"validatorCount"`
	BlockCount     uint64 `json:"blockCount"`
	Formula        string `json:"formula"`
	IScore         uint64 `json:"IScore"`
}

type Beta2 struct {
	Beta2IScore   common.HexInt    `json:"Beta2IScore"`
	DelegatedInfo []*DelegatedInfo `json:"delegated"`
}

func NewBeta2() *Beta2 {
	return &Beta2{*common.NewHexInt(0), make([]*DelegatedInfo, 0)}
}

type DelegatedInfo struct {
	BlockHeight    uint64 `json:"blockHeight"`
	Delegated      uint64 `json:"amount"`
	TotalDelegated uint64 `json:"totalDelegated"`
	Formula        string `json:"formula"`
	IScore         uint64 `json:"IScore"`
}

type Beta3 struct {
	Beta3IScore    common.HexInt     `json:"Beta3IScore"`
	DelegationInfo []*DelegationInfo `json:"delegate"`
}

func NewBeta3() *Beta3 {
	return &Beta3{*common.NewHexInt(0), make([]*DelegationInfo, 0)}
}

type DelegationInfo struct {
	BlockHeight uint64         `json:"blockHeight"`
	Address     common.Address `json:"address"`
	Amount      uint64         `json:"amount"`
	Formula     string         `json:"formula"`
	IScore      uint64         `json:"IScore"`
}

func InitCalcDebugConfig(ctx *Context, debugConfigPath string) {
	ctx.calcDebug = new(CalcDebug)
	ctx.calcDebug.conf = NewCalcDebugConfig()
	debugConfig, err := os.Open(debugConfigPath)
	if err != nil {
		log.Printf("Error while opening calculation debug config file: %s. error : %v"+
			"\nResult file will be store in defaultPath : CalculateResult", debugConfigPath, err)
		return
	}
	defer debugConfig.Close()

	cfgByte, _ := ioutil.ReadAll(debugConfig)
	err = json.Unmarshal(cfgByte, ctx.calcDebug.conf)
	if err != nil {
		log.Printf("Error while Unmarshaling config json")
		return
	}
}

func NeedToUpdateCalcDebugResult(ctx *Context) bool {
	return ctx.calcDebug.conf.Flag && len(ctx.calcDebug.conf.Addresses) > 0
}

func InitCalcDebugResult(ctx *Context, blockHeight uint64, blockHash []byte) {
	if !NeedToUpdateCalcDebugResult(ctx) {
		return
	}
	ctx.calcDebug.result = new(CalcDebugResult)
	ctx.calcDebug.result.BlockHeight = blockHeight
	if len(blockHash) == 0 {
		for i := 0; i < BlockHashSize; i++ {
			blockHash = append(blockHash, 0)
		}
	}
	ctx.calcDebug.result.BlockHash = "0x" + hex.EncodeToString(blockHash)

	for _, gv := range ctx.GV {
		ctx.calcDebug.result.GV = append(ctx.calcDebug.result.GV, gv)
	}

	for _, prep := range ctx.PRepCandidates {
		ctx.calcDebug.result.Preps = append(ctx.calcDebug.result.Preps, prep)
	}

	for _, address := range ctx.calcDebug.conf.Addresses {
		InitCalcResult(ctx, *address)
	}
}

func WriteBeta1Info(ctx *Context, produceReward uint64, bp IISSBlockProduceInfo) {
	if !NeedToUpdateCalcDebugResult(ctx) {
		return
	}
	for _, address := range ctx.calcDebug.conf.Addresses {
		writeBeta1Info(ctx, address, produceReward, bp)
	}
}

func writeBeta1Info(ctx *Context, address *common.Address, produceReward uint64, bp IISSBlockProduceInfo) {
	if bp.Generator.Equal(address) {
		calcResult := GetCalcResult(ctx, *address)
		reward := getReward(calcResult, bp.BlockHeight)
		reward.Beta1.Generate.BlockCount += 1
		reward.Beta1.Generate.IScore += produceReward
		reward.Beta1.Generate.Formula = fmt.Sprintf("%d * %s",
			reward.Beta1.BlockCount, common.NewHexIntFromUint64(produceReward).String())
		iScoreInHex := *common.NewHexIntFromUint64(produceReward)
		reward.Beta1.Beta1IScore.Add(&reward.Beta1.Beta1IScore.Int, &iScoreInHex.Int)
		calcResult.TotalIScore.Add(&calcResult.TotalIScore.Int, &iScoreInHex.Int)
		return
	}

	for _, addr := range bp.Validator {
		if addr.Equal(address) {
			calcResult := GetCalcResult(ctx, *address)
			reward := getReward(calcResult, bp.BlockHeight)
			validatorCount := len(bp.Validator)
			validateInfo := func(validate []*ValidateInfo, validatorCount uint64) *ValidateInfo {
				for _, validate := range validate {
					if validate.ValidatorCount == validatorCount {
						return validate
					}
				}
				validateInfo := &ValidateInfo{ValidatorCount: validatorCount}
				reward.Beta1.Validate = append(validate, validateInfo)
				return validateInfo
			}(reward.Beta1.Validate, uint64(validatorCount))
			validateInfo.BlockCount += 1
			iScore := produceReward / uint64(len(bp.Validator))
			validateInfo.IScore += iScore
			validateInfo.Formula = fmt.Sprintf("%d * %s / %d",
				validateInfo.BlockCount, common.NewHexIntFromUint64(produceReward).String(), validatorCount)
			iScoreInHex := *common.NewHexIntFromUint64(iScore)
			reward.Beta1.Beta1IScore.Add(&reward.Beta1.Beta1IScore.Int, &iScoreInHex.Int)
			calcResult.TotalIScore.Add(&calcResult.TotalIScore.Int, &iScoreInHex.Int)
			return
		}
	}
}

func WriteBeta2Info(ctx *Context, delegationInfo PRepDelegationInfo, prep PRep,
	startBlock uint64, endBlock uint64, prepReward uint64) {
	if !NeedToUpdateCalcDebugResult(ctx) {
		return
	}
	for _, address := range ctx.calcDebug.conf.Addresses {
		writeBeta2Info(ctx, address, delegationInfo, prep, startBlock, endBlock, prepReward)
	}
}

func writeBeta2Info(ctx *Context, address *common.Address, delegationInfo PRepDelegationInfo, prep PRep,
	startBlock uint64, endBlock uint64, prepReward uint64) {
	if delegationInfo.Address.Equal(address) {
		calcResult := GetCalcResult(ctx, *address)
		reward := getReward(calcResult, endBlock)
		period := endBlock - startBlock
		totalDelegation := prep.TotalDelegation.Uint64()
		delegatedAmount := delegationInfo.DelegatedAmount.Uint64()
		iScore := prepReward * delegatedAmount * period / totalDelegation
		delegatedInfo := &DelegatedInfo{BlockHeight: endBlock, TotalDelegated: totalDelegation,
			Delegated: delegatedAmount, IScore: iScore}
		delegatedInfo.Formula = fmt.Sprintf("%s * %d * %d / %d", common.NewHexIntFromUint64(prepReward).String(),
			delegationInfo.DelegatedAmount.Uint64(), period, totalDelegation)

		reward.Beta2.DelegatedInfo = append(reward.Beta2.DelegatedInfo, delegatedInfo)
		iScoreInHex := *common.NewHexIntFromUint64(iScore)
		reward.Beta2.Beta2IScore.Add(&reward.Beta2.Beta2IScore.Int, &iScoreInHex.Int)
		calcResult.TotalIScore.Add(&calcResult.TotalIScore.Int, &iScoreInHex.Int)
	}
}

func WriteBeta3Info(ctx *Context, rewardAddress common.Address, rewardRep uint64,
	delegationInfo *DelegateData, period uint64, endBlock uint64) {
	if !NeedToUpdateCalcDebugResult(ctx) {
		return
	}
	for _, address := range ctx.calcDebug.conf.Addresses {
		writeBeta3Info(ctx, address, rewardAddress, rewardRep, delegationInfo, period, endBlock)
	}
}

func writeBeta3Info(ctx *Context, debugAddress *common.Address, rewardAddress common.Address, rewardRep uint64,
	delegationInfo *DelegateData, period uint64, endBlock uint64) {
	if rewardAddress.Equal(debugAddress) {
		calcResult := GetCalcResult(ctx, *debugAddress)
		reward := getReward(calcResult, endBlock)
		iScore := rewardRep * period * delegationInfo.Delegate.Uint64() / rewardDivider
		dgInfo := &DelegationInfo{BlockHeight: endBlock, Address: delegationInfo.Address,
			Amount: delegationInfo.Delegate.Uint64(), IScore: iScore}
		dgInfo.Formula = fmt.Sprintf("%s * %d * %d / %d", common.NewHexIntFromUint64(rewardRep).String(), period,
			dgInfo.Amount, rewardDivider)
		reward.Beta3.DelegationInfo = append(reward.Beta3.DelegationInfo, dgInfo)
		iScoreInHex := *common.NewHexIntFromUint64(iScore)
		reward.Beta3.Beta3IScore.Add(&reward.Beta3.Beta3IScore.Int, &common.NewHexIntFromUint64(iScore).Int)
		calcResult.TotalIScore.Add(&calcResult.TotalIScore.Int, &iScoreInHex.Int)
	}
}

func GetCalcResult(ctx *Context, address common.Address) *CalcResult {
	calcResult := getCalcResult(ctx, address)
	if calcResult != nil {
		return calcResult
	}
	InitCalcResult(ctx, address)
	calcResult = getCalcResult(ctx, address)
	return calcResult
}

func getCalcResult(ctx *Context, address common.Address) *CalcResult {
	for _, calcResult := range ctx.calcDebug.result.Results {
		if calcResult.Address.Equal(&address) {
			return calcResult
		}
	}
	return nil
}

func getReward(calcResult *CalcResult, blockHeight uint64) *Reward {
	var r *Reward
	rewardLength := len(calcResult.Rewards)
	for i := rewardLength - 1; i >= 0; i-- {
		reward := calcResult.Rewards[i]
		if reward.BlockHeight < blockHeight {
			r = reward
		}
	}
	return r
}

func InitCalcResult(ctx *Context, address common.Address) {
	result := NewCalcResult(&address)
	initialIScore := *common.NewHexInt(0)
	qDB := ctx.DB.getQueryDB(address)
	bucket, _ := qDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(address.Bytes())
	if bs != nil {
		ia, _ := NewIScoreAccountFromBytes(bs)
		initialIScore = ia.IScore
	}
	result.InitialIScore = initialIScore
	result.TotalIScore = initialIScore

	rewards := make([]*Reward, 0)
	for _, gv := range ctx.GV {
		reward := NewReward(gv.BlockHeight)
		rewards = append(rewards, reward)
	}
	result.Rewards = rewards
	ctx.calcDebug.result.Results = append(ctx.calcDebug.result.Results, result)
}

func WriteCalcDebugResult(ctx *Context) {
	calcDebugDB := db.Open(ctx.DB.info.DBRoot, string(db.GoLevelDBBackend), "calculation_debug")
	defer calcDebugDB.Close()
	bucket, _ := calcDebugDB.GetBucket("")
	b, err := ctx.calcDebug.result.Bytes()
	if err != nil {
		log.Print("Error while marshaling calculation debug result")
		return
	}
	bucket.Set(ctx.calcDebug.result.ID(), b)
}

func AddDebuggingAddress(ctx *Context, address common.Address) {
	found := false
	for i := len(ctx.calcDebug.conf.Addresses) - 1; i >= 0; i-- {
		if address.Equal(ctx.calcDebug.conf.Addresses[i]) {
			found = true
		}
	}
	if !found {
		ctx.calcDebug.conf.Addresses = append(ctx.calcDebug.conf.Addresses, &address)
	}
}

func DeleteDebuggingAddress(ctx *Context, address common.Address) {
	for i := len(ctx.calcDebug.conf.Addresses) - 1; i >= 0; i-- {
		if address.Equal(ctx.calcDebug.conf.Addresses[i]) {
			ctx.calcDebug.conf.Addresses = append(ctx.calcDebug.conf.Addresses[:i],
				ctx.calcDebug.conf.Addresses[i+1:]...)
		}
	}
	if ctx.calcDebug.result == nil || ctx.calcDebug.result.Results == nil {
		return
	}

	for i, calcResult := range ctx.calcDebug.result.Results {
		if calcResult.Address.Equal(&address) {
			ctx.calcDebug.result.Results =
				append(ctx.calcDebug.result.Results[:i], ctx.calcDebug.result.Results[i+1:]...)
		}
	}
}

func ResetCalcDebugResults(ctx *Context) {
	ctx.calcDebug.result.Results = ctx.calcDebug.result.Results[:0]
}
