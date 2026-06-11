package arch

import redprotocol "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/red/protocol"

const (
	DefaultSystemBucket = redprotocol.SystemLabel
	DefaultPartSize     = 10 * 1024 * 1024
)

type Config struct {
	DefaultEndpoint string
	SystemBucket    string
	AccessKey       string
	SecretKey       string
	Secure          bool
	Region          string
	PartSize        uint64
}

func (c Config) AffirmSystemBucket() string {
	if c.SystemBucket == "" {
		return DefaultSystemBucket
	}
	return c.SystemBucket
}

func (c Config) AffirmPartSize() uint64 {
	if c.PartSize == 0 {
		return DefaultPartSize
	}
	return c.PartSize
}
