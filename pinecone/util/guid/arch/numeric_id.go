package arch

type NumericID interface {
	Identification

	Int64() int64
	Int() int
	Sizeof() int
	Bitsof() int
	BytesLE() []byte
	BytesBE() []byte
}
