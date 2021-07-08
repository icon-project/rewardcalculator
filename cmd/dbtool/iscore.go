/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/icon-project/rewardcalculator/cmd/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
)

func getIScore(input common.Input) (err error) {
	if input.RcDBRoot == "" {
		fmt.Println("Enter RC DB root path")
		return errors.New("invalid db path")
	}
	rcRoot := input.RcDBRoot
	outputFile := "iscore.json"
	if input.Output != "" {
		outputFile = input.Output
	}

	claimDB := db.Open(input.RcDBRoot, string(db.GoLevelDBBackend), "claim")
	defer claimDB.Close()
	accountDBCount, queryDBSuffix, err := getAccountDBInfo(rcRoot)
	if err != nil {
		return
	}
	pathSlice := make([]string, 0)
	for i := 1; i <= accountDBCount; i++ {
		pathSlice = append(pathSlice, getAccountDBPathWithIndex(rcRoot, i, accountDBCount, queryDBSuffix))
	}

	result := make(map[string]*big.Int)
	prefix := util.BytesPrefix([]byte(db.PrefixIScore))
	for _, accountPath := range pathSlice {
		dir, name := filepath.Split(accountPath)
		accountDB := db.Open(dir, string(db.GoLevelDBBackend), name)
		// iterate
		var iter db.Iterator
		iter, err = accountDB.GetIterator()
		if err != nil {
			fmt.Println("Failed to get iterator")
			return
		}

		var account *core.IScoreAccount
		iter.New(prefix.Start, prefix.Limit)
		for iter.Next() {
			account, err = newIScoreAccount(iter.Key(), iter.Value())
			if err != nil {
				return
			}
			addr := &account.Address
			iScore := new(big.Int).Set(&account.IScore.Int)

			var claim *core.Claim
			if claim, err = getClaim(claimDB, addr); err != nil {
				return errors.Errorf("Error get claim %s", addr.String())
			} else {
				if claim != nil {
					iScore.Sub(iScore, &claim.Data.IScore.Int)
				}
			}
			result[addr.String()] = iScore
		}
		iter.Release()

		err = iter.Error()
		if err != nil {
			return
		}
		accountDB.Close()
	}
	bs, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(outputFile, bs, 0644)
	fmt.Printf("Write %d entries to %s", len(result), outputFile)
	return
}
