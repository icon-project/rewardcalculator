package db

// DB Snapshot
type Snapshot interface {
	New() error
	Get(key []byte) ([]byte, error)
	Release()
	NewIterator([]byte, []byte)
	IterNext() bool
	IterKey() []byte
	IterValue() []byte
	ReleaseIterator()
}

