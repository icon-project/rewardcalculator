package db

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func TestGoLevelDB_Database(t *testing.T) {

	dir, err := ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	testDB := openDatabase(GoLevelDBBackend,"test", dir)
	defer testDB.Close()

	key := []byte("hello")
	value := []byte("world")

	bucket, _ := testDB.GetBucket("hello")
	bucket.Set(key, value)
	result, _ := bucket.Get(key)
	assert.Equal(t, value, result, "equal")
	assert.True(t, bucket.Has(key), "True")

	bucket.Delete(key)
	result, _ = bucket.Get(key)
	assert.Nil(t, result, "empty")
}

func TestGoLevelDB_Iterator(t *testing.T) {
	tests := [] struct {
		key   []byte
		value []byte
	}{
		{key: []byte("key0"), value: []byte("value0")},
		{key: []byte("key1"), value: []byte("value1")},
	}
	dir, err := ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	testDB := openDatabase(GoLevelDBBackend, "test", dir)
	defer testDB.Close()

	bucket, _ := testDB.GetBucket("")
	for _, tt := range tests {
		bucket.Set(tt.key, tt.value)
	}

	iter, _ := testDB.GetIterator()
	prefix := util.BytesPrefix([]byte("key"))
	iter.New(prefix.Start, prefix.Limit)
	for i := 0; iter.Next(); i++ {
		assert.Equal(t, tests[i].key, iter.Key())
		assert.Equal(t, tests[i].value, iter.Value())
	}

	iter.Release()
}

func TestGoLevelDB_Batch(t *testing.T) {
	tests := [] struct {
		key []byte
		value []byte
	}{
		{key: []byte("key0"), value: []byte("value0")},
		{key: []byte("key1"), value: []byte("value1")},
	}
	dir, err := ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	testDB := openDatabase(GoLevelDBBackend, "test", dir)
	defer testDB.Close()

	batch, _ := testDB.GetBatch()
	batch.New()
	for _, tt := range tests {
		batch.Set(tt.key, tt.value)
	}

	assert.Equal(t, len(tests), batch.Len())

	batch.Delete(tests[0].key)
	assert.Equal(t, len(tests) + 1 , batch.Len())

	batch.Write()
	batch.Reset()

	bucket, _ := testDB.GetBucket("")
	assert.False(t, bucket.Has(tests[0].key), "False")
	assert.True(t, bucket.Has(tests[1].key), "True")
}

func TestGoLevelDB_Snapshot(t *testing.T) {
	tests := [] struct {
		key   []byte
		value []byte
	}{
		{key: []byte("key0"), value: []byte("value0")},
		{key: []byte("key1"), value: []byte("value1")},
	}
	dir, err := ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	testDB := openDatabase(GoLevelDBBackend, "test", dir)
	defer testDB.Close()

	bucket, _ := testDB.GetBucket("")
	for _, tt := range tests {
		bucket.Set(tt.key, tt.value)
	}

	snapshot, _ := testDB.GetSnapshot()
	snapshot.New()

	bucket.Set(tests[0].key, []byte("NEW_VALUE"))

	value, _ :=snapshot.Get(tests[0].key)
	assert.Equal(t, tests[0].value, value)

	snapshot.NewIterator(nil, nil)
	for i := 0; snapshot.IterNext(); i++ {
		assert.Equal(t, tests[i].key, snapshot.IterKey())
		assert.Equal(t, tests[i].value, snapshot.IterValue())
	}
	snapshot.ReleaseIterator()
	snapshot.Release()
}
