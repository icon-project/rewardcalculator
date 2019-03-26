package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
	"strings"
)

func (cli *CLI) prep(blockHeight uint64, generator string, validator string, delete bool) {
	bucket, _ := cli.DB.GetBucket(db.PrefixIISSPRep)

	prep := new(rewardcalculator.IISSPRepStat)
	prep.BlockHeight = blockHeight

	key := prep.ID()
	if delete {
		bucket.Delete(key)
		fmt.Printf("Delete P-Rep %d\n", prep.BlockHeight)
	} else {
		if generator == "" || validator == "" {
			fmt.Printf("Can't add P-Rep statistics. You must input GENERATOR and VLIDATOR Info.\n")
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
		fmt.Printf("Add P-Rep: ID:%v, %s\n", prep.ID(), prep.String())
	}
}
