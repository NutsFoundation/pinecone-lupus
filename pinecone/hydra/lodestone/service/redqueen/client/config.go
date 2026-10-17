package client

import (
	"time"

	servicearch "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/arch"
)

const defaultTransportVersion = "redqueen-grpc-service-control/1.0"

type Config struct {
	Address               string
	Insecure              bool
	TLS                   servicearch.RPCTLSConfig
	ClientID              int64
	ClientName            string
	ClientGUID            string
	TransportVersion      string
	CommandTimeout        time.Duration
	ControlSyncTimeout    time.Duration
	HeartbeatInterval     time.Duration
	EnableHeartbeat       bool
	AutoReconnect         bool
	InitialConnectTimeout time.Duration
	Logger                servicearch.Logger
}

func (c Config) Normalize() Config {
	if c.TransportVersion == "" {
		c.TransportVersion = defaultTransportVersion
	}
	if c.CommandTimeout <= 0 {
		c.CommandTimeout = 5 * time.Second
	}
	if c.ControlSyncTimeout <= 0 {
		c.ControlSyncTimeout = 5 * time.Second
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = 5 * time.Second
	}
	if c.InitialConnectTimeout <= 0 {
		c.InitialConnectTimeout = 5 * time.Second
	}
	return c
}
