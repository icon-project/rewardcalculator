package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"io/ioutil"
	"log"
	"os"
)

type CalcDebugConfig struct {
	Flag      bool              `json:"enable"`
	Addresses []*common.Address `json:"addresses"`
	Output    string            `json:"output"`
}

func NewCalcDebugConfig() *CalcDebugConfig {
	return &CalcDebugConfig{Output: "CalculateResult"}
}

type CalcDebugResult struct {
	BlockHeight uint64                `json:"CalculationBlockHeight"`
	BlockHash   string                `json:"CalculationBlockHash"`
	Preps       []*PRepCandidate      `json:"PReps"`
	GV          []*GovernanceVariable `json:"GV"`
	Results     []*CalcResult         `json:"calculation"`
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

	copy(id[BlockHeightSize-len(bh):], bh)
	copy(id[BlockHeightSize:], cb.BlockHash[2:])

	return id
}

func (cb *CalcDebugResult) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&cb.Results); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (cb *CalcDebugResult) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &cb.Results)
	if err != nil {
		return err
	}
	return nil
}

type CalcResult struct {
	Address *common.Address `json:"address"`
	Rewards []*Reward       `json:"rewards"`
}

func (cr CalcResult) String() string {
	b, err := json.Marshal(cr)
	if err != nil {
		return "Failed to marshal CalcResult"
	}
	return string(b)
}

func NewCalcResult(address *common.Address) *CalcResult {
	return &CalcResult{Address: address, Rewards: make([]*Reward, 0)}
}

type Reward struct {
	BlockHeight   uint64        `json:"blockHeight"`
	InitialIScore common.HexInt `json:"InitialIScore"`
	TotalIScore   common.HexInt `json:"TotalIScore"`
	Beta1         *Beta1        `json:"beta1"`
	Beta2         *Beta2        `json:"beta2"`
	Beta3         *Beta3        `json:"beta3"`
}

func (r Reward) String() string {
	b, err := json.Marshal(r)
	if err != nil {
		return "Failed to marshal Reward"
	}
	return string(b)
}

func NewReward(blockHeight uint64) *Reward {
	return &Reward{blockHeight, *common.NewHexInt(0), *common.NewHexInt(0), NewBeta1(), NewBeta2(), NewBeta3()}
}

type Beta1 struct {
	Beta1IScore common.HexInt `json:"Beta1IScore"`
	*Generate   `json:"generate"`
	Validate    []*ValidateInfo `json:"validate"`
}

func (b1 *Beta1) String() string {
	b, err := json.Marshal(b1)
	if err != nil {
		return "Failed to marshal Beta1"
	}
	return string(b)
}

func NewBeta1() *Beta1 {
	return &Beta1{*common.NewHexInt(0), NewGenerate(), make([]*ValidateInfo, 0)}
}

type Generate struct {
	BlockCount uint64 `json:"blockCount"`
	Formula    string `json:"formula"`
	IScore     uint64 `json:"IScore"`
}

func (g Generate) String() string {
	b, err := json.Marshal(g)
	if err != nil {
		return "Failed to marshal Generate"
	}
	return string(b)
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

func (v ValidateInfo) String() string {
	b, err := json.Marshal(v)
	if err != nil {
		return "Failed to marshal ValidateInfo"
	}
	return string(b)
}

type Beta2 struct {
	Beta2IScore   common.HexInt    `json:"Beta2IScore"`
	DelegatedInfo []*DelegatedInfo `json:"delegated"`
}

func (b2 Beta2) String() string {
	b, err := json.Marshal(b2)
	if err != nil {
		return "Failed to marshal Beta2"
	}
	return string(b)
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

func (d DelegatedInfo) String() string {
	b, err := json.Marshal(d)
	if err != nil {
		return "Failed to marshal DelegatedInfo"
	}
	return string(b)
}

type Beta3 struct {
	Beta3IScore    common.HexInt     `json:"Beta3IScore"`
	DelegationInfo []*DelegationInfo `json:"delegate"`
}

func (b3 Beta3) String() string {
	b, err := json.Marshal(b3)
	if err != nil {
		return "Failed to marshal Beta3"
	}
	return string(b)
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

func (d DelegationInfo) String() string {
	b, err := json.Marshal(d)
	if err != nil {
		return "Failed to marshal DelegationInfo"
	}
	return string(b)
}

func initCalcDebugConfig(ctx *Context, debugConfigPath string) {
	ctx.calcDebugConf = NewCalcDebugConfig()
	debugConfig, err := os.Open(debugConfigPath)
	if err != nil {
		log.Printf("Error while opening calculation debug config file: %s. error : %v"+
			"\nResult file will be store in defaultPath : CalculateResult", debugConfigPath, err)
		return
	}
	defer debugConfig.Close()

	cfgByte, _ := ioutil.ReadAll(debugConfig)
	err = json.Unmarshal(cfgByte, ctx.calcDebugConf)
	if err != nil {
		log.Printf("Error while Unmarshaling config json")
		return
	}
}

func needToUpdateCalcDebugResult(ctx *Context) bool {
	return ctx.calcDebugConf.Flag && len(ctx.calcDebugConf.Addresses) > 0
}

func initDebugResult(ctx *Context, blockHeight uint64, blockHash []byte) {
	if !needToUpdateCalcDebugResult(ctx) {
		return
	}
	ctx.calcDebugResult = new(CalcDebugResult)
	ctx.calcDebugResult.BlockHeight = blockHeight
	if len(blockHash) == 0 {
		for i := 0; i < BlockHashSize; i++ {
			blockHash = append(blockHash, 0)
		}
	}
	ctx.calcDebugResult.BlockHash = "0x" + hex.EncodeToString(blockHash)

	for _, gv := range ctx.GV {
		ctx.calcDebugResult.GV = append(ctx.calcDebugResult.GV, gv)
	}

	for _, prep := range ctx.PRepCandidates {
		ctx.calcDebugResult.Preps = append(ctx.calcDebugResult.Preps, prep)
	}

	for _, address := range ctx.calcDebugConf.Addresses {
		initCalcResult(ctx, *address)
	}
}

func initCalcDebugIScores(ctx *Context, ia IScoreAccount, blockHeight uint64) {
	for _, address := range ctx.calcDebugConf.Addresses {
		if ia.Address.Equal(address) {
			reward := getReward(ctx, ia.Address, blockHeight)
			reward.InitialIScore = ia.IScore
			reward.TotalIScore = ia.IScore
		}
	}
}

func WriteBeta1Info(ctx *Context, produceReward uint64, bp IISSBlockProduceInfo) {
	if !needToUpdateCalcDebugResult(ctx) {
		return
	}
	for _, address := range ctx.calcDebugConf.Addresses {
		writeBeta1Info(ctx, address, produceReward, bp)
	}
}

func writeBeta1Info(ctx *Context, address *common.Address, produceReward uint64, bp IISSBlockProduceInfo) {
	if bp.Generator.Equal(address) {
		reward := getReward(ctx, *address, bp.BlockHeight)
		reward.Beta1.Generate.BlockCount += 1
		reward.Beta1.Generate.IScore += produceReward
		reward.Beta1.Generate.Formula = fmt.Sprintf("%d * %s",
			reward.Beta1.BlockCount, common.NewHexIntFromUint64(produceReward).String())
		iScoreInHex := *common.NewHexIntFromUint64(produceReward)
		reward.Beta1.Beta1IScore.Add(&reward.Beta1.Beta1IScore.Int, &iScoreInHex.Int)
		reward.TotalIScore.Add(&reward.TotalIScore.Int, &iScoreInHex.Int)
		return
	}

	for _, addr := range bp.Validator {
		if addr.Equal(address) {
			reward := getReward(ctx, *address, bp.BlockHeight)
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
			reward.TotalIScore.Add(&reward.TotalIScore.Int, &iScoreInHex.Int)
			return
		}
	}
}

func WriteBeta2Info(ctx *Context, delegationInfo PRepDelegationInfo, prep PRep,
	startBlock uint64, endBlock uint64, prepReward uint64) {
	if !needToUpdateCalcDebugResult(ctx) {
		return
	}
	for _, address := range ctx.calcDebugConf.Addresses {
		writeBeta2Info(ctx, address, delegationInfo, prep, startBlock, endBlock, prepReward)
	}
}

func writeBeta2Info(ctx *Context, address *common.Address, delegationInfo PRepDelegationInfo, prep PRep,
	startBlock uint64, endBlock uint64, prepReward uint64) {
	if delegationInfo.Address.Equal(address) {
		reward := getReward(ctx, *address, endBlock)
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
		reward.TotalIScore.Add(&reward.TotalIScore.Int, &iScoreInHex.Int)
	}
}

func WriteBeta3Info(ctx *Context, rewardAddress common.Address, rewardRep uint64,
	delegationInfo *DelegateData, period uint64, endBlock uint64) {
	if !needToUpdateCalcDebugResult(ctx) {
		return
	}
	for _, address := range ctx.calcDebugConf.Addresses {
		writeBeta3Info(ctx, address, rewardAddress, rewardRep, delegationInfo, period, endBlock)
	}
}

func writeBeta3Info(ctx *Context, debugAddress *common.Address, rewardAddress common.Address, rewardRep uint64,
	delegationInfo *DelegateData, period uint64, endBlock uint64) {
	if rewardAddress.Equal(debugAddress) {
		reward := getReward(ctx, *debugAddress, endBlock)
		iScore := rewardRep * period * delegationInfo.Delegate.Uint64() / rewardDivider
		dgInfo := &DelegationInfo{BlockHeight: endBlock, Address: delegationInfo.Address,
			Amount: delegationInfo.Delegate.Uint64(), IScore: iScore}
		dgInfo.Formula = fmt.Sprintf("%s * %d * %d / %d", common.NewHexIntFromUint64(rewardRep).String(), period,
			dgInfo.Amount, rewardDivider)
		reward.Beta3.DelegationInfo = append(reward.Beta3.DelegationInfo, dgInfo)
		iScoreInHex := *common.NewHexIntFromUint64(iScore)
		reward.Beta3.Beta3IScore.Add(&reward.Beta3.Beta3IScore.Int, &common.NewHexIntFromUint64(iScore).Int)
		reward.TotalIScore.Add(&reward.TotalIScore.Int, &iScoreInHex.Int)
	}
}

func getCalcResult(ctx *Context, address common.Address) *CalcResult {
	for _, calcResult := range ctx.calcDebugResult.Results {
		if calcResult.Address.Equal(&address) {
			return calcResult
		}
	}
	return nil
}

func getReward(ctx *Context, rewardAddress common.Address, blockHeight uint64) *Reward {
	var r *Reward
	calcResult := getCalcResult(ctx, rewardAddress)
	if calcResult == nil {
		initCalcResult(ctx, rewardAddress)
		calcResult = getCalcResult(ctx, rewardAddress)
	}
	rewardLength := len(calcResult.Rewards)
	for i := rewardLength - 1; i >= 0; i-- {
		reward := calcResult.Rewards[i]
		if reward.BlockHeight < blockHeight {
			r = reward
		}
	}
	return r
}

func initCalcResult(ctx *Context, address common.Address) {
	result := NewCalcResult(&address)
	rewards := make([]*Reward, 0)
	for _, gv := range ctx.GV {
		reward := NewReward(gv.BlockHeight)
		rewards = append(rewards, reward)
	}
	result.Rewards = rewards
	ctx.calcDebugResult.Results = append(ctx.calcDebugResult.Results, result)
}

func writeResultToFile(ctx *Context) {
	filePath := ctx.calcDebugConf.Output
	data, err := json.MarshalIndent(ctx.calcDebugResult, "", "  ")
	data = append(data, "\n"...)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error while marshaling calcDebugResult")
		return
	}
	defer f.Close()
	if _, e := f.Write(data); e != nil {
		log.Printf("Error while write calculation debug result")
		return
	}
}

func writeCalcDebugOutput(ctx *Context) {
	bucket, _ := ctx.calcDebugDB.GetBucket("")
	b, err := ctx.calcDebugResult.Bytes()
	if err != nil {
		log.Print("Error while marshaling debugOutput")
		return
	}
	bucket.Set(ctx.calcDebugResult.ID(), b)
}

func addDebuggingAddress(ctx *Context, address common.Address) {
	found := false
	for i := len(ctx.calcDebugConf.Addresses) - 1; i >= 0; i-- {
		if address.Equal(ctx.calcDebugConf.Addresses[i]) {
			found = true
		}
	}
	if !found {
		ctx.calcDebugConf.Addresses = append(ctx.calcDebugConf.Addresses, &address)
	}
}

func deleteDebuggingAddress(ctx *Context, address common.Address) {
	for i := len(ctx.calcDebugConf.Addresses) - 1; i >= 0; i-- {
		if address.Equal(ctx.calcDebugConf.Addresses[i]) {
			ctx.calcDebugConf.Addresses = append(ctx.calcDebugConf.Addresses[:i],
				ctx.calcDebugConf.Addresses[i+1:]...)
		}
	}
	if ctx.calcDebugResult == nil || ctx.calcDebugResult.Results == nil {
		return
	}

	for i, calcResult := range ctx.calcDebugResult.Results {
		if calcResult.Address.Equal(&address) {
			ctx.calcDebugResult.Results =
				append(ctx.calcDebugResult.Results[:i], ctx.calcDebugResult.Results[i+1:]...)
		}
	}
}

func resetCalcResults(ctx *Context) {
	ctx.calcDebugResult.Results = ctx.calcDebugResult.Results[:0]
}
