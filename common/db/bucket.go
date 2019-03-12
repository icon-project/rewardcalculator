package db

// Bucket
type Bucket interface {
	Get(key []byte) ([]byte, error)
	Has(key []byte) bool
	Set(key []byte, value []byte) error
	Delete(key []byte) error
}

type BucketID string

//	Bucket ID
const (
	// For I-Score DB
	// I-Score
	PrefixIScore BucketID             = ""

	// For global DB

	// DB infomation for management
	PrefixInfo BucketID               = "I"

	// IISS governance variable
	PrefixGovernanceVariable BucketID = "G"

	// P-Rep candidate list
	PrefixPrepCandidate BucketID      = "P"
)

// internalKey returns key prefixed with the bucket's id.
func internalKey(id BucketID, key []byte) []byte {
	buf := make([]byte, len(key)+len(id))
	copy(buf, id)
	copy(buf[len(id):], key)
	return buf
}

// nonNilBytes returns empty []byte if bz is nil
func nonNilBytes(bz []byte) []byte {
	if bz == nil {
		return []byte{}
	}
	return bz
}