package module

type Address interface {
	String() string
	Bytes() []byte
	ID() []byte
	IsContract() bool
	Equal(Address) bool
}
