package arch

type GuidAllocator interface {
	NextGUID() GUID
	Parse(raw string) (GUID, error)
}
