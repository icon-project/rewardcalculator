package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
)

func (cli *CLI) governanceVariable(blockHeight uint64, price uint64, incentive uint64) {
	fmt.Printf("Start set header of IISS data DB.\n")

	bucket, _ := cli.DB.GetBucket(db.PrefixIISSGV)

	gv := new(rewardcalculator.IISSGovernanceVariable)
	gv.BlockHeight = blockHeight
	gv.IcxPrice = price
	gv.IncentiveRep = incentive

	value, _ := gv.Bytes()
	bucket.Set(gv.ID(), value)

	fmt.Printf("Add governance variable: ID: %+v, %s\n", gv.ID(), gv.String())
}
