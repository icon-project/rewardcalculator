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
	// For query and calculation DB
	// I-Score
	PrefixIScore BucketID             = ""

	// For calculation result DB
	PrefixCalcResult BucketID         = ""

	// For claim DB
	PrefixClaim BucketID              = ""

	// For global DB

	// Information for management
	PrefixManagement BucketID         = "MI"

	// Governance variable
	PrefixGovernanceVariable BucketID = "GV"

	// P-Rep candidate list
	PrefixPRepCandidate BucketID      = "PC"

	// Main/Sub P-Rep list
	PrefixPRep BucketID               = "PR"

	// FOR IISS data DB
	// Header
	PrefixIISSHeader BucketID         = "HD"

	// IISS Governance variable
	PrefixIISSGV BucketID             = "GV"

	// Block Producer Info.
	PrefixIISSBPInfo BucketID         = "BP"

	// Main/Sub P-Rep list
	PrefixIISSPRep BucketID           = "PR"

	// TX
	PrefixIISSTX BucketID             = "TX"

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