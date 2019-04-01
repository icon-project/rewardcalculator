package db

// DB Snapshot
type Snapshot interface {
	New() error
	Get(key []byte) ([]byte, error)
	NewIterator([]byte, []byte)
	IterNext() bool
	IterKey() []byte
	IterValue() []byte
	ReleaseIterator()
	Release()
}

