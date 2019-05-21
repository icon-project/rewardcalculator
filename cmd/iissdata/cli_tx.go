package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
)

func (cli *CLI) transaction(index uint64, address string, blockHeight uint64, dataType uint64,
	dgAddress string, dgAmount uint64) {

	bucket, _ := cli.DB.GetBucket(db.PrefixIISSTX)

	tx := new(core.IISSTX)
	tx.Address = *common.NewAddressFromString(address)
	tx.BlockHeight = blockHeight
	tx.DataType = dataType
	tx.Data = new(codec.TypedObj)
	tx.Index = index

	switch tx.DataType {
	case core.TXDataTypeDelegate:
		var delegation []interface{}
		var dgData []interface{}

		dgData = append(dgData, common.NewAddressFromString(dgAddress))
		dgData = append(dgData, dgAmount)

		delegation = append(delegation, dgData)

		var err error
		tx.Data, err = common.EncodeAny(delegation)
		if err != nil {
			fmt.Printf("Can't encode stake %+v\n", err)
		}
	case core.TXDataTypePRepReg:
		tx.Data.Type = codec.TypeNil
		tx.Data.Object = []byte("")
	case core.TXDataTypePRepUnReg:
		tx.Data.Type = codec.TypeNil
		tx.Data.Object = []byte("")
	}

	key := tx.ID()
	value, _ := tx.Bytes()
	bucket.Set(key, value)

	fmt.Printf("Set transaction %s\n", tx.String())
}
