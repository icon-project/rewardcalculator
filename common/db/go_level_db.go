package db

import (
	"path/filepath"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func init() {
	dbCreator := func(name string, dir string) (Database, error) {
		return NewGoLevelDB(name, dir)
	}
	registerDBCreator(GoLevelDBBackend, dbCreator, false)
}

func NewGoLevelDB(name string, dir string) (*GoLevelDB, error) {
	return NewGoLevelDBWithOpts(name, dir, nil)
}

func NewGoLevelDBWithOpts(name string, dir string, o *opt.Options) (*GoLevelDB, error) {
	dbPath := filepath.Join(dir, name)
	db, err := leveldb.OpenFile(dbPath, o)
	if err != nil {
		return nil, err
	}
	database := &GoLevelDB{
		db: db,
	}
	return database, nil
}

//----------------------------------------
// Database

var _ Database = (*GoLevelDB)(nil)

type GoLevelDB struct {
	db *leveldb.DB
}

func (db *GoLevelDB) GetBucket(id BucketID) (Bucket, error) {
	return &goLevelBucket{
		id: id,
		db: db.db,
	}, nil
}

func (db *GoLevelDB) GetIterator() (Iterator, error) {
	return &goLevelIterator{
		db: db.db,
	}, nil
}

func (db *GoLevelDB) GetBatch() (Batch, error) {
	return &goLevelBatch{
		db: db.db,
	}, nil
}

func (db *GoLevelDB) GetSnapshot() (Snapshot, error) {
	return &goLevelSnapshot{
		db: db.db,
	}, nil
}

func (db *GoLevelDB) Close() error {
	return db.db.Close()
}

//----------------------------------------
// GetBucket

var _ Bucket = (*goLevelBucket)(nil)

type goLevelBucket struct {
	id BucketID
	db *leveldb.DB
}

func (bucket *goLevelBucket) Get(key []byte) ([]byte, error) {
	value, err := bucket.db.Get(internalKey(bucket.id, key), nil)
	if err == leveldb.ErrNotFound {
		return nil, nil
	} else {
		return value, err
	}
}

func (bucket *goLevelBucket) Has(key []byte) bool {
	ret, err := bucket.db.Has(internalKey(bucket.id, key), nil)
	if err != nil {
		return false
	}
	return ret
}

func (bucket *goLevelBucket) Set(key []byte, value []byte) error {
	return bucket.db.Put(internalKey(bucket.id, key), value, nil)
}

func (bucket *goLevelBucket) Delete(key []byte) error {
	return bucket.db.Delete(internalKey(bucket.id, key), nil)
}

//----------------------------------------
// DBIterator

var _ Iterator = (*goLevelIterator)(nil)

type goLevelIterator struct {
	iter iterator.Iterator
	db *leveldb.DB
}

func (i *goLevelIterator) New(start []byte, limit []byte) {
	if start == nil {
		i.iter = i.db.NewIterator(nil, nil)
	} else {
		slice := new(util.Range)
		slice.Start = start
		slice.Limit = limit
		i.iter = i.db.NewIterator(slice, nil)
	}
}

func (i *goLevelIterator) Next() bool {
	return i.iter.Next()
}

func (i *goLevelIterator) Key() []byte {
	return i.iter.Key()
}

func (i *goLevelIterator) Value() []byte {
	return i.iter.Value()
}

func (i *goLevelIterator) Release() {
	i.iter.Release()
}

func (i *goLevelIterator) Error() error {
	return i.iter.Error()
}

//----------------------------------------
// Batch

var _ Batch = (*goLevelBatch)(nil)

type goLevelBatch struct {
	batch *leveldb.Batch
	db *leveldb.DB
}

func (b *goLevelBatch) New() {
	b.batch = new(leveldb.Batch)
}

func (b *goLevelBatch) Len() int {
	return b.batch.Len()
}

func (b *goLevelBatch) Set(key, value []byte) {
	b.batch.Put(key, value)
}

func (b *goLevelBatch) Delete(key []byte) {
	b.batch.Delete(key)
}

func (b *goLevelBatch) Write() error {
	return b.db.Write(b.batch, nil)
}

func (b *goLevelBatch) Reset() {
	b.batch.Reset()
}

//----------------------------------------
// Snapshot

var _ Snapshot = (*goLevelSnapshot)(nil)

type goLevelSnapshot struct {
	snapshot *leveldb.Snapshot
	db *leveldb.DB
}

func (s *goLevelSnapshot) New() error {
	var err error
	s.snapshot, err = s.db.GetSnapshot()
	if err != nil {
		return err
	}
	return nil
}

func (s *goLevelSnapshot) Get(key []byte) ([]byte, error) {
	value, err := s.snapshot.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil, nil
	} else {
		return value, err
	}
}

func (s *goLevelSnapshot) Release() {
	s.snapshot.Release()
}
