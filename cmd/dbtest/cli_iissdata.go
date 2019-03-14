package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (cli *CLI) iissData(dbDir string, dbName string) {
	fmt.Printf("Start check IISS data DB. name: %s/%s\n", dbDir, dbName)

	lvlDB := db.Open(dbDir, DBType, dbName)
	defer lvlDB.Close()

	bucket, _ := lvlDB.GetBucket(db.PrefixIISSHeader)
	data, _ := bucket.Get([]byte(""))
	header := new(rewardcalculator.IISSHeader)
	err := header.SetBytes(data)
	if err != nil {
		fmt.Printf("Failed to read header from IISS Data. err=%+v\n", err)
		return
	}
	fmt.Printf("Header: %s\n", header.String())

	bucket, _ = lvlDB.GetBucket(db.PrefixIISSGV)
	data, _ = bucket.Get([]byte(""))
	gv := new(rewardcalculator.IISSGovernanceVariable)
	err = gv.SetBytes(data)
	if err != nil {
		fmt.Printf("Failed to read governance variable from IISS Data. err=%+v\n", err)
		return
	}
	fmt.Printf("Governance variable: %s\n", gv.String())

	iter, _ := lvlDB.GetIterator()
	iter.New([]byte(db.PrefixIISSPRep), nil)
	for entries := 0; iter.Next(); entries++ {
		prep := new(rewardcalculator.IISSPRep)
		err = prep.SetBytes(iter.Value())
		if err != nil {
			fmt.Printf("Failed to read P-Rep list from IISS Data. err=%+v\n", err)
			return
		}
		addr := iter.Key()[len(db.PrefixIISSPRep):]
		prep.Address = common.NewAddress(addr)
		fmt.Printf("P-Rep %d: %s\n", entries, prep.String())
	}
	iter.Release()

	iter, _ = lvlDB.GetIterator()
	prefix := util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)
	for entries := 0; iter.Next(); entries++ {
		tx := new(rewardcalculator.IISSTX)
		err = tx.SetBytes(iter.Value())
		if err != nil {
			fmt.Printf("Failed to read TX list from IISS Data. err=%+v\n", err)
			return
		}
		tx.TXHash = iter.Key()[len(db.PrefixIISSTX):]
		fmt.Printf("TX %d: %s\n", entries, tx.String())
		//data, _ := common.DecodeAny(tx.Data)
		//fmt.Printf("print tx.data: %+v, data: %+v ", tx.Data, data)
		//switch tx.DataType {
		//case rewardcalculator.TXDataTypeStake:
		//	stake, _ := data.(*common.HexInt)
		//	fmt.Printf("stake: %s\n", stake.String())
		//case rewardcalculator.TXDataTypeDelegate:
		//	dg := new(rewardcalculator.IISSTXDataDelegation)
		//	dg.FromTypedObj(tx.Data)
		//	fmt.Printf("delegation: %+v\n", dg.Delegation)
		//case rewardcalculator.TXDataTypeClaim:
		//	fmt.Println("")
		//case rewardcalculator.TXDataTypePrepReg:
		//	fmt.Println("")
		//case rewardcalculator.TXDataTypePrepUnReg:
		//	fmt.Println("")
		//}
	}
	iter.Release()
}
