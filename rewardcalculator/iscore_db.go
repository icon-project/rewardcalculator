package rewardcalculator

import (
	"fmt"
	"log"
	"sync"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
)


const (
	NumDelegate              = 10
)

type IconDB interface {
	ID() []byte
	Bytes() []byte
	String() string
	SetBytes([]byte) error
}

type IScoreDB struct {
	// Info. for service
	info *DBInfo

	// DB instance
	Global db.Database
	Account []db.Database
	AccountSnapshot []db.Snapshot
	snapshotLock sync.RWMutex
}

type GlobalOptions struct {
	BlockHeight     uint64	// BlockHeight of confirmed calculate message

	db              *IScoreDB
	PRepCandidates  map[common.Address]*PRepCandidate
	GV              []*GovernanceVariable
}

func (opts *GlobalOptions) GetGVList(start uint64, end uint64) []*GovernanceVariable {
	if len(opts.GV) == 0 {
		return nil
	}

	gvList := make([]*GovernanceVariable, 1, len(opts.GV))

	log.Printf("GV count %d", len(opts.GV))

	for _, v := range opts.GV {
		log.Printf("READ: %s\n", v.String())
		if v.BlockHeight <= start {
			gvList[0] = v
			log.Printf("overwrite\n")
		} else if start <= v.BlockHeight && v.BlockHeight <= end {
			gvList = append(gvList, v)
			log.Printf("append\n")
		} else if v.BlockHeight > end {
			break
		}
	}

	return gvList
}

func (opts *GlobalOptions) Print() {
	log.Printf("============================================================================")
	log.Printf("Global Option\n")
	log.Printf("Block height: %d\n", opts.BlockHeight)
	log.Printf("DB Info.: %s\n", opts.db.info.String())
	log.Printf("Governance Variable: %d\n", len(opts.GV))
	for i, v := range opts.GV {
		log.Printf("\t%d: %s\n", i, v.String())
	}
	log.Printf("P-Rep candidate count : %d\n", len(opts.PRepCandidates))
	log.Printf("============================================================================")
}

func (opts *GlobalOptions) getAccountDBIndex(address common.Address) int {
	prefix := int(address.ID()[0])
	return prefix % opts.db.info.DBCount
}

func (opts *GlobalOptions) GetAccountDB(address common.Address) db.Database {
	return opts.db.Account[opts.getAccountDBIndex(address)]
}

func (opts *GlobalOptions) GetAccountDBSnapshot(address common.Address) db.Snapshot {
	return opts.db.AccountSnapshot[opts.getAccountDBIndex(address)]
}

func (opts *GlobalOptions) SetAccountDBSnapshot() error {
	isDB := opts.db

	isDB.snapshotLock.Lock()
	defer isDB.snapshotLock.Unlock()

	// Init snapshot list
	if isDB.AccountSnapshot == nil {
		isDB.AccountSnapshot = make([]db.Snapshot, 0, len(isDB.Account))
	}

	// make new snapshot list
	snapList := make([]db.Snapshot, len(isDB.Account))
	for i, accountDB := range isDB.Account {
		// get new snapshot
		snapshot, err := accountDB.GetSnapshot()
		if err != nil {
			return err
		}
		snapshot.New()
		snapList[i] = snapshot
	}

	// release snapshot
	for _, snapshot := range isDB.AccountSnapshot {
		snapshot.Release()
	}

	// set snapshot list
	isDB.AccountSnapshot = snapList

	return nil
}

func InitIScoreDB(dbPath string, dbType string, dbName string, worker int) (*GlobalOptions, error) {
	gOpts := new(GlobalOptions)
	isDB := new(IScoreDB)
	gOpts.db = isDB
	var err error

	// Open global DB
	globalDB := db.Open(dbPath, dbType, dbName)
	isDB.Global = globalDB

	// read DB Info.
	isDB.info, err = NewDBInfo(globalDB, worker)
	if err != nil {
		log.Panicf("Failed to load DB Information\n")
		return nil, err
	}

	// read Block Info.
	bInfo, err := NewBlockInfo(globalDB)
	if err != nil {
		log.Panicf("Failed to load Block Information\n")
		return nil, err
	}
	gOpts.BlockHeight = bInfo.BlockHeight

	// read Governance variable
	gOpts.GV, err = LoadGovernanceVariable(globalDB, gOpts.BlockHeight)
	if err != nil {
		log.Printf("Failed to load GV structure\n")
		return nil, err
	}

	// read P-Rep candidate list
	gOpts.PRepCandidates, err = LoadPRepCandidate(globalDB)
	if err != nil {
		log.Printf("Failed to load GV structure\n")
		return nil, err
	}

	// Open account DBs
	isDB.Account = make([]db.Database, isDB.info.DBCount)
	for i := 0; i < isDB.info.DBCount; i++ {
		dbNameTemp := fmt.Sprintf("%d_%d", i + 1, isDB.info.DBCount)
		isDB.Account[i] = db.Open(dbPath + "/" + dbName, dbType, dbNameTemp)
	}

	// make snapshot for query and claim message
	// TODO make snapshot with query message. Do not response with query and claim message until get calculate message
	err = gOpts.SetAccountDBSnapshot()
	if err != nil {
		log.Printf("Failed to get snapshot of account DB. err=%+v\n", err)
		return nil, err
	}

	// TODO find IISS data and load

	return gOpts, nil
}

func CloseIScoreDB(isDB *IScoreDB) {
	// close global DB
	isDB.Global.Close()

	// close account DBs
	for _, aDB := range isDB.Account {
		aDB.Close()
	}
	log.Printf("Close 1 global DB and %d account DBs\n", len(isDB.Account))
}
