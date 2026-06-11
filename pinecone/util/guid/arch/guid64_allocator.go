package arch

type GuidAllocator64 interface {
	NextGUID64() GUID64
	NextGUIDi64() int64
	Explain(guid64 int64) string
}
