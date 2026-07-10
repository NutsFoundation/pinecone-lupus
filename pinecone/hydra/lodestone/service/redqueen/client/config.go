package client

import "time"

const defaultTransportVersion = "redqueen-grpc-service-control/1.0"

type Config struct {
	Address                string
	ClientID               int64
	ClientName             string
	ClientGUID             string
	TransportVersion       string
	CommandTimeout         time.Duration
	ControlSyncTimeout     time.Duration
	HeartbeatInterval      time.Duration
	EnableHeartbeat        bool
	AutoSynchronizeOnError bool
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
	return c
}
