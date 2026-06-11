package arch

type Identification interface {
	String() string
	Bytes() []byte
	Compare(that Identification) int
}
