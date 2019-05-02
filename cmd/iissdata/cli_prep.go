package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"strconv"
	"strings"
)

func (cli *CLI) bp(blockHeight uint64, generator string, validator string, delete bool) {
	bucket, _ := cli.DB.GetBucket(db.PrefixIISSBPInfo)

	prep := new(core.IISSBlockProduceInfo)
	prep.BlockHeight = blockHeight

	key := prep.ID()
	if delete {
		bucket.Delete(key)
		fmt.Printf("Delete Block produce Info. : %d\n", prep.BlockHeight)
	} else {
		if generator == "" || validator == "" {
			fmt.Printf("Can't add Block product Info.. You must input GENERATOR and VLIDATOR Info.\n")
			return
		}
		validators := strings.Split(validator, ",")

		prep.Generator = *common.NewAddressFromString(generator)
		prep.Validator = make([]common.Address, 0)
		for _, v := range validators {
			prep.Validator = append(prep.Validator, *common.NewAddressFromString(v))
		}

		value, _ := prep.Bytes()
		bucket.Set(key, value)
		fmt.Printf("Add Block produce Info.: ID:%v, %s\n", prep.ID(), prep.String())
	}
}

func (cli *CLI) prep(blockHeight uint64, preps string, delegations string, delete bool) {
	bucket, _ := cli.DB.GetBucket(db.PrefixIISSPRep)

	prep := new(core.PRep)
	prep.BlockHeight = blockHeight

	key := prep.ID()
	if delete {
		bucket.Delete(key)
		fmt.Printf("Delete P-Rep list : %d\n", prep.BlockHeight)
	} else {
		if preps == "" || delegations == "" {
			fmt.Printf("You must input PREPLIST and DELEGATIONLIST\n")
			return
		}
		prepList := strings.Split(preps, ",")
		delegationList := strings.Split(delegations, ",")

		if len(prepList) != len(delegationList) {
			fmt.Printf("PREPLIST and DELEGATIONLIST must have the same number of elements\n")
			return
		}
		var sum uint64
		for _, dg := range delegationList {
			v, _ := strconv.ParseUint(dg, 10, 0)
			sum = sum + v
		}
		prep.TotalDelegation.SetUint64(sum)

		prep.List = make([]core.PRepDelegationInfo, 0)
		for i := 0; i < len(prepList); i++ {
			prepDGInfo := new(core.PRepDelegationInfo)
			prepDGInfo.Address = *common.NewAddressFromString(prepList[i])
			v, _ := strconv.ParseUint(delegationList[i], 10, 0)
			prepDGInfo.DelegatedAmount.SetUint64(v)

			prep.List = append(prep.List, *prepDGInfo)
		}

		value, _ := prep.Bytes()
		bucket.Set(key, value)
		fmt.Printf("Add P-Rep list: ID:%v, %s\n", prep.ID(), prep.String())
	}
}
