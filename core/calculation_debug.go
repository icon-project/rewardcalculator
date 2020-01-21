package core

import (
	"encoding/json"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type DebugConfig struct {
	Flag      bool     `json:"enable"`
	Addresses []string `json:"addresses"`
	Output    string   `json:"output"`
}

type DebugResult struct {
	Preps   []*PRepCandidate      `json:"PReps"`
	GV      []*GovernanceVariable `json:"GV"`
	Results []*CalcResult         `json:"calculation"`
}

func (dr DebugResult) String() string {
	b, err := json.Marshal(dr)
	if err != nil {
		return "Failed to marshal DebugResult"
	}
	return string(b)
}

type CalcResult struct {
	Address *common.Address `json:"address"`
	Rewards []*Reward       `json:"rewards"`
}

func NewCalcResult(address *common.Address) *CalcResult {
	return &CalcResult{Address: address, Rewards: make([]*Reward, 0)}
}

func (cr CalcResult) String() string {
	b, err := json.Marshal(cr)
	if err != nil {
		return "Failed to marshal CalcResult"
	}
	return string(b)
}

type Reward struct {
	BlockHeight uint64        `json:"blockHeight"`
	IScore      common.HexInt `json:"IScore"`
	Beta1       *Beta1        `json:"beta1"`
	Beta2       *Beta2        `json:"beta2"`
	Beta3       *Beta3        `json:"beta3"`
}

func NewReward(blockHeight uint64) *Reward {
	return &Reward{blockHeight, *common.NewHexInt(0), NewBeta1(), NewBeta2(), NewBeta3()}
}

func (r Reward) String() string {
	b, err := json.Marshal(r)
	if err != nil {
		return "Failed to marshal Reward"
	}
	return string(b)
}

type Beta1 struct {
	IScore    common.HexInt `json:"IScore"`
	*Generate `json:"generate"`
	Validate  map[string]*ValidateInfo `json:"validate"`
}

func NewBeta1() *Beta1 {
	return &Beta1{*common.NewHexInt(0), NewGenerate(), make(map[string]*ValidateInfo)}
}

func (b1 *Beta1) String() string {
	b, err := json.Marshal(b1)
	if err != nil {
		return "Failed to marshal Beta1"
	}
	return string(b)
}

type Generate struct {
	BlockCount uint64        `json:"blockCount"`
	Formula    string        `json:"formula"`
	IScore     common.HexInt `json:"IScore"`
}

func NewGenerate() *Generate {
	return &Generate{0, "", *common.NewHexInt(0)}
}

func (g Generate) String() string {
	b, err := json.Marshal(g)
	if err != nil {
		return "Failed to marshal Generate"
	}
	return string(b)
}

type ValidateInfo struct {
	BlockCount uint64        `json:"blockCount"`
	Formula    string        `json:"formula"`
	IScore     common.HexInt `json:"IScore"`
}

func (v ValidateInfo) String() string {
	b, err := json.Marshal(v)
	if err != nil {
		return "Failed to marshal ValidateInfo"
	}
	return string(b)
}

type Beta2 struct {
	IScore        common.HexInt    `json:"IScore"`
	DelegatedInfo []*DelegatedInfo `json:"delegated"`
}

func NewBeta2() *Beta2 {
	return &Beta2{*common.NewHexInt(0), make([]*DelegatedInfo, 0)}
}

func (b2 Beta2) String() string {
	b, err := json.Marshal(b2)
	if err != nil {
		return "Failed to marshal Beta2"
	}
	return string(b)
}

type DelegatedInfo struct {
	BlockHeight    uint64        `json:"blockHeight"`
	Delegated      common.HexInt `json:"amount"`
	TotalDelegated common.HexInt `json:"totalDelegated"`
	Formula        string        `json:"formula"`
	IScore         common.HexInt `json:"IScore"`
}

func (d DelegatedInfo) String() string {
	b, err := json.Marshal(d)
	if err != nil {
		return "Failed to marshal DelegatedInfo"
	}
	return string(b)
}

type Beta3 struct {
	IScore         common.HexInt     `json:"IScore"`
	DelegationInfo []*DelegationInfo `json:"delegate"`
}

func NewBeta3() *Beta3 {
	return &Beta3{*common.NewHexInt(0), make([]*DelegationInfo, 0)}
}

func (b3 Beta3) String() string {
	b, err := json.Marshal(b3)
	if err != nil {
		return "Failed to marshal Beta3"
	}
	return string(b)
}

type DelegationInfo struct {
	BlockHeight uint64         `json:"blockHeight"`
	Address     common.Address `json:"address"`
	Amount      uint64         `json:"amount"`
	Formula     string         `json:"formula"`
	IScore      common.HexInt  `json:"IScore"`
}

func (d DelegationInfo) String() string {
	b, err := json.Marshal(d)
	if err != nil {
		return "Failed to marshal DelegationInfo"
	}
	return string(b)
}

func setCalcDebugConfig(ctx *Context, debugConfigPath string) {
	debugConfig, err := os.Open(debugConfigPath)
	if err != nil {
		log.Printf("Error while opening calculation debug config file: %s", debugConfigPath)
		return
	}
	defer debugConfig.Close()

	cfg := new(DebugConfig)
	cfgByte, _ := ioutil.ReadAll(debugConfig)

	err = json.Unmarshal(cfgByte, cfg)
	if err != nil {
		log.Printf("Error while Unmarshaling config json")
		return
	}
	ctx.calculationDebugFlag = cfg.Flag

	addresses := make([]*common.Address, 0)
	for _, address := range cfg.Addresses {
		addresses = append(addresses, common.NewAddressFromString(address))
	}
	ctx.debugCalculationAddresses = addresses
	ctx.debuggingOutputPath = cfg.Output
}

func setCalcDebugInfo(ctx *Context) {
	ctx.debugResult = new(DebugResult)

	for _, gv := range ctx.GV {
		ctx.debugResult.GV = append(ctx.debugResult.GV, gv)
	}

	for _, prep := range ctx.PRepCandidates {
		ctx.debugResult.Preps = append(ctx.debugResult.Preps, prep)
	}

	for i := 0; i < len(ctx.debugCalculationAddresses); i++ {
		addCalcResult(ctx, *ctx.debugCalculationAddresses[i])
	}
}

func WriteBeta1Info(ctx *Context, produceReward common.HexInt, bp IISSBlockProduceInfo,
	bpMap map[common.Address]common.HexInt) {
	for _, address := range ctx.debugCalculationAddresses {
		writeBeta1Info(ctx, address, produceReward, bp, bpMap)
	}
}

func writeBeta1Info(ctx *Context, address *common.Address, produceReward common.HexInt,
	bp IISSBlockProduceInfo, bpMap map[common.Address]common.HexInt) {
	if bp.Generator.Equal(address) {
		reward := getReward(ctx, *address, bp.BlockHeight)
		if reward == nil {
			addCalcResult(ctx, *address)
			reward = getReward(ctx, *address, bp.BlockHeight)
		}
		if reward == nil {
			return
		}
		reward.Beta1.Generate.BlockCount += 1
		reward.Beta1.Generate.IScore = bpMap[*address]
		reward.Beta1.Generate.Formula = fmt.Sprintf("%d * %s", reward.Beta1.BlockCount, produceReward.String())
	}

	for _, addr := range bp.Validator {
		if addr.Equal(address) {
			rewards := getReward(ctx, *address, bp.BlockHeight)

			validatorCount := strconv.FormatUint(uint64(len(bp.Validator)), 64)
			validateInfo := rewards.Beta1.Validate[validatorCount]
			validateInfo.BlockCount += 1
			validateInfo.IScore = bpMap[*address]
			validateInfo.Formula = fmt.Sprintf("%d * %d / %s",
				validateInfo.BlockCount, produceReward.Uint64(), validatorCount)

			rewards.Beta1.Validate[validatorCount] = validateInfo
		}
	}
}

func WriteBeta2Info(ctx *Context, delegationInfo PRepDelegationInfo, prep PRep,
	startBlock uint64, endBlock uint64, iScore common.HexInt, prepReward common.HexInt) {
	for _, address := range ctx.debugCalculationAddresses {
		writeBeta2Info(ctx, address, delegationInfo, prep, startBlock, endBlock, iScore, prepReward)
	}
}

func writeBeta2Info(ctx *Context, address *common.Address, delegationInfo PRepDelegationInfo, prep PRep,
	startBlock uint64, endBlock uint64, iScore common.HexInt, prepReward common.HexInt) {
	if delegationInfo.Address.Equal(address) {
		reward := getReward(ctx, *address, endBlock)
		if reward == nil {
			addCalcResult(ctx, *address)
			reward = getReward(ctx, *address, endBlock)
		}
		if reward == nil {
			return
		}
		delegatedInfo := &DelegatedInfo{BlockHeight: endBlock, TotalDelegated: prep.TotalDelegation,
			Delegated: delegationInfo.DelegatedAmount, IScore: iScore}
		period := endBlock - startBlock
		delegatedInfo.Formula = fmt.Sprintf("%d * %d * %d / %d", prepReward.Uint64(),
			delegationInfo.DelegatedAmount.Uint64(), period, prep.TotalDelegation.Uint64())

		reward.Beta2.DelegatedInfo = append(reward.Beta2.DelegatedInfo, delegatedInfo)
	}
}

func WriteBeta3Info(ctx *Context, rewardAddress common.Address, rewardRep common.HexInt,
	delegationInfo *DelegateData, period common.HexInt, endBlock uint64, iScore common.HexInt) {

	for _, address := range ctx.debugCalculationAddresses {
		writeBeta3Info(ctx, address, rewardAddress, rewardRep, delegationInfo, period, endBlock, iScore)
	}
}

func writeBeta3Info(ctx *Context, debugAddress *common.Address, rewardAddress common.Address, rewardRep common.HexInt,
	delegationInfo *DelegateData, period common.HexInt, endBlock uint64, iScore common.HexInt) {

	if rewardAddress.Equal(debugAddress) {
		reward := getReward(ctx, *debugAddress, endBlock)
		if reward == nil {
			addCalcResult(ctx, *debugAddress)
			reward = getReward(ctx, *debugAddress, endBlock)
		}
		if reward == nil {
			return
		}
		dgInfo := &DelegationInfo{BlockHeight: endBlock, Address: delegationInfo.Address, Amount: delegationInfo.Delegate.Uint64(), IScore: iScore}
		dgInfo.Formula = fmt.Sprintf("%d * %d * %d / %d", rewardRep.Uint64(), period.Uint64(),
			dgInfo.Amount, rewardDivider)
		reward.Beta3.DelegationInfo = append(reward.Beta3.DelegationInfo, dgInfo)
	}
}

func getCalcResult(ctx *Context, address common.Address) *CalcResult {
	for _, calcResult := range ctx.debugResult.Results {
		if calcResult.Address.Equal(&address) {
			return calcResult
		}
	}
	return nil
}

func getReward(ctx *Context, rewardAddress common.Address, blockHeight uint64) *Reward {
	if calcResult := getCalcResult(ctx, rewardAddress); calcResult != nil {
		rewardLength := len(calcResult.Rewards)
		for i := rewardLength - 1; i >= 0; i-- {
			reward := calcResult.Rewards[i]
			if reward.BlockHeight < blockHeight {
				return reward
			}
		}
	}
	return nil
}

func addCalcResult(ctx *Context, address common.Address) {
	result := NewCalcResult(&address)
	rewards := make([]*Reward, 0)
	for _, gv := range ctx.GV {
		reward := NewReward(gv.BlockHeight)
		rewards = append(rewards, reward)
	}
	result.Rewards = rewards
	ctx.debugResult.Results = append(ctx.debugResult.Results, result)
}

func setDebuggingAccountInfo(ctx *Context, ia IScoreAccount, blockHeight uint64) {
	for _, address := range ctx.debugCalculationAddresses {
		if ia.Address.Equal(address) {
			if reward := getReward(ctx, ia.Address, blockHeight); reward != nil {
				reward.IScore = ia.IScore
			}
		}
	}
}

func writeResultToFile(ctx *Context, blockHeight uint64) {
	filePath := fmt.Sprintf("%s_%d.json", ctx.debuggingOutputPath, blockHeight)
	data, err := json.Marshal(ctx.debugResult)
	if err != nil {
		log.Printf("Error while marshaling debugResult")
		return
	}
	if e := ioutil.WriteFile(filePath, data, os.ModePerm); e != nil {
		log.Printf("Error while write calculation debug result")
		return
	}
}

func addDebuggingAddress(ctx *Context, address common.Address) {
	found := false
	for i := len(ctx.debugCalculationAddresses) - 1; i >= 0; i-- {
		if address.Equal(ctx.debugCalculationAddresses[i]) {
			found = true
		}
	}
	if !found {
		ctx.debugCalculationAddresses = append(ctx.debugCalculationAddresses, &address)
	}
	addCalcResult(ctx, address)
}

func deleteDebuggingAddress(ctx *Context, address common.Address) {
	for i := len(ctx.debugCalculationAddresses) - 1; i >= 0; i-- {
		if address.Equal(ctx.debugCalculationAddresses[i]) {
			ctx.debugCalculationAddresses = append(ctx.debugCalculationAddresses[:i],
				ctx.debugCalculationAddresses[i+1:]...)
		}
	}

	for i, calcResult := range ctx.debugResult.Results {
		if calcResult.Address.Equal(&address) {
			ctx.debugResult.Results = append(ctx.debugResult.Results[:i], ctx.debugResult.Results[i+1:]...)
		}
	}
}
