package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
)

func (cli *CLI) header(version uint64, blockHeight uint64, revision uint64) {
	bucket, _ := cli.DB.GetBucket(db.PrefixIISSHeader)

	header := new(core.IISSHeader)
	header.Version = version
	header.BlockHeight = blockHeight
	header.Revision = revision

	key := []byte("")
	value, _ := header.Bytes()
	bucket.Set(key, value)

	fmt.Printf("Set header %s\n", header.String())
}
