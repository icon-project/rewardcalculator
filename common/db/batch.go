package db

// DBIterator
type Batch interface {
	New()
	Len() int
	Set(key, value []byte)
	Delete(key []byte)
	Write() error
	Reset()
}

