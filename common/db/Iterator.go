package db

// DBIterator
type Iterator interface {
	New([]byte, []byte)
	Next() bool
	Key() []byte
	Value() []byte
	Release()
	Error() error
}

