package arch

type GUID interface {
	NumericID

	HashCode64() uint64
}
