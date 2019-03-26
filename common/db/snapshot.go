package db

// DB Snapshot
type Snapshot interface {
	New() error
	Get(key []byte) ([]byte, error)
	Release()
}

