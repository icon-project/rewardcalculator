package rewardcalculator

import (
	"log"
	"strconv"

	"github.com/icon-project/rewardcalculator/common/db"
)


const (
	KeyDBInfo                = "INFO"
	KeyGlobalOption          = "GO"
)

const (
	NumDelegate              = 10
	NumPRep                  = 22
)

type IconDB interface {
	ID() []byte
	Bytes() []byte
	String() string
	SetBytes([]byte) error
}

type iscoreDB struct {
	db db.Database
}

func writeGovernanceVariable(lvlDB db.Database, gv []byte, blockHeight uint) {
	bucket, _ := lvlDB.GetBucket(db.PrefixGovernanceVariable)

	bucket.Set([]byte(strconv.FormatUint(uint64(blockHeight), 10)), gv)
}

func writeIscoreDBBytes(data []byte) {
}

func InitIscoreDB(dbPath string, dbType string, dbName string, worker int) (db.Database, error) {
	dbi := db.Open(dbPath, dbType, dbName)

	bucket, err := dbi.GetBucket(db.PrefixInfo)
	if err != nil {
		log.Panicf("Failed to get DB Infomation bucket\n")
	}

	dbInfo := new(DBInfo)
	data, err := bucket.Get([]byte(KeyDBInfo))
	if data != nil {
		err = dbInfo.SetBytes(data)
		if err != nil {
			log.Printf("Failed to set DB Infomation structure\n")
		}
	}

	// write DB count if necessary
	if dbInfo.DbCount == 0 {
		dbInfo.DbCount = worker
	} else {
		log.Panicf("Can't run Reward Calculator with %d worker. DB created with %d wokers\n", worker, dbInfo.DbCount)
	}

	log.Printf("Initialize DB. path: %s, type: %s, name: %s, DBInfo: %s\n", dbPath, dbType, dbName, dbInfo.String())

	return dbi, err
}
